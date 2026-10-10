// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"
	"slices"
	"testing"

	"github.com/go-ndarray/ndarray"
	"github.com/go-plot/plot/render"
)

func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	f()
}

func TestValues(t *testing.T) {
	want := []float64{1, 2, 3}
	f64, _ := ndarray.FromSlice([]float64{1, 2, 3}, 3)
	i16, _ := ndarray.FromSlice([]int16{1, 2, 3}, 3)
	for _, v := range []any{want, []float32{1, 2, 3}, []int{1, 2, 3}, []int64{1, 2, 3}, []int32{1, 2, 3}, f64, i16} {
		if got := Values(v); !slices.Equal(got, want) {
			t.Errorf("Values(%T) = %v", v, got)
		}
	}
	if Values(nil) != nil || Values(2.5)[0] != 2.5 || Values(3)[0] != 3 {
		t.Error("scalars")
	}
	mustPanic(t, "Values(string)", func() { Values("x") })
	mustPanic(t, "ragged", func() { grid([][]float64{{1, 2}, {3}}) })
	mustPanic(t, "1-D grid", func() { grid(f64) })
	mustPanic(t, "grid(int)", func() { grid(3) })
	two, _ := ndarray.FromSlice([]int8{1, 2, 3, 4, 5, 6}, 2, 3)
	if g := grid(two); g.Rows != 2 || g.Cols != 3 || g.Z[5] != 6 {
		t.Errorf("grid %+v", g)
	}
	mustPanic(t, "Plot lengths", func() { NewFigure().AddAxes(0, 0, 1, 1).Plot([]float64{1}, []float64{1, 2}) })
	mustPanic(t, "Bar lengths", func() { NewFigure().AddAxes(0, 0, 1, 1).Bar([]float64{1, 2}, []float64{1, 2, 3}) })
	mustPanic(t, "contour shape", func() { NewFigure().AddAxes(0, 0, 1, 1).Contour([]float64{1}, nil, [][]float64{{1, 2}}) })
	mustPanic(t, "line style", func() { LineStyle("~~")(&props{}); validLineStyle("~~") })
	mustPanic(t, "marker", func() { getMarker("?") })
	mustPanic(t, "legend loc", func() { NewFigure().AddAxes(0, 0, 1, 1).Legend(Loc("nowhere")) })
}

func TestLineStylesAndMarkers(t *testing.T) {
	for in, want := range map[string]string{"solid": "-", "dashed": "--", "dashdot": "-.", "dotted": ":", "none": "", "-.": "-."} {
		if got := validLineStyle(in); got != want {
			t.Errorf("%q -> %q", in, got)
		}
	}
	fig, axs := Subplots(1, 1)
	ax := axs[0][0]
	for i, m := range []string{"o", ".", ",", "s", "^", "v", "<", ">", "D", "d", "p", "h", "*", "+", "x", "|", "_"} {
		ax.Plot([]float64{float64(i), math.NaN()}, []float64{1, 1}, Marker(m), LineStyle("solid"), RGBA(0, 0, 1, 1), Label(m))
	}
	ax.Legend()
	p, s := renderBoth(t, fig)
	dumpFigure(t, "markers", p.Bytes(), s)
	// Star and pentagon are matplotlib's unit polygons: radius 1 at the top.
	star := polygon(5, 0.381966, 1)
	if star.Segs[0].P[0].Y != 1 || len(star.Segs) != 11 {
		t.Errorf("star %v", star.Segs[:2])
	}
}

func TestImages(t *testing.T) {
	rgb, _ := ndarray.FromSlice([]uint8{255, 0, 0, 0, 255, 0, 0, 0, 255, 9, 9, 9}, 2, 2, 3)
	rgbaF, _ := ndarray.FromSlice([]float64{1, 0, 0, 1, 0, 1, 0, 0.5, 2, -1, 0, 1, 0, 0, 0, 0}, 2, 2, 4)
	big := make([][]float64, 600)
	for i := range big {
		big[i] = make([]float64, 700)
		for j := range big[i] {
			big[i][j] = float64(i ^ j)
		}
	}
	fig, axs := Subplots(2, 2)
	axs[0][0].Imshow(rgb)
	axs[0][1].Imshow(rgbaF, Alpha(0.5), Extent(2, 0, 0, 2)) // right to left: flipped
	two, _ := ndarray.FromSlice([]float32{1, 2, 3, 4}, 2, 2)
	axs[0][0].Imshow(two, Alpha(0))
	axs[1][0].Imshow(big) // more pixels than the axes: averaged
	m := axs[1][1].Imshow([][]float64{{math.NaN(), math.NaN()}, {math.NaN(), math.NaN()}}, LogNorm())
	axs[1][1].SetYLim(-0.5, 1.5) // undo the inversion: rows flipped
	fig.Colorbar(m, axs[1][1])
	p, s := renderBoth(t, fig)
	dumpFigure(t, "images", p.Bytes(), s)
	if lo, hi := m.Norm.Range(); lo != 0 || hi != 1 {
		t.Errorf("all-NaN norm %v %v", lo, hi)
	}
	img := fig.Image().Img
	// The RGB image's top-left pixel is red.
	x, y := axs[0][0].toPx(0, 0)
	if c := img.RGBAAt(int(x), int(y)); c.R < 250 || c.G > 5 {
		t.Errorf("rgb pixel %v", c)
	}
	// Averaging keeps the mean.
	small := boxDownsample(fig.Image().Img, 2, 2)
	if small.Rect.Dx() != 2 {
		t.Error("downsample size")
	}
}

