// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"github.com/go-plot/plot/colors"
)

// An Option sets one property of what a plotting method creates, as a
// matplotlib keyword argument does: ax.Plot(x, y, plot.Color("C1"),
// plot.LineStyle("--"), plot.Label("data")). Each method documents the
// options it honours and ignores the rest.
type Option func(*props)

// props collects every option; unset fields keep their zero value and the
// set* flags say which were given.
type props struct {
	color, edgeColor, faceColor, ecolor colors.RGBA
	setColor, setEdge, setFace, setE    bool

	colorsPer []colors.RGBA // Colors
	sizes     []float64     // Sizes
	cvalues   []float64     // CValues

	lineWidth, markerSize, markerEdgeWidth float64
	lineStyle, marker, label               string
	setLineStyle, setMarker                bool
	alpha                                  float64
	setAlpha                               bool
	zorder                                 float64
	setZorder                              bool

	cmap       string
	vmin, vmax float64
	setVmin    bool
	setVmax    bool
	logNorm    bool

	width      float64 // bar width
	bottom     float64 // bar baseline
	align      string
	bins       int
	binEdges   []float64
	density    bool
	levels     []float64
	nlevels    int
	extent     [4]float64
	setExtent  bool
	origin     string
	aspect     string
	capsize    float64
	fontSize   float64
	ha, va     string
	rotation   float64
	clip       int // 0 default, 1 on, 2 off
	where      []bool
	arrow      bool
	xerr, yerr []float64
	loc        string
}

func collect(opts []Option) props {
	var p props
	for _, o := range opts {
		o(&p)
	}
	return p
}

// Color sets the colour: a name ("red", "C1", "tab:blue"), a hex string or
// a grey level, as matplotlib accepts. An invalid colour panics.
func Color(c string) Option {
	col := colors.MustParse(c)
	return func(p *props) { p.color, p.setColor = col, true }
}

// RGBA sets the colour from components in [0, 1].
func RGBA(r, g, b, a float64) Option {
	return func(p *props) { p.color, p.setColor = colors.RGBA{R: r, G: g, B: b, A: a}, true }
}

// EdgeColor sets the edge colour of patches and markers.
func EdgeColor(c string) Option {
	col := colors.MustParse(c)
	return func(p *props) { p.edgeColor, p.setEdge = col, true }
}

// FaceColor sets the fill colour of patches and markers.
func FaceColor(c string) Option {
	col := colors.MustParse(c)
	return func(p *props) { p.faceColor, p.setFace = col, true }
}

// ErrorColor sets the colour of error bars (matplotlib's ecolor).
func ErrorColor(c string) Option {
	col := colors.MustParse(c)
	return func(p *props) { p.ecolor, p.setE = col, true }
}

// Colors sets one colour per point (Scatter) or per bar (Bar).
func Colors(cs ...string) Option {
	out := make([]colors.RGBA, len(cs))
	for i, c := range cs {
		out[i] = colors.MustParse(c)
	}
	return func(p *props) { p.colorsPer = out }
}

// CValues gives Scatter one value per point, mapped to colours through the
// colormap (matplotlib's c= with numbers). Any data Values accepts.
func CValues(v any) Option {
	vals := Values(v)
	return func(p *props) { p.cvalues = vals }
}

// Sizes gives Scatter one marker area per point, in points squared
// (matplotlib's s=). A single value applies to every point.
func Sizes(v any) Option {
	vals := Values(v)
	return func(p *props) { p.sizes = vals }
}

// LineWidth sets the line width in points.
func LineWidth(w float64) Option { return func(p *props) { p.lineWidth = w } }

// LineStyle sets the line style: "-", "--", "-.", ":", or "" / "None" for no
// line.
func LineStyle(s string) Option {
	return func(p *props) { p.lineStyle, p.setLineStyle = s, true }
}

// Marker sets the marker: "o", ".", ",", "s", "^", "v", "<", ">", "D", "d",
// "p", "h", "*", "+", "x", "|", "_", or "" for none.
func Marker(m string) Option { return func(p *props) { p.marker, p.setMarker = m, true } }

// MarkerSize sets the marker size in points.
func MarkerSize(s float64) Option { return func(p *props) { p.markerSize = s } }

