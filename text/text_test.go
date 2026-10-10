// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package text

import (
	"errors"
	"math"
	"testing"

	"github.com/go-plot/plot/render"
)

func TestSplitMath(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []part
	}{
		{"plain", []part{{s: "plain"}}},
		{"a $x$ b", []part{{s: "a "}, {s: "x", math: true}, {s: " b"}}},
		{"odd $ only", []part{{s: "odd $ only"}}},
		{`cost \$5`, []part{{s: "cost $5"}}},
		{`$a\$b$ and \$`, []part{{s: `a\$b`, math: true}, {s: " and $"}}},
		{"$$", nil},
	} {
		got := splitMath(c.in)
		if len(got) != len(c.want) {
			t.Errorf("%q: %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: %v, want %v", c.in, got, c.want)
			}
		}
	}
}

func TestLayoutPlain(t *testing.T) {
	d := NewDefault()
	l := d.Layout("Hg", 20)
	if l.Width < 15 || l.Ascent < 10 || l.Descent < 2 {
		t.Errorf("Hg: %+v", l)
	}
	sp := d.Layout(" ", 20)
	if sp.Width != 0 || sp.Ascent != 0 || len(sp.Path.Segs) != 0 {
		t.Errorf("space: %+v", sp)
	}
	a, de, gap := d.LineMetrics(2048)
	// DejaVu Sans: OS/2 typo ascender 1556, descender -492, line gap 410.
	if a != 1556 || de != 492 || gap != 410 {
		t.Errorf("line metrics %v %v %v", a, de, gap)
	}
	if NewDefault() != d {
		t.Error("NewDefault is not shared")
	}
}

func TestMathDefault(t *testing.T) {
	d := NewDefault()
	for _, s := range []string{
		`$\mathdefault{10^{-3}}$`, `$\mathdefault{2\times10^{1}}$`, `$\mathdefault{2\cdot3}$`,
		`$\mathdefault{x_{i}}$`, `$\mathdefault{-1}$`,
	} {
		l := d.Layout(s, 20)
		if l.Width <= 0 || len(l.Path.Segs) == 0 {
			t.Errorf("%s: %+v", s, l)
		}
	}
	sup := d.Layout(`$\mathdefault{10^{2}}$`, 20)
	plain := d.Layout(`$\mathdefault{10}$`, 20)
	if sup.Ascent <= plain.Ascent {
		t.Errorf("superscript does not rise: %v <= %v", sup.Ascent, plain.Ascent)
	}
	// What \mathdefault cannot set goes to TeX, which still renders it.
	for _, s := range []string{`$\mathdefault{x^2}$`, `$\mathdefault{x^{a`, `$\mathdefault{x^{\alpha}}$`, `$\mathdefault{\alpha}$`} {
		if _, ok := d.mathDefault(s[len(`$\mathdefault{`):], 20); ok {
			t.Errorf("%s accepted by mathDefault", s)
		}
	}
	if l := d.Layout(`$\mathdefault{\alpha}$`, 20); len(l.Path.Segs) == 0 {
		t.Error(`\alpha not rendered`)
	}
}

func TestTeXErrorShowsSource(t *testing.T) {
	d := NewDefault()
	l := d.Layout(`$\frac{1}$`, 20)
	if len(l.Path.Segs) == 0 || l.Width < 30 {
		t.Errorf("bad TeX should show its source: %+v", l)
	}
}

func TestMust(t *testing.T) {
	if must(3, nil) != 3 {
		t.Error("must")
	}
	defer func() {
		if recover() == nil {
			t.Error("must did not panic")
		}
	}()
	must(0, errors.New("broken"))
}

func pathOf(d string) *render.Path {
	p := &render.Path{}
	appendPathData(p, d, render.Identity)
	return p
}

func TestAppendPathData(t *testing.T) {
	p := pathOf("M1 2L3,4H5V6Q7 8 9 10C1 2 3 4 5 6Z M0 0 1 1z")
	ops := []render.Op{render.MoveTo, render.LineTo, render.LineTo, render.LineTo, render.QuadTo, render.CubicTo, render.Close, render.MoveTo, render.LineTo, render.Close}
	if len(p.Segs) != len(ops) {
		t.Fatalf("%d segments: %v", len(p.Segs), p.Segs)
	}
	for i, op := range ops {
		if p.Segs[i].Op != op {
			t.Errorf("segment %d: op %v, want %v", i, p.Segs[i].Op, op)
		}
	}
	if p.Segs[2].P[0] != (render.Pt{X: 5, Y: 4}) || p.Segs[3].P[0] != (render.Pt{X: 5, Y: 6}) {
		t.Errorf("H/V: %v %v", p.Segs[2], p.Segs[3])
	}
	// Truncated commands stop the path where they are.
	for _, d := range []string{"M1", "M1 2L3", "M1 2H", "M1 2Q1 2 3", "M1 2C1 2 3 4 5", "1 2"} {
		if p := pathOf(d); len(p.Segs) > 1 {
			t.Errorf("%q: %v", d, p.Segs)
		}
	}
}

func TestParseTransform(t *testing.T) {
	pt := render.Pt{X: 1, Y: 1}
	for _, c := range []struct {
		s    string
		want render.Pt
	}{
		{"translate(2,3)", render.Pt{X: 3, Y: 4}},
		{"translate(2)", render.Pt{X: 3, Y: 1}},
		{"scale(2,3)", render.Pt{X: 2, Y: 3}},
		{"scale(2)", render.Pt{X: 2, Y: 2}},
		{"translate(1,1) scale(2)", render.Pt{X: 3, Y: 3}},
		{"rotate(30) scale(2)", render.Pt{X: 2, Y: 2}},
		{"", pt},
	} {
		if got := parseTransform(c.s).Apply(pt); math.Abs(got.X-c.want.X) > 1e-12 || math.Abs(got.Y-c.want.Y) > 1e-12 {
			t.Errorf("%q: %v, want %v", c.s, got, c.want)
		}
	}
}

func TestParseTeXSVG(t *testing.T) {
	svg := `<svg><g transform="translate(10,0)"><g/><path d="M0 0L1 0Z"/><rect x="1" y="2" width="3" height="4"/></g><path transform="scale(2)" d="M1 1L2 2"/></svg>`
	p := parseTeXSVG(svg, render.Identity)
	x0, y0, x1, y1, ok := p.Bounds()
	if !ok || x0 != 2 || y0 != 0 || x1 != 14 || y1 != 6 {
		t.Errorf("bounds %v %v %v %v", x0, y0, x1, y1)
	}
}
