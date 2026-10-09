// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/contour"
	"github.com/go-plot/plot/render"
	"github.com/go-plot/plot/ticker"
)

// contourArt is a set of iso-lines or iso-bands, as matplotlib's
// ContourSet.
type contourArt struct {
	g      contour.Grid
	levels []float64
	filled bool
	cols   []colors.RGBA // per level (lines) or per band (filled)
	lw     float64
	z      float64
	label  string
}

func (c *contourArt) zorder() float64 { return c.z }

func (c *contourArt) limits() lim {
	lm := emptyLim()
	lm.addXY(c.g.X[0], c.g.Y[0])
	lm.addXY(c.g.X[len(c.g.X)-1], c.g.Y[len(c.g.Y)-1])
	return lm
}

func (c *contourArt) sticky() (x, y []float64) {
	l := c.limits()
	return []float64{l.x0, l.x1}, []float64{l.y0, l.y1}
}

func (c *contourArt) legendEntry() *legendEntry { return nil }

func (c *contourArt) draw(a *Axes, r render.Renderer) {
	a.clipToAxes(r)
	defer r.PopClip()
	if c.filled {
		for k := 0; k+1 < len(c.levels); k++ {
			var p render.Path
			for _, pg := range contour.Band(c.g, c.levels[k], c.levels[k+1]) {
				for i, q := range pg {
					x, y := a.toPx(q.X, q.Y)
					if i == 0 {
						p.MoveTo(x, y)
					} else {
						p.LineTo(x, y)
					}
				}
				p.Close()
			}
			r.DrawPath(&p, render.Style{Fill: c.cols[k]})
		}
		return
	}
	for k, lv := range c.levels {
		var pts []render.Pt
		for _, ln := range contour.Lines(c.g, lv) {
			for _, q := range ln {
				x, y := a.toPx(q.X, q.Y)
				pts = append(pts, render.Pt{X: x, Y: y})
			}
			pts = append(pts, render.Pt{X: math.NaN(), Y: math.NaN()})
		}
		if len(pts) == 0 {
			continue
		}
		r.DrawPath(polylinePath(pts), render.Style{Stroke: c.cols[k], Width: a.fig.pt(c.lw), Join: render.JoinRound, Cap: render.CapButt})
	}
}

// autoLevels is matplotlib's ContourSet._autolev: MaxNLocator(n+1,
// min_n_ticks=1) over the data range, trimmed to one level beyond each end.
func autoLevels(zmin, zmax float64, n int) []float64 {
	loc := &ticker.MaxNLocator{Nbins: n + 1, MinNTicks: 1}
	lev := loc.TickValues(zmin, zmax)
	i0, i1 := 0, len(lev)
	for i, v := range lev {
		if v < zmin {
			i0 = i
		}
	}
	for i, v := range lev {
		if v > zmax {
			i1 = i + 1
			break
		}
	}
	if i1-i0 < 3 {
		i0, i1 = 0, len(lev)
	}
	return lev[i0:i1]
}

// contourGrid builds the grid of a contour call: z as 2-D data, x and y
// as the column and row coordinates (nil for 0, 1, 2, ...).
func contourGrid(x, y, z any) contour.Grid {
	g := grid(z)
	cg := contour.Grid{X: Values(x), Y: Values(y), Z: g.Z}
	if cg.X == nil {
		cg.X = indices(g.Cols)
	}
	if cg.Y == nil {
		cg.Y = indices(g.Rows)
	}
	if len(cg.X) != g.Cols || len(cg.Y) != g.Rows {
		panic("plot: contour: x and y must match the columns and rows of z")
	}
	return cg
}

func (a *Axes) contour(x, y, z any, filled bool, opts []Option) *Mappable {
	g := contourGrid(x, y, z)
	p := collect(opts)
	zmin, zmax := math.Inf(1), math.Inf(-1)
	for _, v := range g.Z {
		if !math.IsNaN(v) {
			zmin, zmax = math.Min(zmin, v), math.Max(zmax, v)
		}
	}
	levels := p.levels
	if levels == nil {
		n := 7
		if p.nlevels > 0 {
			n = p.nlevels
		}
		levels = autoLevels(zmin, zmax, n)
	}
	c := &contourArt{g: g, levels: levels, filled: filled, lw: defaultLineWidth, z: 2, label: p.label}
	if filled {
		c.z = 1
	}
	if p.lineWidth > 0 {
		c.lw = p.lineWidth
	}
	if p.setZorder {
		c.z = p.zorder
	}
	values := levels
	if filled {
		values = make([]float64, len(levels)-1)
		for i := range values {
			values[i] = 0.5 * (levels[i] + levels[i+1])
		}
	}
	// The norm spans the levels, not the data, as in matplotlib.
	m := newMappable(p, levels)
	c.cols = make([]colors.RGBA, len(values))
	for i, v := range values {
		c.cols[i] = m.color(v)
		if p.setColor {
			c.cols[i] = p.color
		}
		if p.setAlpha {
			c.cols[i].A = p.alpha
		}
	}
	if filled {
		m.boundaries, m.values = levels, values
	}
	a.add(c)
	return m
}

// Contour draws iso-lines of z, as matplotlib's ax.contour: about 7 levels
// chosen like axis ticks, each coloured from the colormap over the levels'
// range. x and y give the column and row coordinates and may be nil.
//
// Options: Levels, NLevels, Cmap, Vmin, Vmax, Color (all lines), LineWidth,
// Alpha, ZOrder.
func (a *Axes) Contour(x, y, z any, opts ...Option) *Mappable {
	return a.contour(x, y, z, false, opts)
}

// Contourf fills the bands between levels of z, as matplotlib's
// ax.contourf; each band takes the colour of its middle value.
//
// Options: as Contour.
func (a *Axes) Contourf(x, y, z any, opts ...Option) *Mappable {
	return a.contour(x, y, z, true, opts)
}
