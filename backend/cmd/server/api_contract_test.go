package main

// Route-contract test: fails whenever the frontend calls a backend endpoint
// that is not registered on the router (the "dead endpoint" bug class, e.g.
// the SettingsPage reconciliation 404s and the removed cert-request
// endpoints that shipped in earlier releases).
//
// How it works:
//  1. Builds the real router via setupRouter(nil, testConfig) — GORM is only
//     touched at request time, so a nil *gorm.DB is safe for enumeration —
//     and collects every registered (method, path) under /api/v1.
//  2. Walks frontend/src (.vue/.js, excluding __tests__ and *.spec.js) and
//     extracts API call sites via the repo's known calling patterns:
//       - get/post/put/patch/del/upload('/x') and api.get(...)/axios.get(...)
//       - the same methods called with a variable resolved from a nearby
//         `const url = '/x...'` / `const endpoint = ...` declaration
//       - fetch(`${apiBase}/x`), fetch(url), fetch(`${VITE_API_URL}${endpoint}`)
//  3. Normalizes each extracted path (strips the API base prefix and query
//     string, turns ${...} interpolations into :param) and matches it
//     segment-by-segment against the registered routes.
//  4. Fails with file:line for every frontend call that no backend route can
//     serve. A minimum-extraction guard makes sure a silent regex regression
//     cannot pass vacuously.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/gamatritunggal/smartscan/backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// Guard-the-guard thresholds. If extraction ever collapses below these, the
// test fails loudly instead of passing vacuously. Current real counts are
// ~207 call sites / ~165 distinct endpoints (see t.Log output), so these
// trip only on a genuine extractor regression, not on normal code churn.
// ---------------------------------------------------------------------------
const (
	minCallSites        = 80 // distinct file:line extraction records
	minDistinctPaths    = 60 // distinct (method, normalized path) pairs
	minRegisteredRoutes = 50 // registered /api/v1 routes (router sanity check)
)

// allowedMissing lists frontend calls that are intentionally not served by
// this backend (each entry needs a comment explaining why). Keep it empty
// unless there is a genuinely intentional exception — a dead endpoint must
// be fixed, not whitelisted.
var allowedMissing = map[string]bool{
	// (none)
}

// apiCall is one extracted frontend call site.
type apiCall struct {
	File   string // path relative to frontend/src
	Line   int
	Method string
	Path   string // normalized: /tenant/products/:param
}

// ---------------------------------------------------------------------------
// Normalization
// ---------------------------------------------------------------------------

// basePrefixes are template-literal prefixes that resolve to the API base URL
// (axios baseURL / VITE_API_URL / '/api/v1'), i.e. everything after them is a
// backend route path.
var basePrefixes = []string{
	"${import.meta.env.VITE_API_URL}",
	"${apiBase}",
	"${apiUrl}",
	"${API_URL}",
	"${API_BASE}",
}

var interpRe = regexp.MustCompile(`\$\{[^}]*\}`)

const paramMarker = "\x00PARAM\x00"

// normalizeCallPath turns a raw string/template literal from a frontend call
// site into a normalized API path. ok=false means the literal is not an API
// call (external URL, relative fragment, unrelated string, ...).
func normalizeCallPath(raw string) (string, bool) {
	s := strings.TrimSpace(raw)

	// Strip a recognized API-base interpolation prefix.
	for _, p := range basePrefixes {
		if strings.HasPrefix(s, p) {
			s = s[len(p):]
			break
		}
	}
	// Tour helpers hardcode '/api/v1' as their base.
	s = strings.TrimPrefix(s, "/api/v1")

	// Must be an absolute API path now. This rejects external URLs
	// (https://...), lone query fragments, storage keys, etc.
	if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") || strings.Contains(s, "://") {
		return "", false
	}

	// Replace complete ${...} interpolations before stripping the query
	// string: a '?' inside an interpolation is not a query separator.
	s = interpRe.ReplaceAllString(s, paramMarker)

	// Strip query string / fragment.
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	// A dangling unterminated "${" means the literal was truncated inside an
	// interpolation; cut there.
	if i := strings.Index(s, "${"); i >= 0 {
		s = s[:i]
	}

	// Canonicalize segments: any segment touched by an interpolation becomes
	// ":param" (it can hold anything at runtime).
	rawSegs := strings.Split(strings.Trim(s, "/"), "/")
	segs := make([]string, 0, len(rawSegs))
	for _, seg := range rawSegs {
		if seg == "" {
			continue
		}
		if strings.Contains(seg, paramMarker) {
			segs = append(segs, ":param")
		} else {
			segs = append(segs, seg)
		}
	}
	if len(segs) == 0 {
		return "", false
	}
	return "/" + strings.Join(segs, "/"), true
}

