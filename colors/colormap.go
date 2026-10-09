// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package colors

import (
	"fmt"
	"math"
	"slices"
)

// A Colormap maps [0, 1] to colours through a 256-entry lookup table, as a
// matplotlib Colormap with N=256 does.
type Colormap struct {
	Name string
	lut  *[256][4]float64
	// Under, Over and Bad are the colours for values below 0, above 1 and
	// NaN. Matplotlib's defaults are the first entry, the last entry and
	// transparent black; Get sets them so.
	Under, Over, Bad RGBA
}

// Names returns the names of the built-in colormaps, sorted.
func Names() []string {
	var out []string
	for k := range lutData {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Get returns the named built-in colormap: viridis, plasma, inferno, magma,
// cividis, gray, coolwarm or RdBu. A name ending in "_r" returns the map
// reversed, as in matplotlib.
func Get(name string) (*Colormap, error) {
	base, reversed := name, false
	if len(name) > 2 && name[len(name)-2:] == "_r" {
		base, reversed = name[:len(name)-2], true
	}
	lut, ok := lutData[base]
	if !ok {
		return nil, fmt.Errorf("colors: no colormap %q (have %v)", name, Names())
	}
	if reversed {
		var r [256][4]float64
		for i := range lut {
			r[i] = lut[255-i]
		}
		lut = r
	}
	c := &Colormap{Name: name, lut: &lut}
	c.Under = c.entry(0)
	c.Over = c.entry(255)
	return c, nil
}

// MustGet is Get that panics on an unknown name.
func MustGet(name string) *Colormap {
	c, err := Get(name)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Colormap) entry(i int) RGBA {
	e := c.lut[i]
	return RGBA{e[0], e[1], e[2], e[3]}
}

// Index returns the lookup-table index matplotlib uses for x, or -1 for under,
// 256 for over and -2 for NaN: x*256, with 1.0 itself mapped to the last
// entry, truncated toward zero.
func Index(x float64) int {
	if math.IsNaN(x) {
		return -2
	}
	xa := x * 256
	if xa == 256 {
		xa = 255
	}
	switch {
	case xa < 0:
		return -1
	case xa >= 256:
		return 256
	}
	return int(xa)
}

// At returns the colour for x, as matplotlib's Colormap.__call__ does for a
// float.
func (c *Colormap) At(x float64) RGBA {
	switch i := Index(x); i {
	case -2:
		return c.Bad
	case -1:
		return c.Under
	case 256:
		return c.Over
	default:
		return c.entry(i)
	}
}

// BytesAt returns the colour for x as 8-bit components exactly as
// matplotlib's Colormap.__call__(x, bytes=True) does, which truncates
// (lut*255).astype(uint8) rather than rounding.
func (c *Colormap) BytesAt(x float64) [4]uint8 {
	col := c.At(x)
	return [4]uint8{uint8(col.R * 255), uint8(col.G * 255), uint8(col.B * 255), uint8(col.A * 255)}
}

// A Norm maps data values to [0, 1] before a colormap is applied.
type Norm interface {
	Normalize(v float64) float64
	Range() (vmin, vmax float64)
}

// Normalize is matplotlib's linear Normalize: (v - Vmin) / (Vmax - Vmin), and
// 0 when Vmin == Vmax.
type Normalize struct{ Vmin, Vmax float64 }

// Normalize implements Norm.
func (n Normalize) Normalize(v float64) float64 {
	if n.Vmin == n.Vmax {
		return 0
	}
	return (v - n.Vmin) / (n.Vmax - n.Vmin)
}

// Range implements Norm.
func (n Normalize) Range() (float64, float64) { return n.Vmin, n.Vmax }

// LogNorm normalises on a base-10 logarithmic scale. Values <= 0 map to NaN
// (drawn in the colormap's Bad colour), as matplotlib masks them.
type LogNorm struct{ Vmin, Vmax float64 }

// Normalize implements Norm.
func (n LogNorm) Normalize(v float64) float64 {
	if v <= 0 {
		return math.NaN()
	}
	lo, hi := math.Log10(n.Vmin), math.Log10(n.Vmax)
	if lo == hi {
		return 0
	}
	return (math.Log10(v) - lo) / (hi - lo)
}

// Range implements Norm.
func (n LogNorm) Range() (float64, float64) { return n.Vmin, n.Vmax }
