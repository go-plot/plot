// Command compare renders the reference figures with go-plot. figures.py
// renders the same figures (same data, size and dpi) with matplotlib, and
// compare.py measures the two against each other.
//
//	go run . -out DIR
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-ndarray/ndarray"
	"github.com/go-plot/plot"
)

func linspace(a, b float64, n int) []float64 {
	out := make([]float64, n)
	step := (b - a) / float64(n-1)
	for i := range out {
		out[i] = float64(float64(i)*step) + a
	}
	out[n-1] = b
	return out
}

func apply(xs []float64, f func(float64) float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

// lcg is the deterministic generator figures.py also implements, so both
// sides plot the same "random" data.
type lcg struct{ s uint32 }

func (g *lcg) uniform() float64 {
	g.s = 1664525*g.s + 1013904223
	return float64(g.s) / 4294967296.0
}

func (g *lcg) normal() float64 {
	u1, u2 := g.uniform(), g.uniform()
	return math.Sqrt(-2*math.Log(1-u1)) * math.Cos(2*math.Pi*u2)
}

func field(nx, ny int) (xs, ys []float64, z [][]float64) {
	xs, ys = linspace(-3, 3, nx), linspace(-2, 2, ny)
	z = make([][]float64, ny)
	for i, y := range ys {
		z[i] = make([]float64, nx)
		for j, x := range xs {
			z[i][j] = (1 - x/2 + math.Pow(x, 5) + math.Pow(y, 3)) * math.Exp(-x*x-y*y)
		}
	}
	return xs, ys, z
}

// Figures are the reference figures; figures.py draws the same ones.
var Figures = map[string]func() *plot.Figure{
	"line": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		ax := axs[0][0]
		x := linspace(0, 10, 200)
		ax.Plot(x, apply(x, math.Sin), plot.Label("sin"))
		ax.Plot(x, apply(x, math.Cos), plot.Label("cos"), plot.LineStyle("--"))
		ax.SetTitle("Line plot")
		ax.SetXLabel("time (s)")
		ax.SetYLabel("amplitude")
		ax.Legend()
		return fig
	},
	"scatter": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		ax := axs[0][0]
		g := &lcg{s: 1}
		n := 150
		x, y, c, s := make([]float64, n), make([]float64, n), make([]float64, n), make([]float64, n)
		for i := range n {
			x[i], y[i] = g.normal(), g.normal()
			c[i] = x[i] + y[i]
			s[i] = 10 + 90*g.uniform()
		}
		m := ax.Scatter(x, y, plot.CValues(c), plot.Sizes(s), plot.Cmap("plasma"))
		fig.Colorbar(m, ax)
		ax.SetTitle("Scatter")
		return fig
	},
	"bars": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 2, plot.FigSize(8, 4))
		axs[0][0].Bar([]float64{1, 2, 3, 4, 5}, []float64{3, 7, 2, 5, 4}, plot.Label("a"))
		axs[0][0].Bar([]float64{1, 2, 3, 4, 5}, []float64{1, 2, 1, 2, 1}, plot.Bottom(0), plot.Width(0.4), plot.Label("b"))
		axs[0][0].Legend()
		axs[0][1].Barh([]float64{0, 1, 2}, []float64{10, 25, 17})
		axs[0][1].SetTitle("barh")
		return fig
	},
	"hist": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		g := &lcg{s: 7}
		v := make([]float64, 2000)
		for i := range v {
			v[i] = 5 + 2*g.normal()
		}
		axs[0][0].Hist(v, plot.Bins(30))
		axs[0][0].SetXLabel("value")
		axs[0][0].SetYLabel("count")
		return fig
	},
	"imshow": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		_, _, z := field(40, 30)
		m := axs[0][0].Imshow(z)
		fig.Colorbar(m, axs[0][0])
		return fig
	},
	"contour": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		xs, ys, z := field(80, 60)
		m := axs[0][0].Contourf(xs, ys, z, plot.Cmap("coolwarm"))
		axs[0][0].Contour(xs, ys, z, plot.Color("k"), plot.LineWidth(0.8))
		fig.Colorbar(m, axs[0][0])
		return fig
	},
	"log": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		x := linspace(0.1, 100, 300)
		ax := axs[0][0]
		ax.Plot(x, apply(x, func(v float64) float64 { return v * v }))
		ax.Plot(x, apply(x, func(v float64) float64 { return math.Exp(v / 10) }))
		ax.SetXScale("log")
		ax.SetYScale("log")
		ax.Grid(true)
		return fig
	},
	"misc": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		ax := axs[0][0]
		x := linspace(0, 4, 9)
		y := apply(x, func(v float64) float64 { return v*v/4 + 1 })
		ax.ErrorBar(x, y, plot.YErr(0.3), plot.CapSize(3), plot.Label("data"))
		ax.FillBetween(x, apply(y, func(v float64) float64 { return v - 0.5 }), apply(y, func(v float64) float64 { return v + 0.5 }), plot.Alpha(0.3))
		ax.Axhline(2, plot.Color("gray"), plot.LineStyle(":"))
		ax.Axvline(1.5, plot.Color("r"))
		ax.Text(0.2, 4.5, "a note")
		ax.Annotate("peak", 4, 5, 2.5, 5.5, plot.Arrow())
		ax.SetTitle("Errors, fills and lines")
		return fig
	},
	"twinx": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		ax := axs[0][0]
		x := linspace(0, 10, 50)
		ax.Plot(x, apply(x, math.Exp), plot.Color("C0"))
		ax.SetYLabel("exp")
		t := ax.TwinX()
		t.Plot(x, apply(x, math.Sin), plot.Color("C1"))
		t.SetYLabel("sin")
		ax.SetXLabel("x")
		return fig
	},
	"grid": func() *plot.Figure {
		fig, axs := plot.Subplots(2, 2)
		x := linspace(-1, 1, 101)
		axs[0][0].Plot(x, apply(x, func(v float64) float64 { return v * v * v }))
		axs[0][1].Plot(x, apply(x, func(v float64) float64 { return 1e-6 * v }))
		axs[1][0].Plot(x, apply(x, func(v float64) float64 { return 1000 + v }))
		axs[1][1].Scatter(x, apply(x, math.Abs), plot.Marker("s"))
		for _, row := range axs {
			for _, a := range row {
				a.Grid(true)
			}
		}
		return fig
	},
	"ndarray": func() *plot.Figure {
		fig, axs := plot.Subplots(1, 1)
		a, _ := ndarray.FromSlice([]int32{3, 1, 4, 1, 5, 9, 2, 6, 5, 3}, 10)
		axs[0][0].Plot(nil, a, plot.Marker("o"))
		return fig
	},
}

