// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"testing"
)

// layout.json is written by tools/gen_layout.py: for small figures, where
// matplotlib puts each Axes, its view limits and its tick labels once drawn.
//
//go:embed testdata/layout.json
var layoutJSON []byte

var (
	lx = []float64{0.0, 0.5, 1.3, 2.0, 3.7, 5.0}
	ly = []float64{1.0, -2.0, 0.5, 4.0, 3.0, 2.5}
)

func lz() [][]float64 {
	z := make([][]float64, 4)
	for i := range z {
		z[i] = make([]float64, 6)
		for j := range z[i] {
			z[i][j] = float64((i*7 + j*3) % 11)
		}
	}
	return z
}

func scaled(v []float64, k, c float64) []float64 {
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = k*x + c
	}
	return out
}

// layoutCases builds the same figures as tools/gen_layout.py.
var layoutCases = map[string]func(f *Figure){
	"line":   func(f *Figure) { f.Subplots(1, 1)[0][0].Plot(lx, ly) },
	"bar":    func(f *Figure) { f.Subplots(1, 1)[0][0].Bar([]float64{1, 2, 3}, []float64{4, 9, 2}) },
	"barh":   func(f *Figure) { f.Subplots(1, 1)[0][0].Barh([]float64{1, 2, 3}, []float64{4, 9, 2}) },
	"hist":   func(f *Figure) { f.Subplots(1, 1)[0][0].Hist(append(append([]float64{}, lx...), ly...), Bins(5)) },
	"imshow": func(f *Figure) { f.Subplots(1, 1)[0][0].Imshow(lz()) },
	"imshow_cbar": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		f.Colorbar(ax.Imshow(lz()), ax)
	},
	"scatter": func(f *Figure) { f.Subplots(1, 1)[0][0].Scatter(lx, ly) },
	"loglog": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot([]float64{0.5, 3, 40, 800}, []float64{2e-3, 1, 5, 7e4})
		ax.SetXScale("log")
		ax.SetYScale("log")
	},
	"fill": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.FillBetween(lx, ly, 0.0)
		ax.ErrorBar(lx, ly, YErr(0.5))
	},
	"contourf": func(f *Figure) {
		f.Subplots(1, 1)[0][0].Contourf([]float64{0, 1, 2, 3, 4, 5}, []float64{0, 10, 20, 30}, lz())
	},
	"grid22": func(f *Figure) {
		axs := f.Subplots(2, 2)
		axs[0][0].Plot(lx, ly)
		axs[1][1].Plot(ly, lx)
	},
	"twin": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot(lx, ly)
		ax.TwinX().Plot(lx, scaled(ly, 1000, 0))
	},
	"offset": func(f *Figure) { f.Subplots(1, 1)[0][0].Plot(scaled(lx, 1, 1e5), scaled(ly, 1e-7, 0)) },
	"axlines": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot(lx, ly)
		ax.Axhline(7)
		ax.Axvline(-1)
	},
	"setlim": func(f *Figure) {
		ax := f.Subplots(1, 1)[0][0]
		ax.Plot(lx, ly)
		ax.SetXLim(5, -1)
		ax.SetYLim(2, 2)
	},
}

func closeRel(a, b float64) bool {
	return a == b || math.Abs(a-b) <= 1e-9*math.Max(math.Abs(a), math.Abs(b))
}

// TestLayoutMatchesMatplotlib checks positions and limits to 1e-9 relative
// and every tick label and offset text exactly.
func TestLayoutMatchesMatplotlib(t *testing.T) {
	var fx struct {
		Version string `json:"version"`
		Cases   map[string][]struct {
			Position []float64 `json:"position"`
			Xlim     []float64 `json:"xlim"`
			Ylim     []float64 `json:"ylim"`
			Xticks   []string  `json:"xticks"`
			Yticks   []string  `json:"yticks"`
			Xoffset  string    `json:"xoffset"`
			Yoffset  string    `json:"yoffset"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(layoutJSON, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) != 2*len(layoutCases) {
		t.Fatalf("%d fixture cases for %d builders", len(fx.Cases), len(layoutCases))
	}
	for name, mk := range layoutCases {
		for _, size := range [][2]float64{{6.4, 4.8}, {5, 3}} {
			key := fmt.Sprintf("%s@%gx%g", name, size[0], size[1])
			if size[0] == 5 {
				key = fmt.Sprintf("%s@5.0x3.0", name)
			}
			want, ok := fx.Cases[key]
			if !ok {
				t.Fatalf("no fixture %s", key)
			}
			f := NewFigure(FigSize(size[0], size[1]))
			mk(f)
			f.layout()
			if len(f.axes) != len(want) {
				t.Errorf("%s: %d axes, want %d", key, len(f.axes), len(want))
				continue
			}
			for i, a := range f.axes {
				w := want[i]
				pos := []float64{a.active.x, a.active.y, a.active.w, a.active.h}
				for k := range pos {
					if !closeRel(pos[k], w.Position[k]) {
						t.Errorf("%s axes %d: position %v, want %v", key, i, pos, w.Position)
						break
					}
				}
				x0, x1 := a.XLim()
				y0, y1 := a.YLim()
				if !closeRel(x0, w.Xlim[0]) || !closeRel(x1, w.Xlim[1]) || !closeRel(y0, w.Ylim[0]) || !closeRel(y1, w.Ylim[1]) {
					t.Errorf("%s axes %d: lims x %v %v y %v %v, want %v %v", key, i, x0, x1, y0, y1, w.Xlim, w.Ylim)
				}
				labels := func(ax *axis) []string {
					var out []string
					for _, tk := range ax.ticksOut.Located {
						out = append(out, tk.Label)
					}
					return out
				}
				if gx := labels(a.x); !a.xHidden && !slices.Equal(gx, w.Xticks) {
					t.Errorf("%s axes %d: x ticks %q, want %q", key, i, gx, w.Xticks)
				}
				if gy := labels(a.y); !slices.Equal(gy, w.Yticks) {
					t.Errorf("%s axes %d: y ticks %q, want %q", key, i, gy, w.Yticks)
				}
				if a.x.ticksOut.Offset != w.Xoffset && !a.xHidden || a.y.ticksOut.Offset != w.Yoffset {
					t.Errorf("%s axes %d: offsets %q %q, want %q %q", key, i, a.x.ticksOut.Offset, a.y.ticksOut.Offset, w.Xoffset, w.Yoffset)
				}
			}
		}
	}
}
