// Package pdf renders print-ready QR label sheets.
//
// QR modules are drawn as vector rectangles — no rasterization — so output is
// crisp at any print size and each code costs only a few hundred bytes.
// Memory stays flat regardless of count: codes are encoded one at a time and
// appended to the document page by page.
package pdf

import (
	"fmt"
	"io"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// LabelSpec describes one label cell on the sheet.
type LabelSpec struct {
	SizeMM   float64 // QR side length in mm
	ShowText bool    // print the code string under the QR
}

// LabelPresets are the label sizes offered in the UI.
var LabelPresets = map[string]LabelSpec{
	"25": {SizeMM: 25, ShowText: false},
	"38": {SizeMM: 38, ShowText: true},
	"50": {SizeMM: 50, ShowText: true},
}

// Code is one QR to render.
type Code struct {
	Content string // the URL encoded into the QR
	Label   string // human-readable code printed under the QR (optional)
}

const (
	pageMarginMM = 10.0
	cellGapMM    = 4.0
	textHeightMM = 4.0
	// quiet zone around the QR inside its cell, in modules
	quietModules = 2
)

// Generator renders codes into an A4 sheet grid.
type Generator struct {
	pdf      *fpdf.Fpdf
	spec     LabelSpec
	cols     int
	rows     int
	cellW    float64
	cellH    float64
	perPage  int
	count    int
	pageW    float64
	pageH    float64
}

// NewGenerator prepares an A4 portrait document for the given label spec.
func NewGenerator(spec LabelSpec) *Generator {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetAutoPageBreak(false, 0)
	doc.SetFont("Helvetica", "", 6)

	pageW, pageH := doc.GetPageSize()
	usableW := pageW - 2*pageMarginMM
	usableH := pageH - 2*pageMarginMM

	cellW := spec.SizeMM + cellGapMM
	cellH := spec.SizeMM + cellGapMM
	if spec.ShowText {
		cellH += textHeightMM
	}

	cols := int(usableW / cellW)
	rows := int(usableH / cellH)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	return &Generator{
		pdf:     doc,
		spec:    spec,
		cols:    cols,
		rows:    rows,
		cellW:   cellW,
		cellH:   cellH,
		perPage: cols * rows,
		pageW:   pageW,
		pageH:   pageH,
	}
}

// PerPage returns how many labels fit on one page.
func (g *Generator) PerPage() int { return g.perPage }

// Add renders one code into the next grid cell.
func (g *Generator) Add(code Code) error {
	idx := g.count % g.perPage
	if idx == 0 {
		g.pdf.AddPage()
	}
	g.count++

	col := idx % g.cols
	row := idx / g.cols
	x := pageMarginMM + float64(col)*g.cellW
	y := pageMarginMM + float64(row)*g.cellH

	q, err := qrcode.New(code.Content, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("encode %q: %w", code.Label, err)
	}
	bitmap := q.Bitmap() // includes a 4-module quiet border

	modules := len(bitmap)
	moduleSize := g.spec.SizeMM / float64(modules-2*(4-quietModules))
	// Draw only the payload area plus our chosen quiet zone; skip4-quiet excess.
	offset := 4 - quietModules

	g.pdf.SetFillColor(0, 0, 0)
	for r := offset; r < modules-offset; r++ {
		for c := offset; c < modules-offset; c++ {
			if bitmap[r][c] {
				g.pdf.Rect(
					x+float64(c-offset)*moduleSize,
					y+float64(r-offset)*moduleSize,
					moduleSize, moduleSize, "F")
			}
		}
	}

	if g.spec.ShowText && code.Label != "" {
		g.pdf.SetXY(x, y+g.spec.SizeMM+0.8)
		g.pdf.CellFormat(g.spec.SizeMM, textHeightMM-1, truncateLabel(code.Label, 28), "", 0, "C", false, 0, "")
	}
	return nil
}

// Output writes the finished document.
func (g *Generator) Output(w io.Writer) error {
	if g.count == 0 {
		g.pdf.AddPage()
		g.pdf.SetXY(pageMarginMM, pageMarginMM)
		g.pdf.CellFormat(0, 10, "No codes to render", "", 0, "L", false, 0, "")
	}
	return g.pdf.Output(w)
}

func truncateLabel(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