// Perturbed are negative controls: a reference figure with one thing moved
// on purpose, which the comparison must flag.
var Perturbed = map[string]func() *plot.Figure{
	"line_shifted": func() *plot.Figure {
		fig := Figures["line"]()
		// The same plot with the axes 3 pixels to the right and the x
		// limits slightly wider: every tick moves.
		ax := fig.Axes()[0]
		ax.SetXLim(-0.6, 10.5)
		return fig
	},
	"line_1px": func() *plot.Figure {
		fig := Figures["line"]()
		// The default view is [-0.5, 10.5] (5% margins); sliding it by
		// 0.03 data units moves every x tick by 1.35 pixels.
		fig.Axes()[0].SetXLim(-0.53, 10.47)
		return fig
	},
}

func main() {
	out := flag.String("out", "", "output directory")
	flag.Parse()
	var names []string
	for n := range Figures {
		names = append(names, n)
	}
	sort.Strings(names)
	render := func(name string, fig *plot.Figure) {
		for _, ext := range []string{"png", "svg"} {
			f, err := os.Create(filepath.Join(*out, "go_"+name+"."+ext))
			if err != nil {
				panic(err)
			}
			if ext == "png" {
				err = fig.SavePNG(f)
			} else {
				err = fig.SaveSVG(f)
			}
			if err != nil {
				panic(err)
			}
			f.Close()
		}
		fmt.Println("wrote", name)
	}
	for _, n := range names {
		render(n, Figures[n]())
	}
	for n, mk := range Perturbed {
		render(n, mk())
	}
}