func TestOffsetsAndTitle(t *testing.T) {
	fig, axs := Subplots(1, 2)
	a := axs[0][0]
	x := []float64{1e5 + 1, 1e5 + 2, 1e5 + 3}
	a.Plot(x, []float64{1e-7, 3e-7, 2e-7})
	a.SetTitle("over the offset")
	a.SetYLabel("tiny")
	b := axs[0][1]
	b.Plot(x, x)
	tw := b.TwinX()
	tw.Plot(x, []float64{1e7, 2e7, 3e7})
	tw.SetYLabel("right")
	tw.SetTitle("twin")
	renderBoth(t, fig)
	fig.layout()
	if a.x.ticksOut.Offset == "" || a.y.ticksOut.Offset == "" || tw.y.ticksOut.Offset == "" {
		t.Errorf("offsets %q %q %q", a.x.ticksOut.Offset, a.y.ticksOut.Offset, tw.y.ticksOut.Offset)
	}
}

func TestLogAxesEdgeCases(t *testing.T) {
	for _, c := range []struct{ lo, hi, minpos, wlo, whi float64 }{
		{5, 1, 1, 1, 5},
		{math.Inf(1), 1, 1, 1, 10},
		{-5, -1, 1, 1, 10},
		{0, 10, math.Inf(1), 1e-300, 10},
		{3, 3, 1, 1, 10},
		{10, 10, 1, 1, 100},
	} {
		lo, hi := logNonsingular(c.lo, c.hi, c.minpos)
		if lo != c.wlo || hi != c.whi {
			t.Errorf("logNonsingular(%v, %v, %v) = %v %v", c.lo, c.hi, c.minpos, lo, hi)
		}
	}
	fig, axs := Subplots(1, 1)
	a := axs[0][0]
	a.FillBetween([]float64{1, 10, 100}, []float64{1, 5, 2}, 0.0) // to zero on a log axis
	a.SetYScale("log")
	a.SetXScale("log")
	a.SetXLim(1, 100)
	m := a.Scatter([]float64{2, 3}, []float64{2, 3}, CValues([]float64{1, 100}), LogNorm())
	fig.Colorbar(m, a)
	a.Axhline(math.Inf(1))
	a.Axvline(math.NaN())
	renderBoth(t, fig)
	// Data so wide that the 5% margin overflows keeps its limits.
	f2, ax2 := Subplots(1, 1)
	ax2[0][0].Plot([]float64{-1e308, 1e308}, []float64{0, 1})
	f2.layout()
	if lo, hi := ax2[0][0].XLim(); lo != -1e308 || hi != 1e308 {
		t.Errorf("huge limits %v %v", lo, hi)
	}
}

func TestLegendPlacement(t *testing.T) {
	// Data everywhere: every location collides, the least bad wins.
	fig, axs := Subplots(1, 1)
	a := axs[0][0]
	var xs, ys []float64
	for i := range 21 {
		for j := range 21 {
			xs = append(xs, float64(i))
			ys = append(ys, float64(j))
		}
	}
	a.Scatter(xs, ys, Label("grid"))
	a.Plot([]float64{0, 20}, []float64{0, 20}, Label("diag"))
	a.Bar([]float64{10}, []float64{20}, Label("bar"))
	a.Text(18, 18, "corner")
	a.FillBetween([]float64{0, 20}, []float64{0, 20}, nil, Label("fill"))
	a.FillBetween([]float64{}, []float64{}, nil)
	a.Legend()
	renderBoth(t, fig)
	// Nothing labelled: no legend at all.
	f2, ax2 := Subplots(1, 1)
	ax2[0][0].Plot([]float64{0, 1}, []float64{0, 1})
	ax2[0][0].Legend()
	renderBoth(t, f2)
	if pathIntersectsBox([]render.Pt{{X: 0, Y: 0}, {X: 1, Y: 0}}, box{2, 2, 3, 3}) {
		t.Error("segment far from box intersects")
	}
	if segmentIntersectsBox(render.Pt{X: 0, Y: 5}, render.Pt{X: 10, Y: 5}, box{2, 6, 3, 7}) {
		t.Error("parallel segment outside intersects")
	}
	if segmentIntersectsBox(render.Pt{X: math.NaN()}, render.Pt{}, box{0, 0, 1, 1}) {
		t.Error("NaN segment intersects")
	}
}

