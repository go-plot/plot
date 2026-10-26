// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package plot draws publication-quality figures — line plots, scatter
// plots, bars, histograms, images, contours — to SVG and PNG, in pure Go.
//
// The object model is matplotlib's: a Figure holds Axes, an Axes holds what
// is plotted on it, and the defaults (sizes, fonts, colours, margins, tick
// placement and labels) are matplotlib's own, so a figure looks the way a
// matplotlib user expects:
//
//	fig, axs := plot.Subplots(1, 1)
//	ax := axs[0][0]
//	ax.Plot(x, y, plot.Label("data"))
//	ax.SetXLabel("time $t$ (s)")
//	ax.Legend()
//	fig.SavePNG(w)
//
// Labels may contain TeX math between dollar signs, typeset by go-tex/math.
package plot

import (
	"bytes"
	"image/png"
	"io"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
	"github.com/go-plot/plot/text"
)

// Figure is the whole drawing: a size, a resolution, a background and the
// Axes on it.
type Figure struct {
	// Width and Height are in inches; DPI is pixels per inch. The pixel size
	// is Width*DPI x Height*DPI, rounded as matplotlib rounds it.
	Width, Height, DPI float64
	// Facecolor is the background; white by default.
	Facecolor colors.RGBA
	// Text lays out every string in the figure. The default sets text in
	// DejaVu Sans and TeX math with go-tex/math; replace it to change fonts.
	Text text.Renderer
	// SubplotParams place the grids made by Subplots; TightLayout sets them.
	SubplotParams SubplotParams

	axes      []*Axes
	collect   *[]box // while measuring: the extent of every text drawn
	plainText bool   // ParseMath(false): callers' strings are set without TeX
}

// A FigOption configures a new Figure.
type FigOption func(*Figure)

// FigSize sets the figure size in inches (matplotlib's figsize; default 6.4
// x 4.8).
func FigSize(w, h float64) FigOption { return func(f *Figure) { f.Width, f.Height = w, h } }

// DPI sets the resolution in pixels per inch (default 100).
func DPI(d float64) FigOption { return func(f *Figure) { f.DPI = d } }

// NewFigure returns an empty figure with matplotlib's defaults: 6.4 x 4.8
// inches at 100 dpi on white.
func NewFigure(opts ...FigOption) *Figure {
	f := &Figure{Width: 6.4, Height: 4.8, DPI: 100, Facecolor: colors.RGBA{R: 1, G: 1, B: 1, A: 1}, SubplotParams: defaultSubplotParams}
	for _, o := range opts {
		o(f)
	}
	if f.Text == nil {
		f.Text = text.NewDefault()
	}
	return f
}

// Subplots returns a new figure with a rows x cols grid of Axes, as
// matplotlib's plt.subplots does; axs[i][j] is row i, column j.
func Subplots(rows, cols int, opts ...FigOption) (*Figure, [][]*Axes) {
	f := NewFigure(opts...)
	return f, f.Subplots(rows, cols)
}

// SubplotParams are matplotlib's figure.subplot.* parameters: where the
// grid of Subplots sits in the figure (fractions) and the space between its
// cells (fractions of the average cell width and height).
type SubplotParams struct {
	Left, Right, Bottom, Top float64
	WSpace, HSpace           float64
}

// defaultSubplotParams are matplotlib's defaults.
var defaultSubplotParams = SubplotParams{Left: 0.125, Right: 0.9, Bottom: 0.11, Top: 0.88, WSpace: 0.2, HSpace: 0.2}

// gridSpec is a rows x cols grid made by Subplots.
type gridSpec struct{ rows, cols int }

// cell is where an Axes sits in a grid.
type cell struct {
	g        *gridSpec
	row, col int
}

// cellRect is the position of a grid cell under sp, as matplotlib's
// GridSpec computes it.
func (sp SubplotParams) cellRect(c cell) rect {
	rows, cols := float64(c.g.rows), float64(c.g.cols)
	totW, totH := sp.Right-sp.Left, sp.Top-sp.Bottom
	cellH := totH / (rows + sp.HSpace*(rows-1))
	sepH := sp.HSpace * cellH
	cellW := totW / (cols + sp.WSpace*(cols-1))
	sepW := sp.WSpace * cellW
	top := sp.Top - float64(c.row)*(cellH+sepH)
	left := sp.Left + float64(c.col)*(cellW+sepW)
	return rect{left, top - cellH, cellW, cellH}
}

// Subplots adds a rows x cols grid of Axes placed as matplotlib's GridSpec
// places them.
func (f *Figure) Subplots(rows, cols int) [][]*Axes {
	g := &gridSpec{rows, cols}
	out := make([][]*Axes, rows)
	for i := range rows {
		out[i] = make([]*Axes, cols)
		for j := range cols {
			c := cell{g, i, j}
			r := f.SubplotParams.cellRect(c)
			out[i][j] = f.AddAxes(r.x, r.y, r.w, r.h)
			out[i][j].cell = &c
		}
	}
	return out
}

// AddAxes adds an Axes at [left, bottom, width, height] in figure
// fractions, as matplotlib's fig.add_axes does.
func (f *Figure) AddAxes(left, bottom, width, height float64) *Axes {
	a := newAxes(f, rect{left, bottom, width, height})
	f.axes = append(f.axes, a)
	return a
}

// Axes returns the figure's Axes in the order they were added.
func (f *Figure) Axes() []*Axes { return append([]*Axes(nil), f.axes...) }

// PixelSize returns the figure size in pixels.
func (f *Figure) PixelSize() (w, h int) {
	return int(f.Width*f.DPI + 0.5), int(f.Height*f.DPI + 0.5)
}

// pt converts points to pixels.
func (f *Figure) pt(v float64) float64 { return v * f.DPI / 72 }

// Draw draws the figure through r.
func (f *Figure) Draw(r render.Renderer) {
	w, h := r.Size()
	var bg render.Path
	bg.Rect(0, 0, float64(w), float64(h))
	r.BeginGroup("figure_1")
	r.DrawPath(&bg, render.Style{Fill: f.Facecolor})
	f.layout()
	for i, a := range f.axes {
		a.draw(r, i+1)
	}
	r.EndGroup()
}

// layout settles every Axes: limits, position, ticks.
func (f *Figure) layout() {
	for _, a := range f.axes {
		a.layout()
	}
}

// Image renders the figure to an RGBA raster.
func (f *Figure) Image() *render.Raster {
	w, h := f.PixelSize()
	r := render.NewRaster(w, h)
	f.Draw(r)
	return r
}

// SavePNG writes the figure as a PNG.
func (f *Figure) SavePNG(w io.Writer) error {
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(w, f.Image().Img)
}

// SaveSVG writes the figure as an SVG document.
func (f *Figure) SaveSVG(w io.Writer) error {
	pw, ph := f.PixelSize()
	s := render.NewSVG(pw, ph, f.DPI)
	f.Draw(s)
	_, err := s.WriteTo(w)
	return err
}

// Display returns the figure in every format a notebook front end may show,
// keyed by MIME type: "image/svg+xml" and "image/png". It is the display
// contract of the go-notebook kernels, which show the richest format they
// support.
func (f *Figure) Display() map[string][]byte {
	var svg, pngb bytes.Buffer
	_ = f.SaveSVG(&svg)  // writing to memory cannot fail
	_ = f.SavePNG(&pngb) // likewise
	return map[string][]byte{"image/svg+xml": svg.Bytes(), "image/png": pngb.Bytes()}
}
