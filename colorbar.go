// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"image"
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// colorbarSpec marks an Axes as a colorbar for a Mappable.
type colorbarSpec struct{ m *Mappable }

// colorbarArt paints the colour scale inside a colorbar Axes.
type colorbarArt struct{ m *Mappable }

func (c *colorbarArt) zorder() float64           { return 0 }
func (c *colorbarArt) sticky() (x, y []float64)  { return nil, nil }
func (c *colorbarArt) legendEntry() *legendEntry { return nil }

func (c *colorbarArt) limits() lim {
	lm := emptyLim()
	lo, hi := c.m.Norm.Range()
	lm.addXY(0, lo)
	lm.addXY(1, hi)
	return lm
}

func (c *colorbarArt) draw(a *Axes, r render.Renderer) {
	if c.m.boundaries != nil {
		// Discrete bands (contour sets): one rectangle per band.
		for k, v := range c.m.values {
			col := c.m.color(v)
			y0, y1 := a.yPx(c.m.boundaries[k]), a.yPx(c.m.boundaries[k+1])
			var p render.Path
			p.Rect(a.px.x0, math.Min(y0, y1), a.px.x1-a.px.x0, math.Abs(y1-y0))
			r.DrawPath(&p, render.Style{Fill: col, Snap: true})
		}
		return
	}
	// A continuous colorbar shows every entry of the lookup table, the
	// highest value at the top, as matplotlib's 256 bands do.
	pix := image.NewRGBA(image.Rect(0, 0, 1, 256))
	for i := range 256 {
		col := c.m.Cmap.At((float64(i) + 0.5) / 256)
		b := [4]uint8{uint8(col.R * 255), uint8(col.G * 255), uint8(col.B * 255), uint8(col.A * 255)}
		copy(pix.Pix[(255-i)*4:], b[:])
	}
	r.DrawImage(render.Image{X: a.px.x0, Y: a.px.y0, W: a.px.x1 - a.px.x0, H: a.px.y1 - a.px.y0, Pix: pix})
}

// colorbarOutline is the black frame around a colorbar (z-order 2.5).
type colorbarOutline struct{}

func (colorbarOutline) zorder() float64           { return 2.5 }
func (colorbarOutline) limits() lim               { return emptyLim() }
func (colorbarOutline) sticky() (x, y []float64)  { return nil, nil }
func (colorbarOutline) legendEntry() *legendEntry { return nil }

func (colorbarOutline) draw(a *Axes, r render.Renderer) {
	var p render.Path
	p.Rect(a.px.x0, a.px.y0, a.px.x1-a.px.x0, a.px.y1-a.px.y0)
	r.DrawPath(&p, render.Style{Stroke: black, Width: a.fig.pt(spineWidth), Join: render.JoinMiter, Snap: true})
}

// Colorbar adds a colour scale for m beside ax, taking its space from ax as
// matplotlib's fig.colorbar(m, ax=ax) does: 15% of the width, a 5% gap, a
// bar 20 times as tall as wide, ticks on the right. It returns the
// colorbar's Axes, whose y label can be set.
func (f *Figure) Colorbar(m *Mappable, ax *Axes) *Axes {
	cax := f.AddAxes(0, 0, 1, 1)
	cax.boxAspect = colorbarAspect
	cax.anchor = [2]float64{0, 0.5}
	cax.colorbar = &colorbarSpec{m: m}
	cax.xHidden = true
	cax.y.right = true
	lo, hi := m.Norm.Range()
	cax.x.vmin, cax.x.vmax, cax.x.auto = 0, 1, false
	cax.y.vmin, cax.y.vmax, cax.y.auto = lo, hi, false
	if _, ok := m.Norm.(colors.LogNorm); ok {
		cax.y.log = true
	}
	cax.add(&colorbarArt{m: m})
	cax.add(colorbarOutline{})
	ax.cbar = cax
	ax.splitForColorbar(ax.pos)
	return cax
}

// colorbar geometry: matplotlib's make_axes defaults.
const colorbarFraction, colorbarPad, colorbarAspect = 0.15, 0.05, 20.0

// splitForColorbar places the Axes and its colorbar in the rectangle pb,
// as matplotlib's make_axes splits it: the bar takes 15% of the width
// after a 5% gap, the Axes the rest, anchored against the bar.
func (a *Axes) splitForColorbar(pb rect) {
	x1 := pb.x + (1-colorbarFraction-colorbarPad)*pb.w
	x2 := pb.x + (1-colorbarFraction)*pb.w
	a.pos = rect{pb.x, pb.y, x1 - pb.x, pb.h}
	a.anchor = [2]float64{1, 0.5}
	a.cbar.pos = rect{x2, pb.y, pb.x + pb.w - x2, pb.h}
}
