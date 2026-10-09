// Command compare renders the reference figures with go-plot, for
// comparison with the same figures rendered by matplotlib (figures.py).
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/go-plot/plot"
)

func linspace(a, b float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = a + (b-a)*float64(i)/float64(n-1)
	}
	return out
}

func apply(xs []float64, f func(float64) float64) []float64 {
	out := make([]float64, len(xs))
	for i, x := range xs {
		out[i] = f(x)
	}
	return out
}

// figures are the reference figures; figures.py draws the same ones.
var figures = map[string]func() *plot.Figure{
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
}

func main() {
	out := flag.String("out", "", "output directory")
	flag.Parse()
	for name, mk := range figures {
		fig := mk()
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
}
