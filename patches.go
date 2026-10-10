// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// patchStyle is a filled shape's paint: a face, and an edge drawn only when
// an edge colour was asked for (matplotlib's patches have none by default).
type patchStyle struct {
	face  colors.RGBA
	edge  colors.RGBA
	width float64 // edge width, points
}

func newPatchStyle(p props, c colors.RGBA) patchStyle {
	if p.setColor {
		c = p.color
	}
	if p.setFace {
		c = p.faceColor
	}
	if p.setAlpha {
		c.A = p.alpha
	}
	ps := patchStyle{face: c, width: 1}
	if p.setEdge {
		ps.edge = p.edgeColor
	}
	if p.lineWidth > 0 {
		ps.width = p.lineWidth
	}
	return ps
}

func (ps patchStyle) render(f *Figure) render.Style {
	st := render.Style{Fill: ps.face, Snap: true, Join: render.JoinMiter}
	if ps.edge.A > 0 {
		st.Stroke, st.Width = ps.edge, f.pt(ps.width)
	}
	return st
}

// bars is a set of rectangles, as matplotlib's BarContainer.
type bars struct {
	rects    [][4]float64 // x, y, w, h in data
	styles   []patchStyle
	label    string
	z        float64
	stickyX  []float64
	stickyY  []float64
	sameFace bool
}

func (b *bars) zorder() float64 { return b.z }

func (b *bars) limits() lim {
	lm := emptyLim()
	for _, r := range b.rects {
		lm.addXY(r[0], r[1])
		lm.addXY(r[0]+r[2], r[1]+r[3])
	}
	return lm
}

func (b *bars) sticky() (x, y []float64) { return b.stickyX, b.stickyY }

func (b *bars) draw(a *Axes, r render.Renderer) {
	a.clipToAxes(r)
	defer r.PopClip()
	for i, rc := range b.rects {
		x0, y0 := a.toPx(rc[0], rc[1])
		x1, y1 := a.toPx(rc[0]+rc[2], rc[1]+rc[3])
		var p render.Path
		p.Rect(math.Min(x0, x1), math.Min(y0, y1), math.Abs(x1-x0), math.Abs(y1-y0))
		r.DrawPath(&p, b.styles[i].render(a.fig))
	}
}

func (b *bars) legendEntry() *legendEntry {
	if b.label == "" || len(b.styles) == 0 {
		return nil
	}
	return &legendEntry{label: b.label, patch: &b.styles[0]}
}

// barOpts reads the options common to Bar and Barh.
func (a *Axes) barStyles(p props, n int) []patchStyle {
	styles := make([]patchStyle, n)
	var c colors.RGBA
	if !p.setColor && !p.setFace && p.colorsPer == nil {
		c = a.nextFillColor()
	}
	for i := range styles {
		styles[i] = newPatchStyle(p, c)
		if p.colorsPer != nil {
			col := p.colorsPer[i%len(p.colorsPer)]
			if p.setAlpha {
				col.A = p.alpha
			}
			styles[i].face = col
		}
	}
	return styles
}

func broadcast(v []float64, n int, what string) []float64 {
	if len(v) == 1 {
		out := make([]float64, n)
		for i := range out {
			out[i] = v[0]
		}
		return out
	}
	if len(v) != n {
		panic("plot: " + what + ": lengths differ")
	}
	return v
}

// Bar draws vertical bars of the given heights at x, as matplotlib's
// ax.bar: 0.8 wide, centred on x, from 0, in the next colour of the fill
// cycle, with no edge.
//
// Options: Width, Bottom, Align ("center" or "edge"), Color, Colors (one per
// bar), FaceColor, EdgeColor, LineWidth (edge), Alpha, Label, ZOrder.
func (a *Axes) Bar(x, height any, opts ...Option) {
	xs := Values(x)
	hs := broadcast(Values(height), len(xs), "Bar")
	p := collect(opts)
	w := 0.8
	if p.width != 0 {
		w = p.width
	}
	b := &bars{styles: a.barStyles(p, len(xs)), label: p.label, z: 1}
	if p.setZorder {
		b.z = p.zorder
	}
	for i, x := range xs {
		left := x - w/2
		if p.align == "edge" {
			left = x
		}
		b.rects = append(b.rects, [4]float64{left, p.bottom, w, hs[i]})
	}
	b.stickyY = []float64{p.bottom}
	a.add(b)
}

// Barh draws horizontal bars of the given widths at y, as matplotlib's
// ax.barh. Width sets the bar thickness (matplotlib's height=) and Bottom
// the left end.
//
// Options: as Bar.
func (a *Axes) Barh(y, width any, opts ...Option) {
	ys := Values(y)
	ws := broadcast(Values(width), len(ys), "Barh")
	p := collect(opts)
	h := 0.8
	if p.width != 0 {
		h = p.width
	}
	b := &bars{styles: a.barStyles(p, len(ys)), label: p.label, z: 1}
	if p.setZorder {
		b.z = p.zorder
	}
	for i, y := range ys {
		bottom := y - h/2
		if p.align == "edge" {
			bottom = y
		}
		b.rects = append(b.rects, [4]float64{p.bottom, bottom, ws[i], h})
	}
	b.stickyX = []float64{p.bottom}
	a.add(b)
}

