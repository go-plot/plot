// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"
	"slices"
	"sort"
	"strconv"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
	"github.com/go-plot/plot/ticker"
)

// rect is a rectangle in figure fractions, y up from the bottom, as
// matplotlib positions axes.
type rect struct{ x, y, w, h float64 }

// Axes is one plotting area: two axes, the things plotted on them, a title
// and a legend. Create it with Figure.Subplots, Subplots or Figure.AddAxes.
type Axes struct {
	fig       *Figure
	pos       rect // as placed
	active    rect // after the aspect constraint
	px        box  // active, in pixels
	Facecolor colors.RGBA

	x, y      *axis
	xHidden   bool // a twin's x axis is drawn by its parent
	frameOff  bool // patch (background) not drawn: twins
	noFrame   bool // neither patch nor spines: matplotlib's frame_on=False (Pie)
	artists   []artist
	lims      []lim // artists[i].limits(), computed once when added
	title     string
	legend    *legendSpec
	gridX     bool
	gridY     bool
	lineCycle int
	fillCycle int
	aspect    float64 // 0 auto; data units ratio (1 for "equal")
	boxAspect float64 // 0 none; height/width of the axes box
	anchor    [2]float64
	colorbar  *colorbarSpec
	cell      *cell // where Subplots put it, or nil
	cbar      *Axes // its colorbar, if any

	hasYOffset bool    // set while drawing: the y offset text exists
	yOffsetTop float64 // and its top, in pixels
}

// axis is the state of one axis. A twin shares its x axis object with its
// parent, which is how the limits stay shared.
type axis struct {
	name       byte // 'x' or 'y'
	log        bool
	vmin, vmax float64
	auto       bool // autoscale on: limits follow the data
	label      string
	right      bool // ticks and label on the right (y) or top (x)
	owners     []*Axes
	ticks      []float64 // manual ticks, or nil
	tickLabels []string
	setTicks   bool
	ticksOut   ticker.Ticks // computed at layout
	inverted   bool
}

func newAxes(f *Figure, r rect) *Axes {
	a := &Axes{
		fig:       f,
		pos:       r,
		Facecolor: colors.RGBA{R: 1, G: 1, B: 1, A: 1},
		x:         &axis{name: 'x', vmin: 0, vmax: 1, auto: true},
		y:         &axis{name: 'y', vmin: 0, vmax: 1, auto: true},
		anchor:    [2]float64{0.5, 0.5},
	}
	a.x.owners = []*Axes{a}
	a.y.owners = []*Axes{a}
	return a
}

// Figure returns the figure the Axes is on.
func (a *Axes) Figure() *Figure { return a.fig }

// TwinX returns a new Axes sharing this one's x axis, with its y axis on the
// right, as matplotlib's ax.twinx() does. It has its own colour cycle, which
// starts again at C0, as in matplotlib.
func (a *Axes) TwinX() *Axes {
	t := a.fig.AddAxes(a.pos.x, a.pos.y, a.pos.w, a.pos.h)
	t.x = a.x
	t.cell = a.cell
	a.x.owners = append(a.x.owners, t)
	t.xHidden = true
	t.frameOff = true
	t.y.right = true
	return t
}

// SetTitle sets the title above the Axes.
func (a *Axes) SetTitle(s string) { a.title = s }

// SetXLabel sets the label of the x axis.
func (a *Axes) SetXLabel(s string) { a.x.label = s }

// SetYLabel sets the label of the y axis.
func (a *Axes) SetYLabel(s string) { a.y.label = s }

// SetXLim fixes the x view limits and turns x autoscaling off. Equal limits
// are widened, as matplotlib widens them.
func (a *Axes) SetXLim(lo, hi float64) { a.x.setLim(lo, hi) }

// SetYLim fixes the y view limits and turns y autoscaling off.
func (a *Axes) SetYLim(lo, hi float64) { a.y.setLim(lo, hi) }

// XLim returns the x view limits as of the last draw (or as set).
func (a *Axes) XLim() (float64, float64) { return a.x.lims() }

// YLim returns the y view limits as of the last draw (or as set).
func (a *Axes) YLim() (float64, float64) { return a.y.lims() }

// SetXScale sets the x scale: "linear" or "log".
func (a *Axes) SetXScale(s string) { a.x.log = s == "log" }

// SetYScale sets the y scale: "linear" or "log".
func (a *Axes) SetYScale(s string) { a.y.log = s == "log" }

