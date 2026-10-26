// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"bytes"
	"embed"
	"flag"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var updateGolden = flag.Bool("update-golden", false, "rewrite testdata/golden/*.svg from the code")

//go:embed testdata/golden
var goldenFS embed.FS

// goldenFigures are drawn and compared with testdata/golden/<name>.svg.
var goldenFigures = map[string]func(t *testing.T) *Figure{
	"pie_basic":  func(t *testing.T) *Figure { f, _ := newPie(t, "basic"); return f },
	"pie_donut":  func(t *testing.T) *Figure { f, _ := newPie(t, "donut"); return f },
	"pie_legend": func(t *testing.T) *Figure { f, _ := newPie(t, "legend"); return f },
	"pie_spread": func(t *testing.T) *Figure {
		f := NewFigure(FigSize(4, 4), DPI(50))
		a := f.Subplots(1, 1)[0][0]
		if err := a.Pie([]float64{1000, 1, 1, 1, 500}, Labels("Large", "tiny one", "tiny two", "tiny three", "Medium"),
			SpreadLabels(), PlainText(), StartAngle(90), CounterClock(false)); err != nil {
			t.Fatal(err)
		}
		return f
	},
}

// tokens splits an SVG document into numbers and the text between them.
func tokens(b []byte) (text []string, nums []float64) {
	isNum := func(c byte) bool { return c >= '0' && c <= '9' || c == '.' || c == '-' }
	var cur []byte
	for i := 0; i < len(b); {
		if b[i] == '#' { // a colour: text, digits included
			j := i + 1
			for j < len(b) && isHex(b[j]) {
				j++
			}
			cur = append(cur, b[i:j]...)
			i = j
			continue
		}
		if isNum(b[i]) {
			j := i
			for j < len(b) && isNum(b[j]) {
				j++
			}
			if v, err := strconv.ParseFloat(string(b[i:j]), 64); err == nil {
				text = append(text, string(cur))
				nums = append(nums, v)
				cur = cur[:0]
				i = j
				continue
			}
		}
		cur = append(cur, b[i])
		i++
	}
	return append(text, string(cur)), nums
}

func isHex(c byte) bool { return c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' || c >= '0' && c <= '9' }

// sameSVG reports whether two documents are equal but for numbers within
// 2/1000 pixel: the architectures that fuse multiply-adds round some
// coordinates one step differently.
//
// Line ends are compared as LF: a Windows checkout turns the golden files'
// into CRLF.
func sameSVG(a, b []byte) bool {
	crlf, lf := []byte("\r\n"), []byte("\n")
	ta, na := tokens(bytes.ReplaceAll(a, crlf, lf))
	tb, nb := tokens(bytes.ReplaceAll(b, crlf, lf))
	if len(ta) != len(tb) || len(na) != len(nb) {
		return false
	}
	for i := range ta {
		if ta[i] != tb[i] {
			return false
		}
	}
	for i := range na {
		if math.Abs(na[i]-nb[i]) > 0.0021 {
			return false
		}
	}
	return true
}

// TestGoldenSVG compares each figure's SVG with its golden file.
// Regenerate with go test -run TestGoldenSVG -update-golden, and look at
// the result before committing it.
func TestGoldenSVG(t *testing.T) {
	for name, mk := range goldenFigures {
		var buf bytes.Buffer
		if err := mk(t).SaveSVG(&buf); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", "golden", name+".svg")
		if *updateGolden {
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := goldenFS.ReadFile("testdata/golden/" + name + ".svg")
		if err != nil {
			t.Fatalf("%s: %v (run with -update-golden)", name, err)
		}
		if !sameSVG(buf.Bytes(), want) {
			t.Errorf("%s: the SVG differs from %s", name, path)
		}
	}
}

// TestSameSVG is the comparison's own check: a number off by more than the
// tolerance, a changed attribute or a missing element are differences.
func TestSameSVG(t *testing.T) {
	base := []byte(`<path d="M1.5 2 L-3 4" fill="#1f77b4"/>`)
	for _, c := range []struct {
		doc  string
		same bool
	}{
		{`<path d="M1.5 2 L-3 4" fill="#1f77b4"/>`, true},
		{`<path d="M1.501 2 L-3.002 4" fill="#1f77b4"/>`, true},
		{`<path d="M1.51 2 L-3 4" fill="#1f77b4"/>`, false},
		{`<path d="M1.5 2 L-3 4" fill="#1f77b5"/>`, false},
		{`<path d="M1.5 2 L-3 4 L5 6" fill="#1f77b4"/>`, false},
		{`<path d="M1.5 2 C-3 4" fill="#1f77b4"/>`, false},
	} {
		if got := sameSVG(base, []byte(c.doc)); got != c.same {
			t.Errorf("%s: same=%v, want %v", c.doc, got, c.same)
		}
	}
	if !sameSVG([]byte("<g>\n<path d=\"M1 2\"/>\n"), []byte("<g>\r\n<path d=\"M1 2\"/>\r\n")) {
		t.Error("CRLF line ends make a difference")
	}
}
