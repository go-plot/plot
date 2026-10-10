// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"bytes"
	"image/png"
	"math"
	"testing"

)

// render draws f both ways and checks that each output is well formed.
func renderBoth(t *testing.T, f *Figure) (*bytes.Buffer, []byte) {
	t.Helper()
	d := f.Display()
	img, err := png.Decode(bytes.NewReader(d["image/png"]))
	if err != nil {
		t.Fatal(err)
	}
	w, h := f.PixelSize()
	if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Fatalf("png is %v, want %dx%d", img.Bounds(), w, h)
	}
	svg := d["image/svg+xml"]
	if !bytes.HasPrefix(svg, []byte("<?xml")) || !bytes.HasSuffix(svg, []byte("</svg>\n")) {
		t.Fatalf("svg is not a document: %.80s", svg)
	}
	if bytes.Count(svg, []byte("<g")) != bytes.Count(svg, []byte("</g>")) {
		t.Fatal("unbalanced groups in svg")
	}
	return bytes.NewBuffer(d["image/png"]), svg
}

func seq(n int, f func(float64) float64) (x, y []float64) {
	x, y = make([]float64, n), make([]float64, n)
	for i := range n {
		x[i] = float64(i) / float64(n-1) * 10
		y[i] = f(x[i])
	}
	return x, y
}

// TestEverything builds a figure using every plotting method and option and
// renders it; each piece is checked more closely by the tests after it.
func TestEverything(t *testing.T) {
	fig, axs := Subplots(3, 3, FigSize(12, 10), DPI(80))
	x, y := seq(300, math.Sin)

	a := axs[0][0]
	a.Plot(x, y, Label("sin"), Color("C3"), LineWidth(2), LineStyle("-."), ZOrder(4), Clip(true))
	a.Plot(x, y, Label("_hidden"), LineStyle(":"), Alpha(0.5))
	a.Plot(nil, []float64{1, 0, 1}, Marker("o"), LineStyle("None"), MarkerSize(4), MarkerEdgeWidth(0.5), FaceColor("w"), EdgeColor("k"), Label("pts"))
	a.Plot([]float64{0, 5, math.NaN(), 7, 9}, []float64{0, 1, 2, math.Inf(1), 0}, LineStyle("dashed"), Clip(false))
	a.Axhline(0.5, Label("h"), ZOrder(1))
	a.Axvline(2, LineStyle("dotted"), ZOrder(1))
	a.Grid(true)
	a.SetTitle("$\\alpha^2$ and text")
	a.SetXLabel("x\nsecond line")
	a.SetYLabel("$\\frac{a}{b}$")
	a.Legend(Loc("lower left"), FontSize(8))

	b := axs[0][1]
	b.Bar([]float64{1, 2, 3}, []int{3, -1, 2}, Colors("r", "g"), EdgeColor("k"), LineWidth(2), Label("bars"), Align("edge"), Width(0.5), Bottom(0.5), Alpha(0.7), ZOrder(2))
	b.Barh([]float64{1, 2}, 1.5, Align("edge"), Width(0.3), FaceColor("y"), ZOrder(2), Label("h"))
	b.Bar([]float64{4}, []float64{1}, Color("purple"))
	b.Legend()

	c := axs[0][2]
	c.Hist([]float64{1, 2, 2, 3, 3, 3, math.NaN()}, Density(), Label("dens"), ZOrder(2))
	c.Hist([]float64{1, 2, 2, 3}, BinEdges(0, 1, 2.5, 4), Color("k"), Alpha(0.3))
	c.Hist([]float64{5, 5})
	c.Hist([]float64{})
	c.FillBetween([]float64{0, 1, 2, 3, 4}, []float64{1, 2, math.NaN(), 2, 1}, []float64{0, 0, 0, 0, 0}, Where([]bool{true, true, true, true, false}), EdgeColor("k"), Label("fill"), ZOrder(0))
	c.FillBetween([]float64{0, 1}, []float64{1, 1}, nil, FaceColor("c"), Alpha(0.2))
	c.Legend(Loc("best"))

	d := axs[1][0]
	m := d.Scatter(x[:50], y[:50], CValues(y[:50]), Sizes(y[:50]), Cmap("viridis_r"), Vmin(-0.5), Vmax(0.5), Marker("^"), Label("s"), ZOrder(3), LineWidth(0.5), EdgeColor("k"))
	d.Scatter([]float64{1, 2}, []float64{1, 2}, Colors("r", "b"), Alpha(0.5))
	d.Scatter([]float64{3}, []float64{0}, Color("k"), Marker("x"), Label("x"))
	d.Scatter([]float64{3, -1}, []float64{1, math.NaN()})
	d.Legend()
	fig.Colorbar(m, d)

	e := axs[1][1]
	z := make([][]float64, 20)
	for i := range z {
		z[i] = make([]float64, 30)
		for j := range z[i] {
			z[i][j] = math.Sin(float64(i)/3) * math.Cos(float64(j)/4)
		}
	}
	z[3][4] = math.NaN()
	im := e.Imshow(z, Cmap("RdBu"), Origin("lower"), Aspect("auto"), Alpha(0.9), ZOrder(0.5))
	fig.Colorbar(im, e)
	e.Text(3, 3, "text", FontSize(14), Color("w"), HA("center"), VA("center"), Rotation(30), Alpha(0.8), ZOrder(5))
	e.Annotate("here", 10, 10, 20, 15, Arrow(), Color("r"))
	e.Annotate("no arrow", 10, 10, 20, 5)
	e.Annotate("same", 10, 10, 10, 10, Arrow())

	f := axs[1][2]
	cx, cy := make([]float64, 30), make([]float64, 20)
	for j := range cx {
		cx[j] = float64(j)
	}
	for i := range cy {
		cy[i] = float64(i) * 2
	}
	cs := f.Contourf(cx, cy, z, NLevels(5), Cmap("magma"), Alpha(0.8), ZOrder(0))
	f.Contour(cx, cy, z, Levels(-0.5, 0, 0.5), Color("k"), LineWidth(1), ZOrder(3))
	f.Contour(nil, nil, z, Cmap("gray"))
	fig.Colorbar(cs, f)

	g := axs[2][0]
	lx, ly := seq(100, func(v float64) float64 { return math.Exp(v) })
	g.Plot(lx, ly)
	g.ErrorBar([]float64{1, 2, 3}, []float64{10, 100, 1000}, YErr([]float64{1, 10, 100}), XErr(0.2), CapSize(2), ErrorColor("r"), Label("err"))
	g.ErrorBar([]float64{1, 2}, []float64{5, 6}, YErr(1), Color("g"))
	g.SetYScale("log")
	g.SetXScale("log")
	g.Scatter([]float64{-1, 2}, []float64{3, -4})
	g.Legend(Loc("upper left"))
	g.Imshow(z, Extent(1, 5, 1, 1000), LogNorm(), Aspect("auto"))

	h := axs[2][1]
	h.Plot(x, y)
	h.SetXLim(10, 0)
	h.SetYLim(-2, 2)
	h.SetXTicks([]float64{0, 2.5, 5, 7.5, 10, 12})
	h.SetYTicks([]float64{-1, 0, 1}, "low", "mid")
	tw := h.TwinX()
	tw.Plot(x, y, Color("C1"))
	tw.SetYLabel("twin")
	tw.SetYScale("log")
	tw.SetYTicks([]float64{1e5, 2e5})

	k := axs[2][2]
	k.Imshow(z)
	k.SetAspect(2)
	k.SetTitle("title")

	pngb, svg := renderBoth(t, fig)
	dumpFigure(t, "everything", pngb.Bytes(), svg)
	if l, h := h.XLim(); l != 10 || h != 0 {
		t.Errorf("inverted xlim = %v %v", l, h)
	}
	if fig.Axes()[0] != a || a.Figure() != fig {
		t.Error("Axes/Figure accessors")
	}
}

