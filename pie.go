// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// The data Pie refuses, as matplotlib's ax.pie raises ValueError for it.
// They are errors, not panics: wedge sizes usually come from data.
var (
	ErrPieNegative  = errors.New("plot: Pie: wedge sizes must be non-negative")
	ErrPieNonFinite = errors.New("plot: Pie: wedge sizes must be finite numbers")
	ErrPieZero      = errors.New("plot: Pie: all wedge sizes are zero")
)

// Labels names the wedges of a Pie, one string per wedge: drawn next to
// each wedge and used by Legend.
func Labels(ls ...string) Option { return func(p *props) { p.labels = ls } }

// AutoPct labels each wedge of a Pie with its percentage, formatted by
// fmt.Sprintf(format, pct): "%1.1f%%" gives "33.3%", as matplotlib's
// autopct="%1.1f%%" does (Go's verbs, not Python's: use %.0f, not %d).
func AutoPct(format string) Option {
	return func(p *props) { p.autopct = func(pct float64) string { return fmt.Sprintf(format, pct) } }
}

// AutoPctFunc labels each wedge of a Pie with f(pct), as matplotlib's
// callable autopct. Returning "" draws nothing, which is how to leave tiny
// wedges unlabelled.
func AutoPctFunc(f func(pct float64) string) Option { return func(p *props) { p.autopct = f } }

// PctDistance sets where the AutoPct labels go, as a fraction of the radius
// from the centre (default 0.6; a Donut defaults to the middle of its ring).
func PctDistance(d float64) Option {
	return func(p *props) { p.pctDistance, p.setPctDistance = d, true }
}

// LabelDistance sets where the wedge labels go, as a fraction of the radius
// (default 1.1). NaN draws no labels but keeps them for the legend, as
// matplotlib's labeldistance=None.
func LabelDistance(d float64) Option {
	return func(p *props) { p.labelDistance, p.setLabelDistance = d, true }
}

// StartAngle rotates the start of the first wedge, in degrees
// counter-clockwise from the positive x axis (matplotlib's startangle,
// default 0). StartAngle(90) with CounterClock(false) starts at 12 o'clock
// and runs clockwise, as go-widgets/toolkit's PieChart does.
func StartAngle(deg float64) Option { return func(p *props) { p.startAngle = deg } }

// CounterClock sets the direction of the wedges (default true, as in
// matplotlib).
func CounterClock(on bool) Option { return func(p *props) { p.clockwise = !on } }

// Explode moves each wedge out from the centre along its middle, by the
// given distance in data units (matplotlib's explode).
func Explode(d ...float64) Option { return func(p *props) { p.explode = d } }

// Radius sets the radius of a Pie (default 1).
func Radius(r float64) Option { return func(p *props) { p.radius, p.setRadius = r, true } }

// Center sets the centre of a Pie in data coordinates (default 0, 0).
func Center(x, y float64) Option { return func(p *props) { p.center = [2]float64{x, y} } }

// WedgeWidth makes a ring: each wedge runs from radius-w to the radius
// (matplotlib's wedgeprops=dict(width=w)). 0 is a full wedge.
func WedgeWidth(w float64) Option { return func(p *props) { p.wedgeWidth = w } }

// RotateLabels turns each wedge label to the angle of its wedge.
func RotateLabels() Option { return func(p *props) { p.rotateLabels = true } }

// Normalize(false) draws the sizes as fractions of a full turn, leaving a
// gap when they sum to less than 1 (matplotlib's normalize=False).
func Normalize(on bool) Option { return func(p *props) { p.noNormalize = !on } }

// TextColor sets the colour of a Pie's labels and percentages (matplotlib's
// textprops=dict(color=...)).
func TextColor(c string) Option {
	col := colors.MustParse(c)
	return func(p *props) { p.textColor, p.setTextColor = col, true }
}

// PlainText sets the labels as they are, with no TeX: "$5 and $6" stays
// two dollar signs, and no label reaches the TeX typesetter. Use it for
// labels that come from data.
func PlainText() Option { return func(p *props) { p.plain = true } }

