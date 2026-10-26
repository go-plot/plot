// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package colors parses colour specifications and maps scalars to colours the
// way matplotlib does: the same names, the same default property cycle
// (tab10), and colormaps whose lookup tables are matplotlib's own, generated
// from it by tools/gen_colormaps.py.
package colors

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// RGBA is a colour with straight (not premultiplied) alpha, each component in
// [0, 1], as matplotlib holds colours.
type RGBA struct{ R, G, B, A float64 }

// Bytes returns the colour as 8-bit components, rounded to nearest.
func (c RGBA) Bytes() [4]uint8 {
	return [4]uint8{to8(c.R), to8(c.G), to8(c.B), to8(c.A)}
}

func to8(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, v)) * 255)) }

// Hex returns "#rrggbb", or "#rrggbbaa" when the colour is not opaque.
func (c RGBA) Hex() string {
	b := c.Bytes()
	if b[3] == 255 {
		return fmt.Sprintf("#%02x%02x%02x", b[0], b[1], b[2])
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", b[0], b[1], b[2], b[3])
}

// WithAlpha returns c with its alpha replaced.
func (c RGBA) WithAlpha(a float64) RGBA { c.A = a; return c }

// baseColors are matplotlib's single-letter colours. They are fractions, not
// hex values: "g" is half green.
var baseColors = map[string]RGBA{
	"b": {0, 0, 1, 1},
	"g": {0, 0.5, 0, 1},
	"r": {1, 0, 0, 1},
	"c": {0, 0.75, 0.75, 1},
	"m": {0.75, 0, 0.75, 1},
	"y": {0.75, 0.75, 0, 1},
	"k": {0, 0, 0, 1},
	"w": {1, 1, 1, 1},
}

// Cycle is matplotlib's default property cycle (axes.prop_cycle): the ten
// Tableau colours, tab:blue first.
var Cycle = []string{
	"#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd",
	"#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf",
}

// Tableau10 is Tableau's 2016 "Tableau 10" palette, the one
// go-widgets/toolkit's PieChart and charts use (their first six colours).
// matplotlib's tab10 (Cycle) is the earlier Tableau set. Pass it to a plot
// to match the toolkit: plot.Colors(colors.Tableau10...). d3's
// schemeTableau10 differs from these by one unit in four channels.
var Tableau10 = []string{
	"#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f",
	"#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac",
}

// CycleColor returns colour i of the default cycle, wrapping around.
func CycleColor(i int) RGBA {
	c, _ := parseHex(Cycle[((i%len(Cycle))+len(Cycle))%len(Cycle)])
	return c
}

// Parse reads a colour as matplotlib's to_rgba does: a name ("red",
// "tab:blue", "C3", "k"), a hex string ("#rgb", "#rgba", "#rrggbb",
// "#rrggbbaa"), a grey level ("0.5") or "none" (transparent).
func Parse(s string) (RGBA, error) {
	key := strings.ToLower(strings.TrimSpace(s))
	if key == "none" {
		return RGBA{}, nil
	}
	if c, ok := baseColors[key]; ok {
		return c, nil
	}
	if hex, ok := namedData[key]; ok {
		return parseHex(hex)
	}
	if len(key) == 2 && key[0] == 'c' && key[1] >= '0' && key[1] <= '9' {
		return CycleColor(int(key[1] - '0')), nil
	}
	if strings.HasPrefix(key, "#") {
		return parseHex(key)
	}
	if v, err := strconv.ParseFloat(key, 64); err == nil {
		if v < 0 || v > 1 {
			return RGBA{}, fmt.Errorf("colors: grey level %q outside [0, 1]", s)
		}
		return RGBA{v, v, v, 1}, nil
	}
	return RGBA{}, fmt.Errorf("colors: %q is not a colour", s)
}

// MustParse is Parse that panics on an invalid colour, for colours written
// in source code.
func MustParse(s string) RGBA {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

func parseHex(s string) (RGBA, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 || len(h) == 4 {
		var b strings.Builder
		for _, r := range h {
			b.WriteRune(r)
			b.WriteRune(r)
		}
		h = b.String()
	}
	if len(h) != 6 && len(h) != 8 {
		return RGBA{}, fmt.Errorf("colors: %q is not a hex colour", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return RGBA{}, fmt.Errorf("colors: %q is not a hex colour", s)
	}
	if len(h) == 6 {
		v = v<<8 | 0xff
	}
	return RGBA{
		R: float64(v>>24&0xff) / 255,
		G: float64(v>>16&0xff) / 255,
		B: float64(v>>8&0xff) / 255,
		A: float64(v&0xff) / 255,
	}, nil
}
