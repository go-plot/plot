// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/render"
)

// nullRenderer draws nothing; measuring an Axes draws it through one.
type nullRenderer struct{ w, h int }

func (n nullRenderer) Size() (int, int)                          { return n.w, n.h }
func (nullRenderer) DrawPath(*render.Path, render.Style)         {}
func (nullRenderer) DrawImage(render.Image)                      {}
func (nullRenderer) PushClip(float64, float64, float64, float64) {}
func (nullRenderer) PopClip()                                    {}
func (nullRenderer) BeginGroup(string)                           {}
func (nullRenderer) EndGroup()                                   {}

// tightBox is the extent of an Axes and of every text it draws (tick
// labels, axis labels, offsets, title, texts), in pixels: matplotlib's
// get_tightbbox for layout.
func (f *Figure) tightBox(a *Axes) box {
	var texts []box
	f.collect = &texts
	w, h := f.PixelSize()
	a.draw(nullRenderer{w, h}, 0)
	f.collect = nil
	b := a.px
	for _, t := range texts {
		b = box{math.Min(b.x0, t.x0), math.Min(b.y0, t.y0), math.Max(b.x1, t.x1), math.Max(b.y1, t.y1)}
	}
	return b
}

// TightLayout moves the grid made by Subplots so that every Axes' tick
// labels, axis labels and title fit in the figure and do not overlap their
// neighbours, as matplotlib's fig.tight_layout() does: a pad of 1.08 times
// the font size (10 points) at the figure edges and between cells. Axes
// added with AddAxes are left where they are; an Axes with a colorbar
// counts as one cell with it.
//
// It reports false, changing nothing, when the decorations cannot fit (or
// there is no grid), as matplotlib then warns and does nothing.
func (f *Figure) TightLayout() bool {
	var g *gridSpec
	for _, a := range f.axes {
		if a.cell != nil {
			g = a.cell.g
			break
		}
	}
	if g == nil {
		return false
	}
	f.layout()
	W, H := f.Width, f.Height
	wpx, hpx := f.PixelSize()
	// Per cell: the union of the positions and of the tight boxes of every
	// Axes in it (twins, colorbars), in figure fractions with y up.
	type ext struct{ pos, tight [4]float64 } // x0, y0, x1, y1
	cells := map[cell]*ext{}
	for _, a := range f.axes {
		if a.cell == nil || a.cell.g != g {
			continue
		}
		members := []*Axes{a}
		if a.cbar != nil {
			members = append(members, a.cbar)
		}
		e := cells[*a.cell]
		if e == nil {
			e = &ext{pos: [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}, tight: [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}}
			cells[*a.cell] = e
		}
		for _, m := range members {
			p := m.pos
			e.pos = [4]float64{math.Min(e.pos[0], p.x), math.Min(e.pos[1], p.y), math.Max(e.pos[2], p.x+p.w), math.Max(e.pos[3], p.y+p.h)}
			t := f.tightBox(m)
			fx0, fx1 := t.x0/float64(wpx), t.x1/float64(wpx)
			fy0, fy1 := 1-t.y1/float64(hpx), 1-t.y0/float64(hpx)
			e.tight = [4]float64{math.Min(e.tight[0], fx0), math.Min(e.tight[1], fy0), math.Max(e.tight[2], fx1), math.Max(e.tight[3], fy1)}
		}
	}
	// matplotlib's _auto_adjust_subplotpars.
	hspaces := make([][]float64, g.rows) // rows x (cols+1)
	for i := range hspaces {
		hspaces[i] = make([]float64, g.cols+1)
	}
	vspaces := make([][]float64, g.rows+1) // (rows+1) x cols
	for i := range vspaces {
		vspaces[i] = make([]float64, g.cols)
	}
	for c, e := range cells {
		hspaces[c.row][c.col] += e.pos[0] - e.tight[0]
		hspaces[c.row][c.col+1] += e.tight[2] - e.pos[2]
		vspaces[c.row][c.col] += e.tight[3] - e.pos[3]
		vspaces[c.row+1][c.col] += e.pos[1] - e.tight[1]
	}
	padInch := 1.08 * defaultFontSize / 72
	colMax := func(m [][]float64, j0, j1 int) float64 {
		v := math.Inf(-1)
		for _, row := range m {
			for _, x := range row[j0:j1] {
				v = math.Max(v, x)
			}
		}
		return v
	}
	rowMax := func(m [][]float64, i0, i1 int) float64 {
		v := math.Inf(-1)
		for _, row := range m[i0:i1] {
			for _, x := range row {
				v = math.Max(v, x)
			}
		}
		return v
	}
	ml := math.Max(colMax(hspaces, 0, 1), 0) + padInch/W
	mr := math.Max(colMax(hspaces, g.cols, g.cols+1), 0) + padInch/W
	mt := math.Max(rowMax(vspaces, 0, 1), 0) + padInch/H
	mb := math.Max(rowMax(vspaces, g.rows, g.rows+1), 0) + padInch/H
	if ml+mr >= 1 || mb+mt >= 1 {
		return false
	}
	sp := f.SubplotParams
	sp.Left, sp.Right, sp.Bottom, sp.Top = ml, 1-mr, mb, 1-mt
	if g.cols > 1 {
		hs := colMax(hspaces, 1, g.cols) + padInch/W
		wAxes := (1 - mr - ml - hs*float64(g.cols-1)) / float64(g.cols)
		if wAxes < 0 {
			return false
		}
		sp.WSpace = hs / wAxes
	}
	if g.rows > 1 {
		vs := rowMax(vspaces, 1, g.rows) + padInch/H
		hAxes := (1 - mt - mb - vs*float64(g.rows-1)) / float64(g.rows)
		if hAxes < 0 {
			return false
		}
		sp.HSpace = vs / hAxes
	}
	f.SubplotParams = sp
	for _, a := range f.axes {
		if a.cell == nil || a.cell.g != g {
			continue
		}
		r := sp.cellRect(*a.cell)
		if a.cbar != nil {
			a.splitForColorbar(r)
		} else {
			a.pos = r
		}
	}
	return true
}