// SetXTicks fixes the x tick locations and, when labels are given, their
// labels.
func (a *Axes) SetXTicks(locs []float64, labels ...string) { a.x.setTicksTo(locs, labels) }

// SetYTicks fixes the y tick locations and, when labels are given, their
// labels.
func (a *Axes) SetYTicks(locs []float64, labels ...string) { a.y.setTicksTo(locs, labels) }

// Grid turns the major grid lines of both axes on or off.
func (a *Axes) Grid(on bool) { a.gridX, a.gridY = on, on }

// SetAspect sets the ratio of y units to x units on screen: 1 for "equal",
// 0 for "auto". The Axes box shrinks to honour it, as with matplotlib's
// adjustable="box".
func (a *Axes) SetAspect(r float64) { a.aspect = r }

func (ax *axis) setLim(lo, hi float64) {
	inv := lo > hi
	if ax.log {
		lo, hi = logNonsingular(lo, hi, ax.minPos())
	} else {
		lo, hi = ticker.Nonsingular(lo, hi, 0.05, 1e-15, true)
	}
	ax.vmin, ax.vmax, ax.inverted, ax.auto = lo, hi, inv, false
}

func (ax *axis) lims() (float64, float64) {
	if ax.inverted {
		return ax.vmax, ax.vmin
	}
	return ax.vmin, ax.vmax
}

func (ax *axis) setTicksTo(locs []float64, labels []string) {
	ax.ticks = append([]float64(nil), locs...)
	ax.tickLabels = labels
	ax.setTicks = true
}

// lim is a data-limits accumulator: the bounding box of everything an
// artist draws, and the smallest positive value (for log scales).
type lim struct {
	x0, x1, y0, y1 float64
	minPosX        float64
	minPosY        float64
}

func emptyLim() lim {
	inf := math.Inf(1)
	return lim{x0: inf, x1: -inf, y0: inf, y1: -inf, minPosX: inf, minPosY: inf}
}

// finite reports whether v is neither NaN nor infinite (v-v is 0 only then).
func finite(v float64) bool { return v-v == 0 }

// The comparisons below are plain ones, not math.Min: the values are known
// finite, and a million points go through here.
func (l *lim) addX(v float64) {
	if !finite(v) {
		return
	}
	if v < l.x0 {
		l.x0 = v
	}
	if v > l.x1 {
		l.x1 = v
	}
	if v > 0 && v < l.minPosX {
		l.minPosX = v
	}
}

func (l *lim) addY(v float64) {
	if !finite(v) {
		return
	}
	if v < l.y0 {
		l.y0 = v
	}
	if v > l.y1 {
		l.y1 = v
	}
	if v > 0 && v < l.minPosY {
		l.minPosY = v
	}
}

func (l *lim) addXY(x, y float64) {
	if !finite(x) || !finite(y) {
		return
	}
	l.addX(x)
	l.addY(y)
}

// minPos returns the smallest positive data value on the axis, over every
// Axes that shares it.
func (ax *axis) minPos() float64 {
	m := math.Inf(1)
	for _, o := range ax.owners {
		for _, l := range o.lims {
			if ax.name == 'x' {
				m = math.Min(m, l.minPosX)
			} else {
				m = math.Min(m, l.minPosY)
			}
		}
	}
	return m
}

// logNonsingular is LogLocator.nonsingular.
func logNonsingular(vmin, vmax, minpos float64) (float64, float64) {
	if vmin > vmax {
		vmin, vmax = vmax, vmin
	}
	switch {
	case math.IsInf(vmin, 0) || math.IsInf(vmax, 0) || math.IsNaN(vmin) || math.IsNaN(vmax):
		return 1, 10
	case vmax <= 0:
		return 1, 10
	}
	if math.IsInf(minpos, 0) {
		minpos = 1e-300
	}
	if vmin <= 0 {
		vmin = minpos
	}
	if vmin == vmax {
		// matplotlib's _decade_less and _decade_greater: the decade
		// strictly below and above.
		e := math.Log10(vmin)
		lo, hi := math.Floor(e), math.Ceil(e)
		if lo == hi {
			lo, hi = lo-1, hi+1
		}
		vmin, vmax = math.Pow(10, lo), math.Pow(10, hi)
	}
	return vmin, vmax
}