// histogram is numpy.histogram with equal-width bins over [lo, hi]: the bin
// of each value is computed as NumPy computes it, then corrected against the
// edges so that a value on an edge lands where NumPy puts it.
func histogram(data []float64, edges []float64, uniform bool) []float64 {
	n := len(edges) - 1
	counts := make([]float64, n)
	lo, hi := edges[0], edges[n]
	denom := hi - lo
	for _, v := range data {
		if math.IsNaN(v) || v < lo || v > hi {
			continue
		}
		var i int
		if uniform {
			i = int(float64((v-lo)/denom) * float64(n))
			if i == n {
				i--
			}
			if v < edges[i] {
				i--
			} else if i+1 < n && v >= edges[i+1] {
				i++
			}
		} else {
			i = searchRight(edges, v) - 1
			if i == n {
				i--
			}
		}
		counts[i]++
	}
	return counts
}

// searchRight is numpy.searchsorted(side="right").
func searchRight(a []float64, v float64) int {
	lo, hi := 0, len(a)
	for lo < hi {
		m := (lo + hi) / 2
		if a[m] <= v {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo
}

// linspace is numpy.linspace(lo, hi, n) for n >= 2 with its rounding:
// step-multiplied and the last value exact.
func linspace(lo, hi float64, n int) []float64 {
	out := make([]float64, n)
	step := (hi - lo) / float64(n-1)
	for i := range out {
		out[i] = float64(float64(i)*step) + lo
	}
	out[n-1] = hi
	return out
}

// Hist draws a histogram of x, as matplotlib's ax.hist with histtype "bar":
// 10 equal bins over the data range by default, touching bars in the next
// colour of the line cycle (as in matplotlib).
//
// Options: Bins, BinEdges, Density, Color, Alpha, EdgeColor, LineWidth,
// Label, ZOrder.
func (a *Axes) Hist(x any, opts ...Option) {
	data := Values(x)
	p := collect(opts)
	edges := p.binEdges
	uniform := false
	if edges == nil {
		nb := p.bins
		if nb <= 0 {
			nb = 10
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, v := range data {
			if !math.IsNaN(v) {
				lo, hi = math.Min(lo, v), math.Max(hi, v)
			}
		}
		if lo > hi {
			lo, hi = 0, 1
		}
		if lo == hi {
			lo, hi = lo-0.5, hi+0.5
		}
		edges = linspace(lo, hi, nb+1)
		uniform = true
	}
	counts := histogram(data, edges, uniform)
	if p.density {
		total := 0.0
		for _, c := range counts {
			total += c
		}
		for i := range counts {
			counts[i] /= total * (edges[i+1] - edges[i])
		}
	}
	var c colors.RGBA
	if !p.setColor && !p.setFace {
		c = a.nextLineColor()
	}
	st := newPatchStyle(p, c)
	b := &bars{label: p.label, z: 1, stickyY: []float64{0}}
	if p.setZorder {
		b.z = p.zorder
	}
	for i, n := range counts {
		b.rects = append(b.rects, [4]float64{edges[i], 0, edges[i+1] - edges[i], n})
		b.styles = append(b.styles, st)
	}
	a.add(b)
}

// polygons is a filled region, as matplotlib's PolyCollection from
// fill_between.
type polygons struct {
	polys [][]render.Pt // data coordinates
	style patchStyle
	label string
	z     float64
}

func (g *polygons) zorder() float64 { return g.z }

func (g *polygons) limits() lim {
	lm := emptyLim()
	for _, pg := range g.polys {
		for _, q := range pg {
			lm.addXY(q.X, q.Y)
		}
	}
	return lm
}

func (g *polygons) sticky() (x, y []float64) { return nil, nil }

func (g *polygons) draw(a *Axes, r render.Renderer) {
	a.clipToAxes(r)
	defer r.PopClip()
	var p render.Path
	for _, pg := range g.polys {
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
	st := g.style.render(a.fig)
	st.Snap = false
	r.DrawPath(&p, st)
}

func (g *polygons) legendEntry() *legendEntry {
	if g.label == "" {
		return nil
	}
	return &legendEntry{label: g.label, patch: &g.style}
}

// FillBetween fills the region between y1 and y2 over x, as matplotlib's
// ax.fill_between. y2 may be nil (zero) or data. Without a colour it takes
// the next colour of the fill cycle.
//
// Options: Where, Color, FaceColor, EdgeColor, LineWidth, Alpha, Label,
// ZOrder.
func (a *Axes) FillBetween(x, y1, y2 any, opts ...Option) {
	xs, y1s := Values(x), Values(y1)
	mustSameLen("FillBetween", xs, y1s)
	var y2s []float64
	if y2 == nil {
		y2s = make([]float64, len(xs))
	} else {
		y2s = broadcast(Values(y2), len(xs), "FillBetween")
	}
	p := collect(opts)
	var c colors.RGBA
	if !p.setColor && !p.setFace {
		c = a.nextFillColor()
	}
	g := &polygons{style: newPatchStyle(p, c), label: p.label, z: 1}
	if p.setZorder {
		g.z = p.zorder
	}
	// Each run of selected points becomes one polygon.
	start := -1
	flush := func(end int) {
		if start < 0 {
			return
		}
		var pg []render.Pt
		for i := start; i < end; i++ {
			pg = append(pg, render.Pt{X: xs[i], Y: y1s[i]})
		}
		for i := end - 1; i >= start; i-- {
			pg = append(pg, render.Pt{X: xs[i], Y: y2s[i]})
		}
		g.polys = append(g.polys, pg)
		start = -1
	}
	for i := range xs {
		ok := !math.IsNaN(y1s[i]) && !math.IsNaN(y2s[i]) && (p.where == nil || p.where[i])
		if ok && start < 0 {
			start = i
		} else if !ok {
			flush(i)
		}
	}
	flush(len(xs))
	a.add(g)
}
