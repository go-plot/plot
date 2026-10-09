// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"fmt"
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// matplotlib's line defaults, in points.
const (
	defaultLineWidth   = 1.5
	defaultMarkerSize  = 6.0
	defaultMarkerEdgeW = 1.0
)

// lineStyle is everything about how a line is stroked and marked.
type lineStyle struct {
	color    colors.RGBA
	width    float64 // points
	dash     string  // "-", "--", "-.", ":", "" (none)
	marker   *markerShape
	ms, mew  float64
	mfc, mec colors.RGBA
}

// dashPattern returns matplotlib's dash pattern for a style, scaled by the
// line width (lines.scale_dashes), in points; nil for solid.
func dashPattern(style string, lw float64) []float64 {
	var base []float64
	switch style {
	case "--":
		base = []float64{3.7, 1.6}
	case "-.":
		base = []float64{6.4, 1.6, 1, 1.6}
	case ":":
		base = []float64{1, 1.65}
	default:
		return nil
	}
	out := make([]float64, len(base))
	for i, v := range base {
		out[i] = v * lw
	}
	return out
}

func validLineStyle(s string) string {
	switch s {
	case "-", "--", "-.", ":":
		return s
	case "", "None", "none", " ":
		return ""
	case "solid":
		return "-"
	case "dashed":
		return "--"
	case "dashdot":
		return "-."
	case "dotted":
		return ":"
	}
	panic(fmt.Sprintf("plot: unknown line style %q", s))
}

// newLineStyle resolves options against the defaults; c is the colour when
// none was given.
func newLineStyle(p props, c colors.RGBA, defaultLS string) lineStyle {
	if p.setColor {
		c = p.color
	}
	if p.setAlpha {
		c.A = p.alpha
	}
	ls := lineStyle{color: c, width: defaultLineWidth, dash: defaultLS, ms: defaultMarkerSize, mew: defaultMarkerEdgeW, mfc: c, mec: c}
	if p.lineWidth > 0 {
		ls.width = p.lineWidth
	}
	if p.setLineStyle {
		ls.dash = validLineStyle(p.lineStyle)
	}
	if p.setMarker && p.marker != "" && p.marker != "None" {
		ls.marker = getMarker(p.marker)
	}
	if p.markerSize > 0 {
		ls.ms = p.markerSize
	}
	if p.markerEdgeWidth > 0 {
		ls.mew = p.markerEdgeWidth
	}
	if p.setFace {
		ls.mfc = p.faceColor
	}
	if p.setEdge {
		ls.mec = p.edgeColor
	}
	return ls
}

// stroke returns the render style for the line itself.
func (ls lineStyle) stroke(f *Figure) render.Style {
	st := render.Style{Stroke: ls.color, Width: f.pt(ls.width), Join: render.JoinRound, Cap: render.CapProjecting}
	if d := dashPattern(ls.dash, ls.width); d != nil {
		for i := range d {
			d[i] = f.pt(d[i])
		}
		st.Dash, st.Cap = d, render.CapButt
	}
	return st
}

// drawMarkers draws the marker at every pixel position.
func (ls lineStyle) drawMarkers(f *Figure, r render.Renderer, pts []render.Pt) {
	if ls.marker == nil {
		return
	}
	st := render.Style{Stroke: ls.mec, Width: f.pt(ls.mew), Join: render.JoinMiter, Snap: true}
	if ls.marker.filled {
		st.Fill = ls.mfc
	} else {
		st.Stroke = ls.color
		st.Cap = render.CapButt
	}
	if ls.marker.pixel {
		st = render.Style{Fill: ls.color}
	}
	size := f.pt(ls.ms)
	for _, p := range pts {
		if math.IsNaN(p.X) || math.IsNaN(p.Y) {
			continue
		}
		// Agg draws markers at whole pixels.
		r.DrawPath(ls.marker.at(math.Round(p.X), math.Round(p.Y), size), st)
	}
}

// line2D is a polyline in data coordinates, as matplotlib's Line2D.
type line2D struct {
	x, y  []float64
	style lineStyle
	label string
	z     float64
	clip  bool
	// kind: 0 a data line, 1 axhline (x in axes fractions), 2 axvline.
	kind int
}

func (l *line2D) zorder() float64 { return l.z }