// SpreadLabels moves the outer labels of a Pie apart vertically where they
// would overlap (thin wedges side by side), and draws a leader line from
// each moved label to its wedge. matplotlib has no such option: without it,
// labels are where matplotlib puts them.
func SpreadLabels() Option { return func(p *props) { p.spreadLabels = true } }

// donut marks a call through Donut.
func donut() Option { return func(p *props) { p.donut = true } }

// wedge is matplotlib's patches.Wedge: a sector of a disc, or of a ring
// when width > 0, from theta1 to theta2 degrees counter-clockwise.
type wedge struct {
	cx, cy, r      float64
	theta1, theta2 float64
	width          float64
	style          patchStyle
	label          string
	plain          bool
	z              float64
}

// Path codes, as matplotlib's Path numbers them.
const (
	codeMoveTo = 1
	codeLineTo = 2
	codeCurve4 = 4
	codeClose  = 79
)

// unitArc is matplotlib's Path.arc(theta1, theta2): the unit circle from
// theta1 to theta2 degrees in cubic Bézier segments of at most a quarter
// turn, as vertices (a move, then three per segment).
func unitArc(theta1, theta2 float64) []render.Pt {
	const halfpi = math.Pi * 0.5
	nTurns := (theta2 - theta1) / 360
	nearest := math.RoundToEven(nTurns)
	// Every product is rounded on its own (float64(...)), as NumPy rounds
	// it: Go fuses x*y-z into one rounding on arm64, ppc64le, s390x, riscv64
	// and loong64, and e2-e1 of an empty wedge then comes out a few ulps
	// above 0, which doubles the segment count.
	eta2 := theta2 - float64(360*math.Floor(nTurns))
	if nearest != 0 && math.Abs(nTurns-nearest) <= 1e-12 {
		eta2 = theta1 + 360
	}
	e1, e2 := float64(theta1*(math.Pi/180)), float64(eta2*(math.Pi/180))
	n := int(math.Pow(2, math.Ceil((e2-e1)/halfpi)))
	deta := (e2 - e1) / float64(n)
	t := math.Tan(0.5 * deta)
	alpha := float64(math.Sin(deta)*(math.Sqrt(float64(4.0+float64(3.0*t*t)))-1)) / 3.0
	steps := linspace(e1, e2, n+1)
	v := make([]render.Pt, 0, 3*n+1)
	v = append(v, render.Pt{X: math.Cos(steps[0]), Y: math.Sin(steps[0])})
	for i := range n {
		xA, yA := math.Cos(steps[i]), math.Sin(steps[i])
		xB, yB := math.Cos(steps[i+1]), math.Sin(steps[i+1])
		v = append(v,
			render.Pt{X: xA + float64(alpha*-yA), Y: yA + float64(alpha*xA)},
			render.Pt{X: xB - float64(alpha*-yB), Y: yB - float64(alpha*xB)},
			render.Pt{X: xB, Y: yB})
	}
	return v
}

// vertices is Wedge._recompute_path: the path in data coordinates, with
// matplotlib's codes.
func (w *wedge) vertices() ([]render.Pt, []uint8) {
	t1, t2 := w.theta1, w.theta2
	connector := uint8(codeLineTo)
	if math.Abs((t2-t1)-360) <= 1e-12 {
		t1, t2, connector = 0, 360, codeMoveTo
	}
	arc := unitArc(t1, t2)
	codes := make([]uint8, len(arc))
	codes[0] = codeMoveTo
	for i := 1; i < len(codes); i++ {
		codes[i] = codeCurve4
	}
	var v []render.Pt
	var c []uint8
	if w.width > 0 {
		v = append(v, arc...)
		for i := len(arc) - 1; i >= 0; i-- {
			v = append(v, render.Pt{X: arc[i].X * (w.r - w.width) / w.r, Y: arc[i].Y * (w.r - w.width) / w.r})
		}
		v = append(v, render.Pt{})
		c = append(append(append(append(c, codes...), connector), codes[1:]...), codeClose)
	} else {
		v = append(append(v, arc...), render.Pt{}, render.Pt{})
		c = append(append(c, codes...), connector, codeClose)
	}
	for i := range v {
		v[i] = render.Pt{X: float64(v[i].X*w.r) + w.cx, Y: float64(v[i].Y*w.r) + w.cy}
	}
	return v, c
}

