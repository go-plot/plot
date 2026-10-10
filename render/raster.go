// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package render

import (
	"image"
	"math"

	"github.com/go-gfx/gfx/vector"
	"github.com/go-widgets/painter"
)

// Raster renders into an RGBA image. Shapes are rasterised to anti-aliased
// coverage by go-gfx/gfx/vector (which draws every cap, join and dash a
// matplotlib line can have) and composited through go-widgets/painter's
// PixelPainter as a coverage mask under its clip stack; images go through the
// painter's DrawImage.
type Raster struct {
	Img  *image.RGBA
	p    *painter.PixelPainter
	rz   vector.Rasterizer
	mask []byte
	clip []clipRect
}

type clipRect struct{ x0, y0, x1, y1 int }

// NewRaster returns a Raster of w x h pixels filled with nothing
// (transparent); the figure paints its own background.
func NewRaster(w, h int) *Raster {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	return &Raster{Img: img, p: painter.NewPixelPainter(img.Pix, w, h)}
}

// Size implements Renderer.
func (r *Raster) Size() (int, int) { return r.Img.Rect.Dx(), r.Img.Rect.Dy() }

// BeginGroup implements Renderer; a raster keeps no structure.
func (r *Raster) BeginGroup(string) {}

// EndGroup implements Renderer.
func (r *Raster) EndGroup() {}

// PushClip implements Renderer. The rectangle is rounded to whole pixels, as
// Agg's clip box is.
func (r *Raster) PushClip(x, y, w, h float64) {
	c := clipRect{int(math.Round(x)), int(math.Round(y)), int(math.Round(x + w)), int(math.Round(y + h))}
	if n := len(r.clip); n > 0 {
		o := r.clip[n-1]
		c = clipRect{max(c.x0, o.x0), max(c.y0, o.y0), min(c.x1, o.x1), min(c.y1, o.y1)}
	}
	r.clip = append(r.clip, c)
	r.p.PushClip(painter.Rect{X: c.x0, Y: c.y0, W: max(0, c.x1-c.x0), H: max(0, c.y1-c.y0)})
}

// PopClip implements Renderer.
func (r *Raster) PopClip() {
	r.clip = r.clip[:len(r.clip)-1]
	r.p.PopClip()
}

// toVector converts a path to go-gfx's representation.
func toVector(p *Path) *vector.Path {
	v := vector.NewPath()
	for _, s := range p.Segs {
		switch s.Op {
		case MoveTo:
			v.MoveTo(s.P[0].X, s.P[0].Y)
		case LineTo:
			v.LineTo(s.P[0].X, s.P[0].Y)
		case QuadTo:
			v.QuadTo(s.P[0].X, s.P[0].Y, s.P[1].X, s.P[1].Y)
		case CubicTo:
			v.CubicTo(s.P[0].X, s.P[0].Y, s.P[1].X, s.P[1].Y, s.P[2].X, s.P[2].Y)
		case Close:
			v.Close()
		}
	}
	return v
}

// DrawPath implements Renderer.
func (r *Raster) DrawPath(p *Path, s Style) {
	stroke := s.Width > 0 && s.Stroke.A > 0
	if s.Snap {
		w := 0.0
		if stroke {
			w = s.Width
		}
		p = snapPath(p, w)
	}
	w, h := r.Size()
	if s.Fill.A > 0 {
		rule := vector.NonZero
		if s.EvenOdd {
			rule = vector.EvenOdd
		}
		cov, ox, oy, cw, ch, ok := r.rz.Fill(toVector(p), rule, w, h)
		if ok {
			r.composite(cov, ox, oy, cw, ch, s.Fill.Bytes())
		}
	}
	if !stroke {
		return
	}
	if len(s.Dash) == 0 && len(p.Segs) > strokeChunk {
		r.strokeChunked(p, s)
		return
	}
	r.stroke(p, s)
}

// stroke rasterises the outline of p and paints it.
func (r *Raster) stroke(p *Path, s Style) {
	w, h := r.Size()
	st := vector.StrokeStyle{
		Width:     s.Width,
		Cap:       [...]vector.LineCap{vector.ButtCap, vector.RoundCap, vector.SquareCap}[s.Cap],
		Join:      [...]vector.LineJoin{vector.MiterJoin, vector.RoundJoin, vector.BevelJoin}[s.Join],
		Dash:      s.Dash,
		DashPhase: s.Offset,
	}
	cov, ox, oy, cw, ch, ok := r.rz.StrokeWith(toVector(p), st, w, h)
	if ok {
		r.composite(cov, ox, oy, cw, ch, s.Stroke.Bytes())
	}
}

// composite turns a coverage grid into an 8-bit mask and paints it.
func (r *Raster) composite(cov []float64, ox, oy, w, h int, c [4]uint8) {
	n := w * h
	if cap(r.mask) < n {
		r.mask = make([]byte, n)
	}
	m := r.mask[:n]
	for i, v := range cov {
		m[i] = uint8(min(1, v)*255 + 0.5)
	}
	r.p.DrawMask(painter.Rect{X: ox, Y: oy, W: w, H: h}, m, w, painter.RGBA{R: c[0], G: c[1], B: c[2], A: c[3]})
}

// DrawImage implements Renderer: the rectangle is rounded to whole pixels and
// the painter scales the image into it, nearest neighbour.
func (r *Raster) DrawImage(im Image) {
	x0, y0 := int(math.Round(im.X)), int(math.Round(im.Y))
	x1, y1 := int(math.Round(im.X+im.W)), int(math.Round(im.Y+im.H))
	sw, sh := im.Pix.Rect.Dx(), im.Pix.Rect.Dy()
	pix := im.Pix.Pix
	if im.Pix.Stride != sw*4 {
		pix = make([]byte, 0, sw*sh*4)
		for y := range sh {
			pix = append(pix, im.Pix.Pix[y*im.Pix.Stride:y*im.Pix.Stride+sw*4]...)
		}
	}
	r.p.DrawImage(painter.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, pix, sw, sh)
}

// strokeChunk is how many segments a stroke is rasterised in at once. The
// coverage sweep sorts the crossings of each scanline by insertion, which
// grows with the square of the segments crossing it: a dense polyline of a
// hundred thousand segments costs seconds in one piece and milliseconds in
// pieces, as matplotlib's agg.path.chunksize does it. The pieces share their
// end vertex and are drawn with round caps so that they meet in the round
// join they would have had; the line's own two ends are round too, which is
// the one visible difference, and only for a line this long.
const strokeChunk = 512

// strokeChunked strokes a long, solid path in pieces of strokeChunk segments.
func (r *Raster) strokeChunked(p *Path, s Style) {
	s.Cap = CapRound
	var cur Path
	var last Pt
	for _, sg := range p.Segs {
		switch {
		case sg.Op == MoveTo:
			r.stroke(&cur, s)
			cur.Segs = cur.Segs[:0]
		case len(cur.Segs) >= strokeChunk:
			r.stroke(&cur, s)
			cur.Segs = cur.Segs[:0]
			cur.MoveTo(last.X, last.Y)
		}
		cur.Segs = append(cur.Segs, sg)
		if n := sg.Op.points(); n > 0 {
			last = sg.P[n-1]
		}
	}
	r.stroke(&cur, s)
}