// autoscale is matplotlib's autoscale_view for one axis: the data limits of
// every Axes sharing the axis, widened by a 5% margin in scale space but
// never across a sticky edge (the base of a bar, the edge of an image).
func (ax *axis) autoscale() {
	if !ax.auto {
		return
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	var stickies []float64
	for _, o := range ax.owners {
		for i, art := range o.artists {
			l := o.lims[i]
			if ax.name == 'x' {
				lo, hi = math.Min(lo, l.x0), math.Max(hi, l.x1)
			} else {
				lo, hi = math.Min(lo, l.y0), math.Max(hi, l.y1)
			}
			sx, sy := art.sticky()
			if ax.name == 'x' {
				stickies = append(stickies, sx...)
			} else {
				stickies = append(stickies, sy...)
			}
		}
	}
	if lo > hi { // no data: matplotlib keeps the initial (0, 1)
		return
	}
	minpos := ax.minPos()
	if ax.log {
		lo, hi = logNonsingular(lo, hi, minpos)
		var pos []float64
		for _, s := range stickies {
			if s > 0 {
				pos = append(pos, s)
			}
		}
		stickies = pos
	} else {
		lo, hi = ticker.Nonsingular(lo, hi, 0.05, 1e-15, true)
	}
	sort.Float64s(stickies)
	tol := 1e-5 * math.Abs(hi-lo)
	i0 := sort.SearchFloat64s(stickies, lo+tol) - 1
	i1 := sort.SearchFloat64s(stickies, hi-tol)
	tr, inv := ax.forward, ax.inverse
	t0, t1 := tr(lo), tr(hi)
	delta := (t1 - t0) * 0.05
	if math.IsInf(delta, 0) || math.IsNaN(delta) {
		delta = 0
	}
	lo, hi = inv(t0-delta), inv(t1+delta)
	if i0 >= 0 {
		lo = math.Max(lo, stickies[i0])
	}
	if i1 < len(stickies) {
		hi = math.Min(hi, stickies[i1])
	}
	if ax.log {
		lo, hi = logNonsingular(lo, hi, minpos)
	} else {
		lo, hi = ticker.Nonsingular(lo, hi, 1e-12, 1e-13, true)
	}
	ax.vmin, ax.vmax = lo, hi
}

// forward is the scale transform: identity or log10 (non-positive values
// clip to -1000, as matplotlib's LogTransform with nonpositive="clip").
func (ax *axis) forward(v float64) float64 {
	if !ax.log {
		return v
	}
	if v <= 0 {
		return -1000
	}
	return math.Log10(v)
}

func (ax *axis) inverse(v float64) float64 {
	if !ax.log {
		return v
	}
	return math.Pow(10, v)
}

// layout settles the Axes before drawing: limits, aspect, pixel box, ticks.
func (a *Axes) layout() {
	if a.x.owners[0] == a {
		a.x.autoscale()
	}
	a.y.autoscale()
	a.active = a.pos
	if a.aspect > 0 {
		ratio := (a.y.forward(a.y.vmax) - a.y.forward(a.y.vmin)) / (a.x.forward(a.x.vmax) - a.x.forward(a.x.vmin))
		a.active = a.shrinkToAspect(a.aspect * ratio)
	} else if a.boxAspect > 0 {
		a.active = a.shrinkToAspect(a.boxAspect)
	}
	w, h := a.fig.PixelSize()
	W, H := float64(w), float64(h)
	a.px = box{a.active.x * W, H - (a.active.y+a.active.h)*H, (a.active.x + a.active.w) * W, H - a.active.y*H}
	a.x.computeTicks(a)
	a.y.computeTicks(a)
}

// shrinkToAspect is matplotlib's Bbox.shrunk_to_aspect followed by
// anchored: the largest box of the given height/width ratio (on screen)
// that fits the position, placed at the anchor.
func (a *Axes) shrinkToAspect(boxAspect float64) rect {
	figAspect := a.fig.Height / a.fig.Width
	w, h := a.pos.w, a.pos.h
	H := w * boxAspect / figAspect
	W := w
	if H > h {
		W = h * figAspect / boxAspect
		H = h
	}
	return rect{
		x: a.pos.x + (a.pos.w-W)*a.anchor[0],
		y: a.pos.y + (a.pos.h-H)*a.anchor[1],
		w: W, h: H,
	}
}

// tickSpace is matplotlib's Axis.get_tick_space: how many ticks fit, with
// labels of 10 points assumed three times (x) or twice (y) as long as high.
func (ax *axis) tickSpace(a *Axes) int {
	length := a.px.x1 - a.px.x0
	k := 3.0
	if ax.name == 'y' {
		length = a.px.y1 - a.px.y0
		k = 2
	}
	return int(math.Floor(length * 72 / a.fig.DPI / (tickLabelSize * k)))
}

func (ax *axis) computeTicks(a *Axes) {
	if ax.setTicks {
		var out ticker.Ticks
		f := ticker.NewScalarFormatter()
		labels := ticker.FormatTicks(f, ax.ticks, ax.vmin, ax.vmax)
		if ax.tickLabels != nil {
			labels = ax.tickLabels
		}
		for i, l := range ax.ticks {
			if l >= ax.vmin && l <= ax.vmax {
				lab := ""
				if i < len(labels) {
					lab = labels[i]
				}
				out.Major = append(out.Major, ticker.Tick{Loc: l, Label: lab})
			}
		}
		out.Located = out.Major
		if ax.tickLabels == nil {
			out.Offset = f.Offset()
		}
		ax.ticksOut = out
		return
	}
	var cfg ticker.Config
	if ax.log {
		cfg = ticker.LogConfig(ax.tickSpace(a), ax.minPos())
	} else {
		cfg = ticker.LinearConfig(ax.tickSpace(a))
	}
	ax.ticksOut = cfg.Compute(ax.vmin, ax.vmax)
}

// toPx maps data coordinates to figure pixels.
func (a *Axes) toPx(x, y float64) (float64, float64) {
	return a.xPx(x), a.yPx(y)
}

func (a *Axes) xPx(x float64) float64 {
	lo, hi := a.x.forward(a.x.vmin), a.x.forward(a.x.vmax)
	t := (a.x.forward(x) - lo) / (hi - lo)
	if a.x.inverted {
		t = 1 - t
	}
	return a.px.x0 + t*(a.px.x1-a.px.x0)
}

func (a *Axes) yPx(y float64) float64 {
	lo, hi := a.y.forward(a.y.vmin), a.y.forward(a.y.vmax)
	t := (a.y.forward(y) - lo) / (hi - lo)
	if a.y.inverted {
		t = 1 - t
	}
	return a.px.y1 - t*(a.px.y1-a.px.y0)
}

// artist is anything drawn in an Axes.
type artist interface {
	zorder() float64
	limits() lim
	sticky() (x, y []float64)
	draw(a *Axes, r render.Renderer)
	legendEntry() *legendEntry
}

// draw paints the Axes: background, then everything by z-order (the axes'
// own ticks, grid and labels at 1.5 and the spines at 2.5, as in
// matplotlib), then the title and legend.
func (a *Axes) draw(r render.Renderer, n int) {
	r.BeginGroup("axes_" + strconv.Itoa(n))
	if !a.frameOff && !a.noFrame {
		var bg render.Path
		bg.Rect(a.px.x0, a.px.y0, a.px.x1-a.px.x0, a.px.y1-a.px.y0)
		r.DrawPath(&bg, render.Style{Fill: a.Facecolor, Snap: true})
	}
	type item struct {
		z    float64
		draw func()
	}
	var items []item
	for _, art := range a.artists {
		items = append(items, item{art.zorder(), func() { art.draw(a, r) }})
	}
	items = append(items, item{1.5, func() { a.drawAxisDecor(r) }})
	items = append(items, item{2.5, func() { a.drawSpines(r) }})
	slices.SortStableFunc(items, func(p, q item) int {
		switch {
		case p.z < q.z:
			return -1
		case p.z > q.z:
			return 1
		}
		return 0
	})
	for _, it := range items {
		it.draw()
	}
	a.drawTitle(r)
	if a.legend != nil {
		a.drawLegend(r)
	}
	r.EndGroup()
}

// add appends an artist. Its data limits are taken now: data given to a
// plot does not change afterwards, and a million points are not walked
// once per axis per draw.
func (a *Axes) add(art artist) {
	a.artists = append(a.artists, art)
	a.lims = append(a.lims, art.limits())
}

// clipToAxes pushes the Axes box as the clip.
func (a *Axes) clipToAxes(r render.Renderer) {
	r.PushClip(a.px.x0, a.px.y0, a.px.x1-a.px.x0, a.px.y1-a.px.y0)
}

// nextLineColor and nextFillColor are matplotlib's two property cyclers:
// lines (Plot, ErrorBar, Hist) and filled things (Bar, Scatter,
// FillBetween) each start at C0.
func (a *Axes) nextLineColor() colors.RGBA {
	c := colors.CycleColor(a.lineCycle)
	a.lineCycle++
	return c
}

func (a *Axes) nextFillColor() colors.RGBA {
	c := colors.CycleColor(a.fillCycle)
	a.fillCycle++
	return c
}
