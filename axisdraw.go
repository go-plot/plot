// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
	"github.com/go-plot/plot/ticker"
)

// matplotlib's rcParams defaults, in points.
const (
	tickLabelSize   = 10.0 // xtick.labelsize "medium"
	majorTickSize   = 3.5
	minorTickSize   = 2.0
	majorTickWidth  = 0.8
	minorTickWidth  = 0.6
	tickPad         = 3.5 // xtick.major.pad
	minorTickPad    = 3.4 // xtick.minor.pad
	labelPad        = 4.0 // axes.labelpad
	offsetTextPad   = 3.0 // Axis.OFFSETTEXTPAD
	titlePad        = 6.0 // axes.titlepad
	titleSize       = 12.0
	labelSize       = 10.0 // axes.labelsize "medium"
	spineWidth      = 0.8  // axes.linewidth
	gridWidth       = 0.8
	defaultFontSize = 10.0
)

var (
	black     = colors.RGBA{A: 1}
	gridColor = colors.MustParse("#b0b0b0")
)

// line returns a two-point path.
func line(x0, y0, x1, y1 float64) *render.Path {
	var p render.Path
	p.MoveTo(x0, y0)
	p.LineTo(x1, y1)
	return &p
}

// drawAxisDecor draws grid lines, ticks, tick labels, offset texts and axis
// labels — what matplotlib's XAxis and YAxis artists draw, at z-order 1.5.
func (a *Axes) drawAxisDecor(r render.Renderer) {
	if !a.xHidden {
		a.drawXAxis(r)
	}
	a.drawYAxis(r)
}

func (a *Axes) drawXAxis(r render.Renderer) {
	f := a.fig
	ax := a.x
	if a.gridX {
		a.clipToAxes(r)
		for _, t := range ax.ticksOut.Major {
			x := a.xPx(t.Loc)
			r.DrawPath(line(x, a.px.y1, x, a.px.y0), render.Style{Stroke: gridColor, Width: f.pt(gridWidth), Cap: render.CapProjecting, Snap: true})
		}
		r.PopClip()
	}
	y := a.px.y1
	size := f.pt(tickLabelSize)
	labels := []box{}
	for _, ticks := range [][]tickInfo{tickInfos(ax.ticksOut.Major, true), tickInfos(ax.ticksOut.Minor, false)} {
		for _, t := range ticks {
			x := a.xPx(t.loc)
			r.DrawPath(line(x, y, x, y+f.pt(t.size)), render.Style{Stroke: black, Width: f.pt(t.width), Snap: true})
			if t.label != "" {
				b := f.drawText(r, textSpec{S: t.label, X: x, Y: y + f.pt(t.size+t.pad), Size: size, HA: "center", VA: "top", Color: black, Plain: ax.callerTicks(f)})
				labels = append(labels, b)
			}
		}
	}
	bottom := y
	for _, b := range labels {
		bottom = math.Max(bottom, b.y1)
	}
	if ax.ticksOut.Offset != "" {
		f.drawText(r, textSpec{S: ax.ticksOut.Offset, X: a.px.x1, Y: bottom + f.pt(offsetTextPad), Size: size, HA: "right", VA: "top", Color: black})
	}
	if ax.label.s != "" {
		// The spine's extent counts, and matplotlib extends it by the
		// length of the outward ticks.
		bottom = math.Max(bottom, y+f.pt(majorTickSize))
		f.drawText(r, textSpec{S: ax.label.s, X: (a.px.x0 + a.px.x1) / 2, Y: bottom + f.pt(labelPad), Size: f.pt(labelSize), HA: "center", VA: "top", Color: black, Plain: f.plain(ax.label.mode)})
	}
}