func (w *wedge) zorder() float64          { return w.z }
func (w *wedge) sticky() (x, y []float64) { return nil, nil }

func (w *wedge) limits() lim {
	lm := emptyLim()
	v, _ := w.vertices()
	for _, q := range v {
		lm.addXY(q.X, q.Y)
	}
	return lm
}

// pixels returns the path's vertices in figure pixels.
func (w *wedge) pixels(a *Axes) []render.Pt {
	v, _ := w.vertices()
	for i, q := range v {
		v[i].X, v[i].Y = a.toPx(q.X, q.Y)
	}
	return v
}

// draw paints the wedge, unclipped (matplotlib's pie sets clip_on=False).
func (w *wedge) draw(a *Axes, r render.Renderer) {
	v := w.pixels(a)
	_, codes := w.vertices()
	var p render.Path
	for i := 0; i < len(v); i++ {
		switch codes[i] {
		case codeMoveTo:
			p.MoveTo(v[i].X, v[i].Y)
		case codeLineTo:
			p.LineTo(v[i].X, v[i].Y)
		case codeCurve4:
			p.CubicTo(v[i].X, v[i].Y, v[i+1].X, v[i+1].Y, v[i+2].X, v[i+2].Y)
			i += 2
		default:
			p.Close()
		}
	}
	st := w.style.render(a.fig)
	st.Snap = false
	r.DrawPath(&p, st)
}

func (w *wedge) legendEntry() *legendEntry {
	if w.label == "" || w.label[0] == '_' {
		return nil
	}
	return &legendEntry{label: w.label, patch: &w.style, plain: w.plain}
}

// pieFractions checks the sizes as matplotlib does — negatives first, then
// non-finite values, then a zero sum — and returns each wedge's fraction of
// a turn. A sum that overflows is taken after scaling by the largest size,
// where matplotlib would draw every wedge empty.
func pieFractions(xs []float64, normalize bool) ([]float64, error) {
	for _, v := range xs {
		if v < 0 {
			return nil, ErrPieNegative
		}
	}
	sum, max := 0.0, 0.0
	for _, v := range xs {
		if !finite(v) {
			return nil, ErrPieNonFinite
		}
		sum += v
		max = math.Max(max, v)
	}
	if sum == 0 {
		return nil, ErrPieZero
	}
	fracs := make([]float64, len(xs))
	if !normalize {
		if sum > 1 {
			return nil, errors.New("plot: Pie: cannot draw an unnormalized pie with sum(x) > 1")
		}
		copy(fracs, xs)
		return fracs, nil
	}
	scale := 1.0
	if math.IsInf(sum, 1) {
		scale, sum = max, 0
		for _, v := range xs {
			sum += v / max
		}
	}
	for i, v := range xs {
		fracs[i] = v / scale / sum
	}
	return fracs, nil
}

