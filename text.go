// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// textArt is a string placed in data coordinates, optionally with an arrow
// to a point (an annotation).
type textArt struct {
	s      string
	x, y   float64
	size   float64 // points
	ha, va string
	rot    float64
	color  colors.RGBA
	z      float64
	arrow  bool
	ax, ay float64 // the annotated point
	plain  bool    // no TeX
}

func (t *textArt) zorder() float64           { return t.z }
func (t *textArt) limits() lim               { return emptyLim() }
func (t *textArt) sticky() (x, y []float64)  { return nil, nil }
func (t *textArt) legendEntry() *legendEntry { return nil }

// spec is the text placed in figure pixels.
func (t *textArt) spec(a *Axes) textSpec {
	x, y := a.toPx(t.x, t.y)
	return textSpec{S: t.s, X: x, Y: y, Size: a.fig.pt(t.size), HA: t.ha, VA: t.va, Rot: t.rot, Color: t.color, Plain: t.plain}
}

func (t *textArt) draw(a *Axes, r render.Renderer) {
	f := a.fig
	b := f.drawText(r, t.spec(a))
	if !t.arrow {
		return
	}
	// An arrow from the edge of the text box to the point, as matplotlib's
	// arrowprops=dict(arrowstyle="->") draws it: shrunk 2 points at both
	// ends, a head of two strokes.
	tx, ty := a.toPx(t.ax, t.ay)
	cx, cy := (b.x0+b.x1)/2, (b.y0+b.y1)/2
	sx, sy := clipToBox(cx, cy, tx, ty, b)
	dx, dy := tx-sx, ty-sy
	l := math.Hypot(dx, dy)
	if l == 0 {
		return
	}
	ux, uy := dx/l, dy/l
	shrink := f.pt(2)
	sx, sy = sx+ux*shrink, sy+uy*shrink
	tx, ty = tx-ux*shrink, ty-uy*shrink
	st := render.Style{Stroke: t.color, Width: f.pt(1), Cap: render.CapButt, Join: render.JoinMiter}
	r.DrawPath(line(sx, sy, tx, ty), st)
	head := f.pt(0.4 * 10) // head_length 0.4 x mutation scale (font size)
	var p render.Path
	p.MoveTo(tx-head*ux+head*0.5*uy, ty-head*uy-head*0.5*ux)
	p.LineTo(tx, ty)
	p.LineTo(tx-head*ux-head*0.5*uy, ty-head*uy+head*0.5*ux)
	r.DrawPath(&p, st)
}

// clipToBox returns where the segment from (cx, cy) (inside b) towards (tx,
// ty) leaves the box.
func clipToBox(cx, cy, tx, ty float64, b box) (float64, float64) {
	t := 1.0
	dx, dy := tx-cx, ty-cy
	if dx != 0 {
		for _, e := range []float64{b.x0, b.x1} {
			if s := (e - cx) / dx; s > 0 && s < t {
				t = s
			}
		}
	}
	if dy != 0 {
		for _, e := range []float64{b.y0, b.y1} {
			if s := (e - cy) / dy; s > 0 && s < t {
				t = s
			}
		}
	}
	return cx + t*dx, cy + t*dy
}

func newText(p props, s string, x, y float64) *textArt {
	t := &textArt{s: s, x: x, y: y, size: defaultFontSize, ha: "left", va: "baseline", color: black, z: 3, rot: p.rotation}
	if p.fontSize > 0 {
		t.size = p.fontSize
	}
	if p.ha != "" {
		t.ha = p.ha
	}
	if p.va != "" {
		t.va = p.va
	}
	if p.setColor {
		t.color = p.color
	}
	if p.setAlpha {
		t.color.A = p.alpha
	}
	if p.setZorder {
		t.z = p.zorder
	}
	return t
}

// Text writes s at (x, y) in data coordinates, as matplotlib's ax.text: 10
// points, black, left-aligned on its baseline, not clipped. s may contain
// TeX math between dollar signs.
//
// Options: FontSize, Color, Alpha, HA, VA, Rotation, ZOrder.
func (a *Axes) Text(x, y float64, s string, opts ...Option) {
	a.add(newText(collect(opts), s, x, y))
}

// Annotate writes s at (tx, ty) and, with the Arrow option, draws an arrow
// from it to the point (x, y), as matplotlib's ax.annotate(s, xy=(x, y),
// xytext=(tx, ty), arrowprops=dict(arrowstyle="->")). Both points are in
// data coordinates.
//
// Options: Arrow, and those of Text.
func (a *Axes) Annotate(s string, x, y, tx, ty float64, opts ...Option) {
	p := collect(opts)
	t := newText(p, s, tx, ty)
	t.arrow, t.ax, t.ay = p.arrow, x, y
	a.add(t)
}