func TestHistogramEdges(t *testing.T) {
	// Equal bins whose arithmetic lands one bin off are corrected both ways.
	got := histogram([]float64{2, 6, 10}, []float64{0, 9, 10}, true)
	if !slices.Equal(got, []float64{2, 1}) {
		t.Errorf("uniform correction down %v", got)
	}
	got = histogram([]float64{0.5, 4, 6}, []float64{0, 1, 10}, true)
	if !slices.Equal(got, []float64{1, 2}) {
		t.Errorf("uniform correction %v", got)
	}
	got = histogram([]float64{0, 1, 4, 4, 5}, []float64{0, 1, 2.5, 4}, false)
	if !slices.Equal(got, []float64{1, 1, 2}) {
		t.Errorf("explicit edges %v", got)
	}
	if l := linspace(0, 1, 5); !slices.Equal(l, []float64{0, 0.25, 0.5, 0.75, 1}) {
		t.Errorf("linspace %v", l)
	}
}

func TestSimplify(t *testing.T) {
	// A line that runs forward, back past its start, and forward again,
	// all on one horizontal: simplification keeps its full extent.
	var pts []render.Pt
	for i := range 200 {
		x := float64(i % 50)
		if (i/50)%2 == 1 {
			x = 50 - x - 10
		}
		pts = append(pts, render.Pt{X: x, Y: 10 + 0.001*float64(i%3)})
	}
	pts = append(pts, render.Pt{X: 5, Y: 10}, render.Pt{X: 5, Y: 10}, render.Pt{X: 5, Y: 30})
	out := simplify(pts)
	if len(out) >= len(pts) {
		t.Errorf("not simplified: %d -> %d", len(pts), len(out))
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range out {
		lo, hi = math.Min(lo, p.X), math.Max(hi, p.X)
	}
	if lo != -9 || hi != 49 || out[len(out)-1] != (render.Pt{X: 5, Y: 30}) {
		t.Errorf("extent %v..%v, end %v", lo, hi, out[len(out)-1])
	}
	// Ending on a backward run.
	var back []render.Pt
	for i := range 130 {
		back = append(back, render.Pt{X: float64(i), Y: 0})
	}
	back = append(back, render.Pt{X: -5, Y: 0}, render.Pt{X: 3, Y: 0})
	out = simplify(back)
	if out[len(out)-1] != (render.Pt{X: 3, Y: 0}) {
		t.Errorf("end %v", out)
	}
	// Forward, back, then forward past the first maximum, then a turn: the
	// backward extreme is written before the forward one.
	var fb []render.Pt
	for i := 0; i <= 60; i++ {
		fb = append(fb, render.Pt{X: float64(i), Y: 0})
	}
	for i := 59; i >= -30; i-- {
		fb = append(fb, render.Pt{X: float64(i), Y: 0})
	}
	for i := -29; i <= 90; i++ {
		fb = append(fb, render.Pt{X: float64(i), Y: 0})
	}
	fb = append(fb, render.Pt{X: 90, Y: 50})
	out = simplify(fb)
	want := []render.Pt{{X: 0}, {X: -30}, {X: 90}, {X: 90, Y: 50}}
	if !slices.Equal(out, want) {
		t.Errorf("forward-back-forward: %v", out)
	}
	out = simplify(fb[:len(fb)-1])
	if !slices.Equal(out, want[:3]) {
		t.Errorf("forward-back-forward at the end: %v", out)
	}
	if short := simplify(back[:10]); len(short) != 10 {
		t.Error("short line simplified")
	}
}

func TestTextAlignment(t *testing.T) {
	f := NewFigure()
	for _, ha := range []string{"left", "center", "right"} {
		for _, va := range []string{"top", "bottom", "center", "baseline", "center_baseline"} {
			for _, anchor := range []bool{false, true} {
				p := f.layoutText(textSpec{S: "Ab\n\ncd", X: 100, Y: 100, Size: 12, HA: ha, VA: va, Rot: 30, Anchor: anchor})
				if p.bbox.x1 <= p.bbox.x0 || p.bbox.y1 <= p.bbox.y0 {
					t.Errorf("%s %s %v: empty box %v", ha, va, anchor, p.bbox)
				}
			}
		}
	}
	// What has no place in a legend says so.
	for _, a := range []artist{&textArt{}, &contourArt{}, &colorbarArt{}, colorbarOutline{}} {
		if a.legendEntry() != nil {
			t.Errorf("%T in legend", a)
		}
	}
	for _, a := range []artist{&colorbarArt{}, colorbarOutline{}} {
		if x, y := a.sticky(); x != nil || y != nil {
			t.Errorf("%T sticky", a)
		}
	}
	if lv := autoLevels(1, 1+1e-12, 7); len(lv) < 3 {
		t.Errorf("levels %v", lv)
	}
}