func (l *line2D) limits() lim {
	lm := emptyLim()
	for i := range l.x {
		switch l.kind {
		case 0:
			lm.addXY(l.x[i], l.y[i])
		case 1:
			lm.addY(l.y[i])
		case 2:
			lm.addX(l.x[i])
		}
	}
	return lm
}

func (l *line2D) sticky() (x, y []float64) { return nil, nil }

// pixels maps the line to figure pixels, NaN marking a break.
func (l *line2D) pixels(a *Axes) []render.Pt {
	pts := make([]render.Pt, len(l.x))
	for i := range l.x {
		switch l.kind {
		case 1:
			pts[i] = render.Pt{X: a.px.x0 + l.x[i]*(a.px.x1-a.px.x0), Y: a.yPx(l.y[i])}
		case 2:
			pts[i] = render.Pt{X: a.xPx(l.x[i]), Y: a.px.y1 - l.y[i]*(a.px.y1-a.px.y0)}
		default:
			x, y := l.x[i], l.y[i]
			if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) ||
				(a.x.log && x <= 0) || (a.y.log && y <= 0) {
				pts[i] = render.Pt{X: math.NaN(), Y: math.NaN()}
				continue
			}
			pts[i] = render.Pt{X: a.xPx(x), Y: a.yPx(y)}
		}
	}
	return pts
}

// polylinePath builds a path from pixel points, breaking at NaN and
// simplifying each run as matplotlib does.
func polylinePath(pts []render.Pt) *render.Path {
	var p render.Path
	run := func(seg []render.Pt) {
		seg = simplify(seg)
		for i, q := range seg {
			if i == 0 {
				p.MoveTo(q.X, q.Y)
			} else {
				p.LineTo(q.X, q.Y)
			}
		}
	}
	start := 0
	for i, q := range pts {
		if math.IsNaN(q.X) {
			run(pts[start:i])
			start = i + 1
		}
	}
	run(pts[start:])
	return &p
}

func (l *line2D) draw(a *Axes, r render.Renderer) {
	pts := l.pixels(a)
	if l.clip {
		a.clipToAxes(r)
		defer r.PopClip()
	}
	if l.style.dash != "" {
		st := l.style.stroke(a.fig)
		st.Snap = true
		r.DrawPath(polylinePath(pts), st)
	}
	l.style.drawMarkers(a.fig, r, pts)
}

func (l *line2D) legendEntry() *legendEntry {
	if l.label == "" || l.label[0] == '_' {
		return nil
	}
	return &legendEntry{label: l.label, line: &l.style}
}

// Plot draws y against x as lines and/or markers, as matplotlib's ax.plot
// does. x may be nil, meaning 0, 1, 2, ... . Without a colour option the
// line takes the next colour of the cycle.
//
// Options: Color, RGBA, Alpha, LineWidth, LineStyle, Marker, MarkerSize,
// MarkerEdgeWidth, FaceColor (marker fill), EdgeColor (marker edge), Label,
// ZOrder, Clip.
func (a *Axes) Plot(x, y any, opts ...Option) {
	ys := Values(y)
	xs := Values(x)
	if xs == nil {
		xs = indices(len(ys))
	}
	mustSameLen("Plot", xs, ys)
	p := collect(opts)
	var c colors.RGBA
	if !p.setColor {
		c = a.nextLineColor()
	}
	l := &line2D{x: xs, y: ys, style: newLineStyle(p, c, "-"), label: p.label, z: 2, clip: p.clip != 2}
	if p.setZorder {
		l.z = p.zorder
	}
	a.add(l)
}

// Axhline draws a horizontal line across the Axes at y, as matplotlib's
// ax.axhline: it spans the full width whatever the x limits, and only y
// takes part in autoscaling. Its colour is C0 unless given; it does not
// advance the cycle.
//
// Options: as Plot.
func (a *Axes) Axhline(y float64, opts ...Option) {
	p := collect(opts)
	l := &line2D{x: []float64{0, 1}, y: []float64{y, y}, style: newLineStyle(p, colors.CycleColor(0), "-"), label: p.label, z: 2, clip: p.clip != 2, kind: 1}
	if p.setZorder {
		l.z = p.zorder
	}
	a.add(l)
}