// Pie draws a pie chart of x, as matplotlib's ax.pie: wedge i covers
// x[i]/sum(x) of a turn, counter-clockwise from the positive x axis, in the
// colours of the fill cycle; labels sit at 1.1 radii, percentages (AutoPct)
// at 0.6. The Axes gets an equal aspect, no frame, no ticks and limits of
// ±1.25 around the centre, as matplotlib sets them.
//
// x is any data Values accepts. Negative, NaN or infinite sizes, or sizes
// that are all zero, return an error and draw nothing, as matplotlib raises
// ValueError. A zero size is a wedge of no width.
//
// The defaults are matplotlib's, not go-widgets/toolkit's PieChart (which
// starts at 12 o'clock and runs clockwise): StartAngle(90) and
// CounterClock(false) give the toolkit's layout.
//
// Options: Labels, AutoPct, AutoPctFunc, PctDistance, LabelDistance,
// StartAngle, CounterClock, Explode, Radius, Center, WedgeWidth,
// RotateLabels, Normalize, Colors, Color, FaceColor, EdgeColor, LineWidth,
// Alpha, FontSize, TextColor, PlainText, SpreadLabels, ZOrder.
func (a *Axes) Pie(x any, opts ...Option) error {
	xs := Values(x)
	p := collect(opts)
	fracs, err := pieFractions(xs, !p.noNormalize)
	if err != nil {
		return err
	}
	n := len(xs)
	labels := p.labels
	if labels == nil {
		labels = make([]string, n)
	}
	explode := p.explode
	if explode == nil {
		explode = make([]float64, n)
	}
	switch {
	case len(labels) != n:
		return fmt.Errorf("plot: Pie: %d labels for %d wedges", len(labels), n)
	case len(explode) != n:
		return fmt.Errorf("plot: Pie: %d explode values for %d wedges", len(explode), n)
	case p.setRadius && !(p.radius > 0 && finite(p.radius)):
		return fmt.Errorf("plot: Pie: radius must be a positive number, not %v", p.radius)
	case !finite(p.startAngle) || !finite(p.center[0]) || !finite(p.center[1]):
		return errors.New("plot: Pie: start angle and centre must be finite")
	}
	for _, e := range explode {
		if !finite(e) {
			return errors.New("plot: Pie: explode values must be finite")
		}
	}
	radius := 1.0
	if p.setRadius {
		radius = p.radius
	}
	if !(p.wedgeWidth >= 0 && p.wedgeWidth <= radius) {
		return fmt.Errorf("plot: Pie: wedge width must be between 0 and the radius, not %v", p.wedgeWidth)
	}

	a.SetAspect(1)
	z := 1.0
	if p.setZorder {
		z = p.zorder
	}
	wedges := make([]*wedge, n)
	theta1 := p.startAngle / 360
	for i, frac := range fracs {
		theta2 := theta1 + frac
		if p.clockwise {
			theta2 = theta1 - frac
		}
		thetam := 2 * math.Pi * 0.5 * (theta1 + theta2)
		var c colors.RGBA
		switch {
		case p.colorsPer != nil:
			c = p.colorsPer[i%len(p.colorsPer)]
		case !p.setColor && !p.setFace:
			c = a.nextFillColor()
		}
		st := newPatchStyle(p, c)
		if p.colorsPer != nil {
			st.face = c
			if p.setAlpha {
				st.face.A = p.alpha
			}
		}
		w := &wedge{
			cx: p.center[0] + float64(explode[i]*math.Cos(thetam)), cy: p.center[1] + float64(explode[i]*math.Sin(thetam)),
			r: radius, theta1: 360 * math.Min(theta1, theta2), theta2: 360 * math.Max(theta1, theta2),
			width: p.wedgeWidth, style: st, label: labels[i], plain: p.plain, z: z,
		}
		wedges[i] = w
		a.add(w)
		theta1 = theta2
	}

	labelSize, pctSize := tickLabelSize, defaultFontSize
	if p.fontSize > 0 {
		labelSize, pctSize = p.fontSize, p.fontSize
	}
	labelDist := 1.1
	if p.setLabelDistance {
		labelDist = p.labelDistance
	}
	if !math.IsNaN(labelDist) {
		a.pieLabel(wedges, labels, labelDist, true, p.rotateLabels, labelSize, p)
	}
	if p.autopct != nil {
		pcts := make([]string, n)
		for i, f := range fracs {
			pcts[i] = p.autopct(100. * f)
		}
		pctDist := 0.6
		switch {
		case p.setPctDistance:
			pctDist = p.pctDistance
		case p.donut:
			pctDist = 1 - p.wedgeWidth/(2*radius)
		}
		a.pieLabel(wedges, pcts, pctDist, false, false, pctSize, p)
	}

	a.noFrame = true
	a.SetXTicks(nil)
	a.SetYTicks(nil)
	a.SetXLim(-1.25+p.center[0], 1.25+p.center[0])
	a.SetYLim(-1.25+p.center[1], 1.25+p.center[1])
	return nil
}

// Donut draws a ring chart: a Pie whose wedges are width deep from the
// rim (WedgeWidth), with the percentages, if any, in the middle of the
// ring unless PctDistance says otherwise. width must be in (0, radius].
//
// Options: those of Pie.
func (a *Axes) Donut(x any, width float64, opts ...Option) error {
	if !(width > 0) {
		return fmt.Errorf("plot: Donut: width must be positive, not %v", width)
	}
	return a.Pie(x, append([]Option{WedgeWidth(width), donut()}, opts...)...)
}

