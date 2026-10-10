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

// Mappable is what maps data values to colours — the result of Imshow,
// Scatter with CValues, Contour and Contourf — and what Colorbar draws.
type Mappable struct {
	Cmap *colors.Colormap
	Norm colors.Norm
	// boundaries and values make a discrete colorbar (contour sets): band k
	// spans boundaries[k]..boundaries[k+1] in the colour of values[k].
	boundaries, values []float64
}

// newMappable resolves Cmap, Vmin, Vmax and LogNorm against the data range.
func newMappable(p props, data []float64) *Mappable {
	name := p.cmap
	if name == "" {
		name = "viridis"
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range data {
		if math.IsNaN(v) || math.IsInf(v, 0) || (p.logNorm && v <= 0) {
			continue
		}
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if lo > hi {
		lo, hi = 0, 1
	}
	if p.setVmin {
		lo = p.vmin
	}
	if p.setVmax {
		hi = p.vmax
	}
	m := &Mappable{Cmap: colors.MustGet(name)}
	if p.logNorm {
		m.Norm = colors.LogNorm{Vmin: lo, Vmax: hi}
	} else {
		m.Norm = colors.Normalize{Vmin: lo, Vmax: hi}
	}
	return m
}

// color returns the colour for a data value.
func (m *Mappable) color(v float64) colors.RGBA { return m.Cmap.At(m.Norm.Normalize(v)) }

// image is an Imshow: a raster placed over an extent in data coordinates.
type imageArt struct {
	pix    *image.RGBA
	extent [4]float64 // left, right, bottom, top
	z      float64
	label  string
}

func (im *imageArt) zorder() float64 { return im.z }

func (im *imageArt) limits() lim {
	lm := emptyLim()
	lm.addXY(im.extent[0], im.extent[2])
	lm.addXY(im.extent[1], im.extent[3])
	return lm
}

func (im *imageArt) sticky() (x, y []float64) {
	return []float64{im.extent[0], im.extent[1]}, []float64{im.extent[2], im.extent[3]}
}

func (im *imageArt) legendEntry() *legendEntry { return nil }

func (im *imageArt) draw(a *Axes, r render.Renderer) {
	x0, y0 := a.toPx(im.extent[0], im.extent[3])
	x1, y1 := a.toPx(im.extent[1], im.extent[2])
	pix := im.pix
	// The image's own rows run top to bottom in data order "top"; flip
	// whatever the axes direction flipped.
	if x1 < x0 {
		x0, x1 = x1, x0
		pix = flipX(pix)
	}
	if y1 < y0 {
		y0, y1 = y1, y0
		pix = flipY(pix)
	}
	// Shown smaller than its resolution, an image is averaged down to the
	// pixels it covers (matplotlib anti-aliases it too) rather than sampled.
	dw, dh := int(math.Round(x1)-math.Round(x0)), int(math.Round(y1)-math.Round(y0))
	if dw > 0 && dh > 0 && (pix.Rect.Dx() > dw || pix.Rect.Dy() > dh) {
		pix = boxDownsample(pix, min(dw, pix.Rect.Dx()), min(dh, pix.Rect.Dy()))
	}
	a.clipToAxes(r)
	r.DrawImage(render.Image{X: x0, Y: y0, W: x1 - x0, H: y1 - y0, Pix: pix})
	r.PopClip()
}

func flipY(src *image.RGBA) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		copy(out.Pix[y*out.Stride:y*out.Stride+w*4], src.Pix[(h-1-y)*src.Stride:])
	}
	return out
}

func flipX(src *image.RGBA) *image.RGBA {
	w, h := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			copy(out.Pix[y*out.Stride+x*4:y*out.Stride+x*4+4], src.Pix[y*src.Stride+(w-1-x)*4:])
		}
	}
	return out
}

// boxDownsample averages src down to w x h: each destination pixel is the
// mean (alpha-weighted) of the source pixels whose centres fall in it.
func boxDownsample(src *image.RGBA, w, h int) *image.RGBA {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	acc := make([]float64, w*5)
	for y := range h {
		sy0, sy1 := y*sh/h, (y+1)*sh/h
		clear(acc)
		for sy := sy0; sy < sy1; sy++ {
			row := src.Pix[sy*src.Stride:]
			for x := range w {
				sx0, sx1 := x*sw/w, (x+1)*sw/w
				for sx := sx0; sx < sx1; sx++ {
					o := sx * 4
					a := float64(row[o+3])
					acc[x*5] += float64(row[o]) * a
					acc[x*5+1] += float64(row[o+1]) * a
					acc[x*5+2] += float64(row[o+2]) * a
					acc[x*5+3] += a
					acc[x*5+4]++
				}
			}
		}
		for x := range w {
			o := y*out.Stride + x*4
			if a := acc[x*5+3]; a > 0 {
				out.Pix[o] = uint8(acc[x*5]/a + 0.5)
				out.Pix[o+1] = uint8(acc[x*5+1]/a + 0.5)
				out.Pix[o+2] = uint8(acc[x*5+2]/a + 0.5)
				out.Pix[o+3] = uint8(a/acc[x*5+4] + 0.5)
			}
		}
	}
	return out
}

// Imshow shows 2-D data as an image, as matplotlib's ax.imshow: each value
// coloured through the colormap (viridis by default, scaled to the data
// range), pixel centres at integer coordinates, row 0 at the top, square
// pixels. A 3-D *ndarray.Array of shape (rows, cols, 3 or 4) is shown as
// RGB(A): floats in [0, 1], or uint8.
//
// Options: Cmap, Vmin, Vmax, LogNorm, Extent, Origin, Aspect, Alpha, ZOrder.
func (a *Axes) Imshow(data any, opts ...Option) *Mappable {
	p := collect(opts)
	pix, m := toImage(data, p)
	rows, cols := pix.Rect.Dy(), pix.Rect.Dx()
	im := &imageArt{pix: pix, z: 0}
	if p.setZorder {
		im.z = p.zorder
	}
	if p.setExtent {
		im.extent = p.extent
	} else if p.origin == "lower" {
		im.extent = [4]float64{-0.5, float64(cols) - 0.5, -0.5, float64(rows) - 0.5}
	} else {
		im.extent = [4]float64{-0.5, float64(cols) - 0.5, float64(rows) - 0.5, -0.5}
	}
	if p.origin == "lower" {
		im.pix = flipY(pix)
	}
	if im.extent[2] > im.extent[3] {
		a.y.inverted = true
	}
	if p.aspect != "auto" {
		a.aspect = 1
	}
	a.add(im)
	return m
}

// toImage colours 2-D data, or takes RGB(A) data as it is.
func toImage(data any, p props) (*image.RGBA, *Mappable) {
	if rgb, ok := rgbImage(data, p); ok {
		return rgb, nil
	}
	g := grid(data)
	m := newMappable(p, g.Z)
	pix := image.NewRGBA(image.Rect(0, 0, g.Cols, g.Rows))
	for i, v := range g.Z {
		c := m.color(v)
		if p.setAlpha {
			c.A *= p.alpha
		}
		// matplotlib quantises with to_rgba(bytes=True), which truncates.
		pix.Pix[i*4] = uint8(c.R * 255)
		pix.Pix[i*4+1] = uint8(c.G * 255)
		pix.Pix[i*4+2] = uint8(c.B * 255)
		pix.Pix[i*4+3] = uint8(c.A * 255)
	}
	return pix, m
}