// MarkerEdgeWidth sets the marker edge width in points.
func MarkerEdgeWidth(w float64) Option { return func(p *props) { p.markerEdgeWidth = w } }

// Label names the artist in the legend.
func Label(s string) Option { return func(p *props) { p.label = s } }

// Alpha sets the opacity in [0, 1].
func Alpha(a float64) Option { return func(p *props) { p.alpha, p.setAlpha = a, true } }

// ZOrder sets the drawing order: higher is drawn later.
func ZOrder(z float64) Option { return func(p *props) { p.zorder, p.setZorder = z, true } }

// Cmap sets the colormap by name (see colors.Names).
func Cmap(name string) Option {
	colors.MustGet(name)
	return func(p *props) { p.cmap = name }
}

// Vmin sets the data value mapped to the bottom of the colormap.
func Vmin(v float64) Option { return func(p *props) { p.vmin, p.setVmin = v, true } }

// Vmax sets the data value mapped to the top of the colormap.
func Vmax(v float64) Option { return func(p *props) { p.vmax, p.setVmax = v, true } }

// LogNorm maps data to the colormap on a logarithmic scale.
func LogNorm() Option { return func(p *props) { p.logNorm = true } }

// Width sets the bar width (height, for Barh) in data units.
func Width(w float64) Option { return func(p *props) { p.width = w } }

// Bottom sets the bar baseline (left edge, for Barh).
func Bottom(b float64) Option { return func(p *props) { p.bottom = b } }

// Align sets bar alignment on its coordinate: "center" (default) or "edge".
func Align(a string) Option { return func(p *props) { p.align = a } }

// Bins sets the number of histogram bins.
func Bins(n int) Option { return func(p *props) { p.bins = n } }

// BinEdges sets the histogram bin edges explicitly.
func BinEdges(e ...float64) Option { return func(p *props) { p.binEdges = e } }

// Density normalises a histogram to unit area.
func Density() Option { return func(p *props) { p.density = true } }

// Levels sets contour levels explicitly.
func Levels(l ...float64) Option { return func(p *props) { p.levels = l } }

// NLevels asks for about n contour levels (matplotlib's levels=n).
func NLevels(n int) Option { return func(p *props) { p.nlevels = n } }

// Extent places an image: left, right, bottom, top in data coordinates.
func Extent(left, right, bottom, top float64) Option {
	return func(p *props) { p.extent, p.setExtent = [4]float64{left, right, bottom, top}, true }
}

// Origin sets where Imshow puts row 0: "upper" (default) or "lower".
func Origin(o string) Option { return func(p *props) { p.origin = o } }

// Aspect sets Imshow's aspect: "equal" (default) or "auto".
func Aspect(a string) Option { return func(p *props) { p.aspect = a } }

// CapSize sets error bar cap length in points.
func CapSize(s float64) Option { return func(p *props) { p.capsize = s } }

// XErr gives ErrorBar symmetric horizontal errors: one value, or one per
// point.
func XErr(v any) Option {
	vals := Values(v)
	return func(p *props) { p.xerr = vals }
}

// YErr gives ErrorBar symmetric vertical errors: one value, or one per point.
func YErr(v any) Option {
	vals := Values(v)
	return func(p *props) { p.yerr = vals }
}

// FontSize sets text size in points.
func FontSize(s float64) Option { return func(p *props) { p.fontSize = s } }

// HA sets horizontal text alignment: "left", "center" or "right".
func HA(a string) Option { return func(p *props) { p.ha = a } }

// VA sets vertical text alignment: "top", "center", "baseline",
// "center_baseline" or "bottom".
func VA(a string) Option { return func(p *props) { p.va = a } }

// Rotation sets text rotation in degrees, counter-clockwise.
func Rotation(deg float64) Option { return func(p *props) { p.rotation = deg } }

// Clip turns clipping to the axes on or off.
func Clip(on bool) Option {
	return func(p *props) {
		p.clip = 2
		if on {
			p.clip = 1
		}
	}
}

// Where restricts FillBetween to the points where the mask is true.
func Where(mask []bool) Option { return func(p *props) { p.where = mask } }

// Arrow draws an arrow from an annotation's text to the point it annotates.
func Arrow() Option { return func(p *props) { p.arrow = true } }