// ---------------------------------------------------------------------------
// Matching
// ---------------------------------------------------------------------------

// segmentsMatch reports whether a normalized frontend path can be served by a
// gin route pattern, segment by segment:
//   - literal == literal
//   - frontend :param matches any backend segment (gin :param OR a literal —
//     the interpolated value could be that literal at runtime)
//   - backend :param matches any frontend segment
//   - backend *wildcard consumes the rest
//   - otherwise depth must match exactly
func segmentsMatch(feSegs, beSegs []string) bool {
	i := 0
	for ; i < len(beSegs); i++ {
		be := beSegs[i]
		if strings.HasPrefix(be, "*") {
			return true // wildcard consumes the remainder (even if empty)
		}
		if i >= len(feSegs) {
			return false
		}
		fe := feSegs[i]
		if strings.HasPrefix(be, ":") || fe == ":param" || fe == be {
			continue
		}
		return false
	}
	return i == len(feSegs)
}

func splitPath(p string) []string {
	trimmed := strings.Trim(p, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// ---------------------------------------------------------------------------
// Extraction
// ---------------------------------------------------------------------------

// Quote alternation shared by the extraction regexes. Go's RE2 has no
// backreferences, so each quote style gets its own capture group.
const quotedLit = `(?:'([^']*)'|"([^"]*)"` + "|`([^`]*)`)"

var (
	// get('/x'), api.get(`/x/${id}`), axios.post(`${apiBase}/x`), del('/x'), upload('/x')
	callLitRe = regexp.MustCompile(`\b(get|post|put|patch|delete|del|upload)\s*\(\s*` + quotedLit)
	// get(url), post(endpoint, payload) — first arg is a plain identifier
	callVarRe = regexp.MustCompile(`\b(get|post|put|patch|delete|del|upload)\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*[,)]`)
	// fetch(`...`) / fetch(url, {...})
	fetchRe = regexp.MustCompile(`\bfetch\s*\(\s*(?:` + quotedLit + `|([A-Za-z_$][A-Za-z0-9_$]*)\s*[,)])`)
	// const url = '/x...'  — used to resolve variable-based call sites
	assignRe = regexp.MustCompile(`\b(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*=\s*` + quotedLit)
	// method: 'POST' inside fetch options
	fetchMethodRe = regexp.MustCompile(`method:\s*['"]([A-Za-z]+)['"]`)
	// a template literal that STARTS with a simple local variable, e.g.
	// `${url}&page=${pageNum}` — the variable is resolved from a nearby
	// declaration. (Dotted expressions like ${import.meta.env.X} don't match.)
	leadingVarRe = regexp.MustCompile(`^\$\{([A-Za-z_$][A-Za-z0-9_$]*)\}`)
)

var methodMap = map[string]string{
	"get":    "GET",
	"post":   "POST",
	"put":    "PUT",
	"patch":  "PATCH",
	"delete": "DELETE",
	"del":    "DELETE",
	"upload": "POST", // useAPI's upload() is api.post with multipart headers
}

// pickQuoted returns the matched literal from the three quote-style capture
// groups (their submatch indices start at `base`).
func pickQuoted(src string, idx []int, base int) (string, bool) {
	for g := base; g < base+3; g++ {
		if idx[2*g] >= 0 {
			return src[idx[2*g]:idx[2*g+1]], true
		}
	}
	return "", false
}

type assignment struct {
	offset  int
	literal string
}

// indexAssignments maps identifier -> ordered list of string-literal
// declarations in the file.
func indexAssignments(src string) map[string][]assignment {
	out := map[string][]assignment{}
	for _, idx := range assignRe.FindAllStringSubmatchIndex(src, -1) {
		name := src[idx[2]:idx[3]]
		if lit, ok := pickQuoted(src, idx, 2); ok {
			out[name] = append(out[name], assignment{offset: idx[0], literal: lit})
		}
	}
	return out
}

// resolveVar returns the literal of the nearest declaration of name that
// precedes offset (the standard shape: `const endpoint = ...` a line or two
// above its use).
func resolveVar(assigns map[string][]assignment, name string, offset int) (string, bool) {
	best := ""
	found := false
	for _, a := range assigns[name] {
		if a.offset < offset {
			best = a.literal
			found = true
		} else {
			break
		}
	}
	return best, found
}

func lineOf(lineStarts []int, offset int) int {
	// binary search: number of line starts <= offset
	lo, hi := 0, len(lineStarts)
	for lo < hi {
		mid := (lo + hi) / 2
		if lineStarts[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo // 1-indexed because lineStarts[0] == 0
}

func buildLineIndex(src string) []int {
	starts := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// fetchMethod finds `method: 'X'` in the option object following a fetch call.
func fetchMethod(src string, from int) string {
	end := from + 400
	if end > len(src) {
		end = len(src)
	}
	if m := fetchMethodRe.FindStringSubmatch(src[from:end]); m != nil {
		return strings.ToUpper(m[1])
	}
	return "GET"
}

// extractCallsFromFile pulls every resolvable API call site out of one
// frontend source file.
func extractCallsFromFile(relPath, src string) []apiCall {
	lineStarts := buildLineIndex(src)
	assigns := indexAssignments(src)
	var calls []apiCall

	// resolveLeading substitutes a leading ${var} whose declaration literal is
	// known, e.g. get(`${url}&page=${p}`) with `let url = '/x?limit=100'`
	// two lines above. Bounded to avoid cycles.
	resolveLeading := func(lit string, offset int) string {
		for range 3 {
			m := leadingVarRe.FindStringSubmatch(lit)
			if m == nil {
				return lit
			}
			inner, ok := resolveVar(assigns, m[1], offset)
			if !ok {
				return lit
			}
			lit = inner + lit[len(m[0]):]
		}
		return lit
	}

	add := func(offset int, method, rawPath string) {
		path, ok := normalizeCallPath(resolveLeading(rawPath, offset))
		if !ok {
			return
		}
		calls = append(calls, apiCall{
			File:   relPath,
			Line:   lineOf(lineStarts, offset),
			Method: method,
			Path:   path,
		})
	}

	// Pattern 1: method call with a string-literal first argument.
	for _, idx := range callLitRe.FindAllStringSubmatchIndex(src, -1) {
		method := methodMap[src[idx[2]:idx[3]]]
		if lit, ok := pickQuoted(src, idx, 2); ok {
			add(idx[0], method, lit)
		}
	}

	// Pattern 2: method call with an identifier resolved from a nearby
	// `const url = ...` / `let endpoint = ...` declaration.
	for _, idx := range callVarRe.FindAllStringSubmatchIndex(src, -1) {
		method := methodMap[src[idx[2]:idx[3]]]
		name := src[idx[4]:idx[5]]
		if lit, ok := resolveVar(assigns, name, idx[0]); ok {
			add(idx[0], method, lit)
		}
	}

	// Pattern 3: fetch() with a literal or an identifier argument.
	for _, idx := range fetchRe.FindAllStringSubmatchIndex(src, -1) {
		method := fetchMethod(src, idx[1])
		lit, ok := pickQuoted(src, idx, 1)
		if !ok {
			// fetch(url, ...) — identifier argument
			if idx[8] >= 0 {
				lit, ok = resolveVar(assigns, src[idx[8]:idx[9]], idx[0])
			}
			if !ok {
				continue
			}
		}
		// fetch(`${VITE_API_URL}${endpoint}`): after stripping the base the
		// whole path lives in one variable — resolve it.
		for _, p := range basePrefixes {
			if rest := strings.TrimPrefix(lit, p); rest != lit {
				if m := regexp.MustCompile(`^\$\{([A-Za-z_$][A-Za-z0-9_$]*)\}$`).FindStringSubmatch(rest); m != nil {
					if inner, ok2 := resolveVar(assigns, m[1], idx[0]); ok2 {
						lit = inner
					}
				}
				break
			}
		}
		add(idx[0], method, lit)
	}

	return calls
}

// collectFrontendCalls walks frontend/src and extracts all API call sites.
func collectFrontendCalls(t *testing.T, frontendSrc string) []apiCall {
	t.Helper()
	var calls []apiCall
	err := filepath.WalkDir(frontendSrc, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "__tests__" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".vue" && ext != ".js" {
			return nil
		}
		if strings.HasSuffix(path, ".spec.js") || strings.HasSuffix(path, ".test.js") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(frontendSrc, path)
		calls = append(calls, extractCallsFromFile(rel, string(data))...)
		return nil
	})
	if err != nil {
		t.Fatalf("walking frontend source: %v", err)
	}

	// De-duplicate identical extraction records (a line can be matched by
	// more than one pattern).
	seen := map[apiCall]bool{}
	uniq := calls[:0]
	for _, c := range calls {
		if !seen[c] {
			seen[c] = true
			uniq = append(uniq, c)
		}
	}
	return uniq
}

// ---------------------------------------------------------------------------
// The contract test
// ---------------------------------------------------------------------------

func frontendSrcDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file location")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "frontend", "src")
}

func TestFrontendBackendRouteContract(t *testing.T) {
	frontendSrc := frontendSrcDir(t)
	if _, err := os.Stat(frontendSrc); err != nil {
		t.Skipf("frontend source not present at %s (backend-only checkout): %v", frontendSrc, err)
	}

	// Build the real router. GORM is only dereferenced at request time, so a
	// nil DB is fine for route enumeration.
	gin.SetMode(gin.TestMode)
	router := setupRouter(nil, testutil.TestConfig())

	const apiPrefix = "/api/v1"
	type beRoute struct {
		method string
		path   string
		segs   []string
	}
	var routes []beRoute
	for _, r := range router.Routes() {
		if r.Path != apiPrefix && !strings.HasPrefix(r.Path, apiPrefix+"/") {
			continue // /health, /metrics, /s/:code, /uploads/* are not called via the API base
		}
		p := strings.TrimPrefix(r.Path, apiPrefix)
		if p == "" {
			p = "/"
		}
		routes = append(routes, beRoute{method: r.Method, path: p, segs: splitPath(p)})
	}
	if len(routes) < minRegisteredRoutes {
		t.Fatalf("router sanity check failed: only %d routes registered under %s (expected >= %d) — did route registration move?",
			len(routes), apiPrefix, minRegisteredRoutes)
	}

	calls := collectFrontendCalls(t, frontendSrc)

	// Guard the guard: a silent extractor regression must not pass vacuously.
	distinct := map[string]bool{}
	for _, c := range calls {
		distinct[c.Method+" "+c.Path] = true
	}
	if len(calls) < minCallSites || len(distinct) < minDistinctPaths {
		t.Fatalf("extraction guard tripped: found %d call sites / %d distinct endpoints (need >= %d / >= %d) — the call-site extractor has likely regressed",
			len(calls), len(distinct), minCallSites, minDistinctPaths)
	}

	// Match every frontend call against the registered routes.
	matchedRoutes := map[string]bool{}
	var missing []apiCall
	for _, c := range calls {
		if allowedMissing[c.Method+" "+c.Path] {
			continue
		}
		feSegs := splitPath(c.Path)
		found := false
		for _, r := range routes {
			if r.method == c.Method && segmentsMatch(feSegs, r.segs) {
				matchedRoutes[r.method+" "+r.path] = true
				found = true
			}
		}
		if !found {
			missing = append(missing, c)
		}
	}

	matchedCalls := len(calls) - len(missing)
	t.Logf("frontend call sites extracted: %d (%d distinct endpoints); matched: %d; backend routes under %s: %d",
		len(calls), len(distinct), matchedCalls, apiPrefix, len(routes))

	// Informational only: backend routes never referenced by the frontend
	// (webhooks, exports triggered elsewhere, admin tooling are legitimate).
	var unreferenced []string
	for _, r := range routes {
		if !matchedRoutes[r.method+" "+r.path] {
			unreferenced = append(unreferenced, r.method+" "+r.path)
		}
	}
	sort.Strings(unreferenced)
	t.Logf("backend routes never referenced by the frontend (informational): %d", len(unreferenced))
	for _, r := range unreferenced {
		t.Logf("  uncalled: %s", r)
	}

	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool {
			if missing[i].File != missing[j].File {
				return missing[i].File < missing[j].File
			}
			return missing[i].Line < missing[j].Line
		})
		var b strings.Builder
		fmt.Fprintf(&b, "%d frontend call(s) target endpoints that DO NOT EXIST on the backend:\n", len(missing))
		for _, c := range missing {
			fmt.Fprintf(&b, "  %s:%d  %s %s\n", c.File, c.Line, c.Method, c.Path)
		}
		b.WriteString("Fix the frontend call or register the route in backend/cmd/server/router.go.\n")
		b.WriteString("(If the call is genuinely served elsewhere, add it to allowedMissing with a comment.)")
		t.Fatal(b.String())
	}
}

// ---------------------------------------------------------------------------
// Unit tests for the normalize / match helpers
// ---------------------------------------------------------------------------

func TestNormalizeCallPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"plain path", "/tenant/products", "/tenant/products", true},
		{"query stripped", "/tenant/analytics?from=${a}&to=${b}", "/tenant/analytics", true},
		{"interpolation to param", "/tenant/products/${id}/images", "/tenant/products/:param/images", true},
		{"two interpolations", "/tenant/qr-batches/${batchId.value}/codes/${codeId.value}", "/tenant/qr-batches/:param/codes/:param", true},
		{"apiBase prefix", "${apiBase}/public/warranty/${uuid.value}", "/public/warranty/:param", true},
		{"API_URL prefix", "${API_URL}/auth/refresh", "/auth/refresh", true},
		{"vite env prefix", "${import.meta.env.VITE_API_URL}/tenant/staff", "/tenant/staff", true},
		{"hardcoded api v1 base", "/api/v1/tenant/templates?type=validation", "/tenant/templates", true},
		{"interp containing query char", "${apiBase}/tenant/geofence/violations/export${qs ? '?' + qs : ''}", "/tenant/geofence/violations/export/:param", false /* see below */},
		{"trailing slash", "/tenant/products/", "/tenant/products", true},
		{"external url rejected", "https://nominatim.openstreetmap.org/search?q=x", "", false},
		{"unknown variable prefix rejected", "${url}&page=2", "", false},
		{"relative literal rejected", "reason", "", false},
		{"lone base rejected", "${apiBase}", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := normalizeCallPath(tc.in)
			if tc.name == "interp containing query char" {
				// The optional-query suffix interpolation glues onto the last
				// segment; the segment must degrade to :param, never to a
				// mangled literal.
				if !ok || got != "/tenant/geofence/violations/:param" {
					t.Fatalf("got (%q, %v), want (/tenant/geofence/violations/:param, true)", got, ok)
				}
				return
			}
			if ok != tc.ok || (ok && got != tc.want) {
				t.Fatalf("normalizeCallPath(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestSegmentsMatch(t *testing.T) {
	cases := []struct {
		name string
		fe   string
		be   string
		want bool
	}{
		{"exact literal", "/tenant/products", "/tenant/products", true},
		{"fe param vs gin param", "/tenant/products/:param", "/tenant/products/:id", true},
		{"fe literal vs gin param", "/tenant/products/123", "/tenant/products/:id", true},
		{"fe param vs gin literal", "/tenant/qr-batches/:param/export/:param", "/tenant/qr-batches/:id/export/csv", true},
		{"depth mismatch short", "/tenant/products", "/tenant/products/:id", false},
		{"depth mismatch long", "/tenant/products/:param/images/extra", "/tenant/products/:id/images", false},
		{"literal mismatch", "/tenant/warranties/export", "/tenant/warranties/import", false},
		{"wildcard consumes rest", "/uploads/backgrounds/a/b/c.png", "/uploads/backgrounds/*filepath", true},
		{"wildcard empty rest", "/uploads/backgrounds", "/uploads/backgrounds/*filepath", true},
		{"prefix only is not a match", "/tenant", "/tenant/products", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := segmentsMatch(splitPath(tc.fe), splitPath(tc.be))
			if got != tc.want {
				t.Fatalf("segmentsMatch(%q, %q) = %v, want %v", tc.fe, tc.be, got, tc.want)
			}
		})
	}
}