// Axvline draws a vertical line across the Axes at x, as matplotlib's
// ax.axvline.
//
// Options: as Plot.
func (a *Axes) Axvline(x float64, opts ...Option) {
	p := collect(opts)
	l := &line2D{x: []float64{x, x}, y: []float64{0, 1}, style: newLineStyle(p, colors.CycleColor(0), "-"), label: p.label, z: 2, clip: p.clip != 2, kind: 2}
	if p.setZorder {
		l.z = p.zorder
	}
	a.add(l)
}

// errorBars is the bars (and caps) of an ErrorBar.
type errorBars struct {
	segs   [][4]float64 // x0, y0, x1, y1 in data
	caps   []render.Pt  // cap centres in data
	capV   []bool       // cap is vertical (for x errors)
	style  lineStyle
	capPts float64
}

func (e *errorBars) zorder() float64 { return 2 }

func (e *errorBars) limits() lim {
	lm := emptyLim()
	for _, s := range e.segs {
		lm.addXY(s[0], s[1])
		lm.addXY(s[2], s[3])
	}
	return lm
}

func (e *errorBars) sticky() (x, y []float64) { return nil, nil }

func (e *errorBars) legendEntry() *legendEntry { return nil }

func (e *errorBars) draw(a *Axes, r render.Renderer) {
	a.clipToAxes(r)
	defer r.PopClip()
	st := render.Style{Stroke: e.style.color, Width: a.fig.pt(e.style.width), Cap: render.CapProjecting, Snap: true}
	var p render.Path
	for _, s := range e.segs {
		x0, y0 := a.toPx(s[0], s[1])
		x1, y1 := a.toPx(s[2], s[3])
		p.MoveTo(x0, y0)
		p.LineTo(x1, y1)
	}
	r.DrawPath(&p, st)
	if e.capPts <= 0 {
		return
	}
	half := a.fig.pt(e.capPts)
	cst := render.Style{Stroke: e.style.color, Width: a.fig.pt(e.style.width), Cap: render.CapButt, Snap: true}
	for i, c := range e.caps {
		x, y := a.toPx(c.X, c.Y)
		x, y = math.Round(x), math.Round(y)
		if e.capV[i] {
			r.DrawPath(line(x, y-half, x, y+half), cst)
		} else {
			r.DrawPath(line(x-half, y, x+half, y), cst)
		}
	}
}

// ErrorBar draws y against x with error bars, as matplotlib's ax.errorbar:
// a line in the next cycle colour, and symmetric bars of YErr and/or XErr
// (one value for all points, or one per point) in the same colour, with
// caps CapSize points long on each side if CapSize is given.
//
// Options: XErr, YErr, CapSize, ErrorColor, and those of Plot.
func (a *Axes) ErrorBar(x, y any, opts ...Option) {
	xs, ys := Values(x), Values(y)
	mustSameLen("ErrorBar", xs, ys)
	p := collect(opts)
	var c colors.RGBA
	if !p.setColor {
		c = a.nextLineColor()
	}
	ls := newLineStyle(p, c, "-")
	bars := &errorBars{style: ls, capPts: p.capsize}
	if p.setE {
		bars.style.color = p.ecolor
	}
	at := func(v []float64, i int) float64 {
		if len(v) == 1 {
			return v[0]
		}
		return v[i]
	}
	for i := range xs {
		if p.yerr != nil {
			e := at(p.yerr, i)
			bars.segs = append(bars.segs, [4]float64{xs[i], ys[i] - e, xs[i], ys[i] + e})
			bars.caps = append(bars.caps, render.Pt{X: xs[i], Y: ys[i] - e}, render.Pt{X: xs[i], Y: ys[i] + e})
			bars.capV = append(bars.capV, false, false)
		}
		if p.xerr != nil {
			e := at(p.xerr, i)
			bars.segs = append(bars.segs, [4]float64{xs[i] - e, ys[i], xs[i] + e, ys[i]})
			bars.caps = append(bars.caps, render.Pt{X: xs[i] - e, Y: ys[i]}, render.Pt{X: xs[i] + e, Y: ys[i]})
			bars.capV = append(bars.capV, true, true)
		}
	}
	a.add(bars)
	l := &line2D{x: xs, y: ys, style: ls, label: p.label, z: 2.1, clip: true}
	a.add(l)
}