// pieLabel is matplotlib's Axes.pie_label: one text per wedge at distance
// radii along the middle of the wedge, aligned away from the centre when
// outer ("left" right of x = 0, "right" left of it, as matplotlib tests the
// data x and not the centre), and turned to the wedge when rotate.
func (a *Axes) pieLabel(ws []*wedge, labels []string, distance float64, outer, rotate bool, size float64, p props) {
	color := black
	if p.setTextColor {
		color = p.textColor
	}
	var group *pieLabels
	if outer && p.spreadLabels && !rotate {
		group = &pieLabels{}
	}
	for i, w := range ws {
		if labels[i] == "" {
			continue
		}
		thetam := 2 * math.Pi * 0.5 * (w.theta1 + w.theta2) / 360
		// Rounded product by product, as NumPy does: on the fusing
		// architectures, xt of a label at 12 o'clock changes sign, and
		// with it the alignment.
		xt := w.cx + float64(float64(distance*w.r)*math.Cos(thetam))
		yt := w.cy + float64(float64(distance*w.r)*math.Sin(thetam))
		ha, va, rot := "center", "center", 0.0
		if outer {
			ha = "right"
			if xt > 0 {
				ha = "left"
			}
		}
		if rotate {
			if outer {
				va = "top"
				if yt > 0 {
					va = "bottom"
				}
			}
			rot = thetam * (180 / math.Pi)
			if xt <= 0 {
				rot += 180
			}
		}
		t := &textArt{s: labels[i], x: xt, y: yt, size: size, ha: ha, va: va, rot: rot, color: color, z: 3, plain: p.plain}
		if group == nil {
			a.add(t)
			continue
		}
		group.texts = append(group.texts, t)
		group.rims = append(group.rims, render.Pt{X: w.cx + w.r*math.Cos(thetam), Y: w.cy + w.r*math.Sin(thetam)})
	}
	if group != nil && len(group.texts) > 0 {
		a.add(group)
	}
}

// pieLabels is the outer labels of a Pie drawn with SpreadLabels: on each
// side of the pie, from the top down, a label that would overlap the one
// above it moves down until it clears it, and a leader line joins it to
// its wedge.
type pieLabels struct {
	texts []*textArt
	rims  []render.Pt // the rim of each label's wedge, at its middle
}

func (g *pieLabels) zorder() float64           { return 3 }
func (g *pieLabels) limits() lim               { return emptyLim() }
func (g *pieLabels) sticky() (x, y []float64)  { return nil, nil }
func (g *pieLabels) legendEntry() *legendEntry { return nil }

// leaderColor is the leader lines' grey (matplotlib's grid colour).
var leaderColor = gridColor

func (g *pieLabels) draw(a *Axes, r render.Renderer) {
	f := a.fig
	specs, moved := g.place(a)
	st := render.Style{Stroke: leaderColor, Width: f.pt(0.8), Cap: render.CapButt}
	for i := range specs {
		if moved[i] {
			x, y := a.toPx(g.rims[i].X, g.rims[i].Y)
			r.DrawPath(line(x, y, specs[i].X, specs[i].Y), st)
		}
		f.drawText(r, specs[i])
	}
}

// place returns where each label goes, in pixels, and which ones moved.
func (g *pieLabels) place(a *Axes) ([]textSpec, []bool) {
	f := a.fig
	specs := make([]textSpec, len(g.texts))
	boxes := make([]box, len(g.texts))
	for i, t := range g.texts {
		specs[i] = t.spec(a)
		boxes[i] = f.layoutText(specs[i]).bbox
	}
	moved := make([]bool, len(specs))
	gap := f.pt(1)
	for _, side := range []string{"left", "right"} {
		var idx []int
		for i := range specs {
			if specs[i].HA == side {
				idx = append(idx, i)
			}
		}
		slices.SortStableFunc(idx, func(i, j int) int { return cmp.Compare(boxes[i].y0, boxes[j].y0) })
		bottom := math.Inf(-1)
		for _, i := range idx {
			if d := bottom + gap - boxes[i].y0; d > 0 {
				specs[i].Y += d
				boxes[i].y0 += d
				boxes[i].y1 += d
				moved[i] = true
			}
			bottom = boxes[i].y1
		}
	}
	return specs, moved
}
