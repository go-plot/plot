// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
)

var tightCases = map[string]func(f *Figure){
	"line":   layoutCases["line"],
	"offset": layoutCases["offset"],
	"loglog": layoutCases["loglog"],
	"labelled": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot(lx, ly)
		ax.SetTitle("A title")
		ax.SetXLabel("x label")
		ax.SetYLabel("y label")
	},
	"grid22_labelled": func(f *Figure) {
		axs := f.Subplots(2, 2)
		for i := range 4 {
			a := axs[i/2][i%2]
			a.Plot(lx, scaled(ly, math.Pow(10, float64(i)), 0))
			a.SetTitle(fmt.Sprintf("panel %d", i))
			a.SetXLabel("time")
			a.SetYLabel("value")
		}
	},
	"twin_labelled": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot(lx, ly)
		ax.SetYLabel("left")
		t := ax.TwinX()
		t.Plot(lx, scaled(ly, 1000, 0))
		t.SetYLabel("right")
	},
}

// tightTol is how far, in figure fractions, an Axes edge may sit from
// matplotlib's after TightLayout. The algorithm is matplotlib's; what
// differs is the measured width of text: go-plot uses the string's advance,
// matplotlib FreeType's hinted bounding box, in which every glyph is a whole
// number of pixels, so it is up to a few pixels wider on a long label. The
// worst case measured is 0.0049 (2.4 pixels of a 500-pixel figure).
const tightTol = 0.005

func TestTightLayoutMatchesMatplotlib(t *testing.T) {
	var fx struct {
		Tight map[string][][]float64 `json:"tight"`
	}
	if err := json.Unmarshal(layoutJSON, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Tight) != 2*len(tightCases) {
		t.Fatalf("%d fixtures for %d cases", len(fx.Tight), len(tightCases))
	}
	worst := 0.0
	for name, mk := range tightCases {
		for _, size := range [][2]float64{{6.4, 4.8}, {5, 3}} {
			key := fmt.Sprintf("%s@%gx%g", name, size[0], size[1])
			if size[0] == 5 {
				key = name + "@5.0x3.0"
			}
			f := NewFigure(FigSize(size[0], size[1]))
			mk(f)
			if !f.TightLayout() {
				t.Errorf("%s: not applied", key)
				continue
			}
			for i, a := range f.axes {
				w := fx.Tight[key][i]
				got := []float64{a.pos.x, a.pos.y, a.pos.w, a.pos.h}
				for k := range got {
					d := math.Abs(got[k] - w[k])
					worst = math.Max(worst, d)
					if d > tightTol {
						t.Errorf("%s axes %d: %v, want %v", key, i, got, w)
						break
					}
				}
			}
		}
	}
	t.Logf("worst edge difference %.5f of the figure", worst)
}

func TestTightLayoutRefuses(t *testing.T) {
	if NewFigure().TightLayout() {
		t.Error("no grid: applied")
	}
	f := NewFigure(FigSize(1, 1))
	ax := f.Subplots(1, 1)[0][0]
	ax.Plot(lx, ly)
	ax.SetTitle("a very long title that cannot possibly fit")
	ax.SetYLabel("label\nlabel\nlabel\nlabel\nlabel")
	ax.SetXLabel("label\nlabel\nlabel\nlabel\nlabel")
	if f.TightLayout() {
		t.Error("impossible layout applied")
	}
	for _, rc := range [][2]int{{1, 12}, {12, 1}} {
		f := NewFigure(FigSize(2, 2))
		for _, row := range f.Subplots(rc[0], rc[1]) {
			for _, a := range row {
				a.SetYLabel("long label")
				a.SetXLabel("long label")
				a.SetTitle("long")
			}
		}
		if f.TightLayout() {
			t.Errorf("%v cells too many: applied", rc)
		}
	}
	// With a colorbar, and an extra Axes outside the grid.
	f2 := NewFigure()
	a2 := f2.Subplots(1, 1)[0][0]
	f2.Colorbar(a2.Imshow(lz()), a2)
	f2.AddAxes(0.7, 0.7, 0.2, 0.2)
	if !f2.TightLayout() {
		t.Error("colorbar layout not applied")
	}
	renderBoth(t, f2)
	if w, h := (nullRenderer{3, 4}).Size(); w != 3 || h != 4 {
		t.Error("nullRenderer size")
	}
}