// TestTeXLabels renders labels typeset by go-tex/math and checks that each
// produced ink of a plausible size.
func TestTeXLabels(t *testing.T) {
	fig, axs := Subplots(1, 1, FigSize(5, 3))
	a := axs[0][0]
	a.Plot([]float64{0, 1}, []float64{0, 1})
	a.SetTitle(`Energy $E = mc^2$ and $\alpha_i \leq \frac{1}{2}\sqrt{x}$`)
	a.SetXLabel("first line\nsecond line")
	a.SetYLabel(`$\int_0^\infty e^{-x^2}\,dx$`)
	a.Text(0.1, 0.7, `$\sum_{i=1}^n i = \frac{n(n+1)}{2}$`, FontSize(16))
	p, s := renderBoth(t, fig)
	dumpFigure(t, "tex", p.Bytes(), s)
	for _, src := range []string{`$\sum_{i=1}^n i$`, `$\frac{a}{b}$`, `$\alpha^2$`} {
		l := fig.Text.Layout(src, 20)
		x0, y0, x1, y1, ok := l.Path.Bounds()
		if !ok || x1-x0 < 5 || y1-y0 < 10 || l.Width < 5 {
			t.Errorf("%s: ink %v %v %v %v, width %v", src, x0, y0, x1, y1, l.Width)
		}
	}
}
