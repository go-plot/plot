// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package render

import (
	"bytes"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/go-plot/plot/colors"
)

var black = colors.RGBA{A: 1}

func alphaAt(r *Raster, x, y int) uint8 { return r.Img.Pix[y*r.Img.Stride+x*4+3] }

func TestAffine(t *testing.T) {
	m := Translate(1, 2).Then(Scale(2, 3)).Then(Rotate(90))
	p := m.Apply(Pt{X: 1, Y: 1})
	// (1,1) -> (2,3) -> (4,9) -> rotate 90 deg counter-clockwise on screen.
	if math.Abs(p.X-9) > 1e-12 || math.Abs(p.Y+4) > 1e-12 {
		t.Errorf("got %v", p)
	}
	if Identity.Apply(Pt{X: 3, Y: 4}) != (Pt{X: 3, Y: 4}) {
		t.Error("identity")
	}
}

func TestPathBounds(t *testing.T) {
	var p Path
	if _, _, _, _, ok := p.Bounds(); ok {
		t.Error("empty path has bounds")
	}
	p.MoveTo(1, 1)
	p.QuadTo(5, 0, 2, 2)
	p.CubicTo(0, 3, 0, 3, 1, 1)
	p.Close()
	x0, y0, x1, y1, ok := p.Bounds()
	if !ok || x0 != 0 || y0 != 0 || x1 != 5 || y1 != 3 {
		t.Errorf("bounds %v %v %v %v", x0, y0, x1, y1)
	}
	var q Path
	q.Circle(10, 10, 5)
	q.Append(&p, Translate(100, 0))
	if len(q.Segs) != 6+4 {
		t.Errorf("%d segments", len(q.Segs))
	}
}

func TestSnap(t *testing.T) {
	var p Path
	p.MoveTo(10.2, 20.7)
	p.LineTo(30.4, 20.7)
	p.Close()
	q := snapPath(&p, 1)
	if q.Segs[0].P[0] != (Pt{X: 10.5, Y: 21.5}) || q.Segs[1].P[0] != (Pt{X: 30.5, Y: 21.5}) {
		t.Errorf("odd width: %v", q.Segs)
	}
	q = snapPath(&p, 2)
	if q.Segs[0].P[0] != (Pt{X: 10, Y: 21}) {
		t.Errorf("even width: %v", q.Segs)
	}
	var slanted Path
	slanted.MoveTo(0, 0)
	slanted.LineTo(1, 1)
	if snapPath(&slanted, 1) != &slanted {
		t.Error("slanted path snapped")
	}
	var curve Path
	curve.MoveTo(0, 0)
	curve.QuadTo(1, 0, 1, 1)
	if snapPath(&curve, 1) != &curve {
		t.Error("curve snapped")
	}
	var long Path
	for i := range 1100 {
		long.LineTo(float64(i), 0)
	}
	if snapPath(&long, 1) != &long {
		t.Error("long path snapped")
	}
}

func TestRasterShapes(t *testing.T) {
	r := NewRaster(40, 40)
	if w, h := r.Size(); w != 40 || h != 40 {
		t.Fatal("size")
	}
	r.BeginGroup("x")
	r.EndGroup()
	var sq Path
	sq.Rect(5, 5, 10, 10)
	sq.Rect(8, 8, 4, 4)
	r.DrawPath(&sq, Style{Fill: black, EvenOdd: true})
	if alphaAt(r, 6, 6) != 255 || alphaAt(r, 10, 10) != 0 {
		t.Error("even-odd fill")
	}
	// Clip, nested: only the intersection is painted.
	r.PushClip(20, 20, 15, 15)
	r.PushClip(25, 0, 40, 40)
	var big Path
	big.Rect(0, 0, 40, 40)
	r.DrawPath(&big, Style{Fill: black})
	r.PopClip()
	r.PopClip()
	if alphaAt(r, 22, 22) != 0 || alphaAt(r, 27, 27) != 255 || alphaAt(r, 37, 37) != 0 {
		t.Error("nested clip")
	}
	// A long stroke is drawn in chunks, a subpath break included.
	r2 := NewRaster(200, 40)
	var zig Path
	for i := range 1200 {
		x, y := float64(i)/6, 10+5*float64(i%2)
		if i == 600 {
			zig.MoveTo(x, y)
			continue
		}
		zig.LineTo(x, y)
	}
	r2.DrawPath(&zig, Style{Stroke: black, Width: 1})
	if alphaAt(r2, 50, 12) == 0 || alphaAt(r2, 150, 12) == 0 {
		t.Error("chunked stroke missing")
	}
	// Stroke with every cap, join and a dash.
	for c := range 3 {
		for j := range 3 {
			var l Path
			l.MoveTo(5, 30)
			l.LineTo(20, 30)
			l.LineTo(20, 35)
			r.DrawPath(&l, Style{Stroke: black, Width: 2, Cap: Cap(c), Join: Join(j), Dash: []float64{2, 1}, Offset: 0.5})
		}
	}
	if alphaAt(r, 6, 30) == 0 {
		t.Error("stroke missing")
	}
}

func TestRasterImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range src.Pix {
		src.Pix[i] = 255
	}
	sub := src.SubImage(image.Rect(1, 0, 2, 2)).(*image.RGBA) // stride != width*4
	r := NewRaster(10, 10)
	r.DrawImage(Image{X: 2, Y: 2, W: 4, H: 4, Pix: sub})
	if alphaAt(r, 3, 3) != 255 || alphaAt(r, 7, 7) != 0 {
		t.Error("image placement")
	}
}

func TestSVG(t *testing.T) {
	s := NewSVG(100, 50, 72)
	if w, h := s.Size(); w != 100 || h != 50 {
		t.Fatal("size")
	}
	s.BeginGroup("g1")
	s.PushClip(0, 0, 50, 50)
	var p Path
	p.MoveTo(0, -0.0001)
	p.QuadTo(1, 1, 2, 2)
	p.CubicTo(1, 1, 2, 2, 3, 3)
	p.Close()
	s.DrawPath(&p, Style{Fill: colors.RGBA{R: 1, A: 0.5}, EvenOdd: true, Stroke: colors.RGBA{B: 1, A: 0.25}, Width: 1.5, Cap: CapRound, Join: JoinBevel, Dash: []float64{1, 2}, Offset: 3})
	s.DrawPath(&Path{}, Style{Fill: black})
	s.DrawPath(&p, Style{Stroke: black, Width: 1})
	src := image.NewRGBA(image.Rect(1, 1, 3, 3))
	s.DrawImage(Image{X: 1, Y: 1, W: 2, H: 2, Pix: src})
	s.PopClip()
	s.EndGroup()
	var out bytes.Buffer
	if _, err := s.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	doc := out.String()
	for _, want := range []string{
		`width="100pt" height="50pt" viewBox="0 0 100 50"`,
		`"><rect x="0" y="0" width="50" height="50"/></clipPath>`,
		`d="M0 0 Q1 1 2 2 C1 1 2 2 3 3 z"`,
		`fill="#ff0000" fill-opacity="0.5" fill-rule="evenodd" stroke="#0000ff" stroke-opacity="0.25" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="bevel" stroke-dasharray="1,2" stroke-dashoffset="3"`,
		`fill="none" stroke="#000000"`,
		`image-rendering="pixelated"`,
		`<image x="1" y="1" width="2" height="2"`,
		`<g id="g1">`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in\n%s", want, doc)
		}
	}
	if strings.Count(doc, "<path") != 2 {
		t.Error("empty path written")
	}
	if strings.Contains(doc, "style=") {
		t.Error("a style attribute: a strict CSP drops it when the SVG is inlined")
	}
}

// TestSVGClipIDs: a clip's id names its rectangle, so the same rectangle is
// defined once, a different one gets a different id, and the id does not
// depend on how many clips came before (two figures inlined in one page must
// not resolve each other's url(#…) to a different box).
func TestSVGClipIDs(t *testing.T) {
	write := func(rects ...[4]float64) string {
		s := NewSVG(100, 100, 72)
		for _, r := range rects {
			s.PushClip(r[0], r[1], r[2], r[3])
			s.PopClip()
		}
		var out bytes.Buffer
		if _, err := s.WriteTo(&out); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	a, b := [4]float64{0, 0, 50, 50}, [4]float64{10, 10, 20, 20}
	one := write(a, a, b)
	if n := strings.Count(one, "<clipPath"); n != 2 {
		t.Errorf("%d clipPath definitions for two distinct rectangles:\n%s", n, one)
	}
	other := write(b)
	idOf := func(doc string, at int) string {
		i := strings.Index(doc, `<clipPath id="`)
		for ; at > 0; at-- {
			i += 1 + strings.Index(doc[i+1:], `<clipPath id="`)
		}
		rest := doc[i+len(`<clipPath id="`):]
		return rest[:strings.IndexByte(rest, '"')]
	}
	if idOf(one, 1) != idOf(other, 0) {
		t.Errorf("the same rectangle got ids %q and %q in two documents", idOf(one, 1), idOf(other, 0))
	}
	if idOf(one, 0) == idOf(one, 1) {
		t.Error("two rectangles share an id")
	}
	// No clip, no defs.
	var bare bytes.Buffer
	if _, err := NewSVG(1, 1, 100).WriteTo(&bare); err != nil || strings.Contains(bare.String(), "<defs>") {
		t.Errorf("bare document: %v %s", err, bare.String())
	}
}

func TestRasterCurvesAndSnap(t *testing.T) {
	r := NewRaster(40, 40)
	var c Path
	c.Circle(20, 20, 10)
	c.MoveTo(0, 0)
	c.QuadTo(10, 0, 10, 10)
	r.DrawPath(&c, Style{Fill: black, Snap: true})
	if alphaAt(r, 20, 20) != 255 {
		t.Error("circle fill")
	}
	var l Path
	l.MoveTo(2.3, 35.2)
	l.LineTo(30.6, 35.2)
	r.DrawPath(&l, Style{Stroke: black, Width: 1, Snap: true})
	if alphaAt(r, 10, 35) != 255 || alphaAt(r, 10, 36) != 0 {
		t.Errorf("snapped 1px line not crisp: %d %d", alphaAt(r, 10, 35), alphaAt(r, 10, 36))
	}
}
