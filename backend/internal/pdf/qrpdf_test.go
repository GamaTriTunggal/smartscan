package pdf

import (
	"bytes"
	"fmt"
	"testing"
)

func TestGeneratorProducesValidPDF(t *testing.T) {
	for name, spec := range LabelPresets {
		t.Run("label-"+name, func(t *testing.T) {
			gen := NewGenerator(spec)
			if gen.PerPage() < 1 {
				t.Fatalf("perPage must be >= 1, got %d", gen.PerPage())
			}
			// Render enough codes to span multiple pages.
			n := gen.PerPage()*2 + 3
			for i := 0; i < n; i++ {
				err := gen.Add(Code{
					Content: fmt.Sprintf("https://example.com/s/AbCdEf%06d", i),
					Label:   fmt.Sprintf("CODE%06d", i),
				})
				if err != nil {
					t.Fatalf("add code %d: %v", i, err)
				}
			}
			var buf bytes.Buffer
			if err := gen.Output(&buf); err != nil {
				t.Fatalf("output: %v", err)
			}
			if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
				t.Fatalf("output is not a PDF (got %q)", buf.Bytes()[:8])
			}
			if buf.Len() < 1000 {
				t.Fatalf("suspiciously small PDF: %d bytes", buf.Len())
			}
			t.Logf("%d codes → %d bytes (%.0f B/code)", n, buf.Len(), float64(buf.Len())/float64(n))
		})
	}
}

func TestGeneratorEmptyOutput(t *testing.T) {
	gen := NewGenerator(LabelPresets["25"])
	var buf bytes.Buffer
	if err := gen.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("empty generator must still produce a valid PDF")
	}
}