func (a *Axes) drawYAxis(r render.Renderer) {
	f := a.fig
	ax := a.y
	if a.gridY {
		a.clipToAxes(r)
		for _, t := range ax.ticksOut.Major {
			y := a.yPx(t.Loc)
			r.DrawPath(line(a.px.x0, y, a.px.x1, y), render.Style{Stroke: gridColor, Width: f.pt(gridWidth), Cap: render.CapProjecting, Snap: true})
		}
		r.PopClip()
	}
	x, dir, ha := a.px.x0, -1.0, "right"
	if ax.right {
		x, dir, ha = a.px.x1, 1, "left"
	}
	size := f.pt(tickLabelSize)
	var labels []box
	for _, ticks := range [][]tickInfo{tickInfos(ax.ticksOut.Major, true), tickInfos(ax.ticksOut.Minor, false)} {
		for _, t := range ticks {
			y := a.yPx(t.loc)
			r.DrawPath(line(x, y, x+dir*f.pt(t.size), y), render.Style{Stroke: black, Width: f.pt(t.width), Snap: true})
			if t.label != "" {
				b := f.drawText(r, textSpec{S: t.label, X: x + dir*f.pt(t.size+t.pad), Y: y, Size: size, HA: ha, VA: "center_baseline", Color: black, Plain: ax.callerTicks(f)})
				labels = append(labels, b)
			}
		}
	}
	if ax.ticksOut.Offset != "" {
		ox, oha := a.px.x0, "left"
		if ax.right {
			ox, oha = a.px.x1, "right"
		}
		b := f.drawText(r, textSpec{S: ax.ticksOut.Offset, X: ox, Y: a.px.y0 - f.pt(offsetTextPad), Size: size, HA: oha, VA: "baseline", Color: black})
		a.yOffsetTop = b.y0
		a.hasYOffset = true
	}
	if ax.label.s != "" {
		edge := x + dir*f.pt(majorTickSize)
		for _, b := range labels {
			if ax.right {
				edge = math.Max(edge, b.x1)
			} else {
				edge = math.Min(edge, b.x0)
			}
		}
		va := "bottom"
		if ax.right {
			va = "top"
		}
		f.drawText(r, textSpec{S: ax.label.s, X: edge + dir*f.pt(labelPad), Y: (a.px.y0 + a.px.y1) / 2, Size: f.pt(labelSize), HA: "center", VA: va, Rot: 90, Anchor: true, Color: black, Plain: f.plain(ax.label.mode)})
	}
}

// callerTicks reports whether the tick labels are the caller's and set
// without TeX. The formatters' labels are go-plot's own and stay math.
func (ax *axis) callerTicks(f *Figure) bool {
	return ax.setTicks && ax.tickLabels != nil && f.plain(ax.tickMode)
}

// tickInfo is a tick with the size of its kind.
type tickInfo struct {
	loc              float64
	label            string
	size, width, pad float64
}

func tickInfos(ts []ticker.Tick, major bool) []tickInfo {
	out := make([]tickInfo, len(ts))
	for i, t := range ts {
		out[i] = tickInfo{loc: t.Loc, label: t.Label, size: majorTickSize, width: majorTickWidth, pad: tickPad}
		if !major {
			out[i].size, out[i].width, out[i].pad = minorTickSize, minorTickWidth, minorTickPad
		}
	}
	return out
}

// drawSpines draws the four lines around the Axes box (z-order 2.5).
func (a *Axes) drawSpines(r render.Renderer) {
	if a.colorbar != nil || a.noFrame {
		return
	}
	st := render.Style{Stroke: black, Width: a.fig.pt(spineWidth), Cap: render.CapProjecting, Snap: true}
	b := a.px
	r.DrawPath(line(b.x0, b.y1, b.x0, b.y0), st)
	r.DrawPath(line(b.x1, b.y1, b.x1, b.y0), st)
	r.DrawPath(line(b.x0, b.y1, b.x1, b.y1), st)
	r.DrawPath(line(b.x0, b.y0, b.x1, b.y0), st)
}

// drawTitle draws the title centred above the Axes, its baseline 6 points
// above the box — higher if the y axis offset text would run into it.
func (a *Axes) drawTitle(r render.Renderer) {
	if a.title.s == "" {
		return
	}
	f := a.fig
	spec := textSpec{S: a.title.s, X: (a.px.x0 + a.px.x1) / 2, Y: a.px.y0 - f.pt(titlePad), Size: f.pt(titleSize), HA: "center", VA: "baseline", Color: black, Plain: f.plain(a.title.mode)}
	if a.hasYOffset {
		if p := f.layoutText(spec); p.bbox.y1 > a.yOffsetTop {
			spec.Y = a.yOffsetTop - f.pt(titlePad)
		}
	}
	f.drawText(r, spec)
}
