// Command readme draws the figure shown in the README:
//
//	cd tools/compare && go run ./readme > ../../docs/readme.svg
package main

import (
	"math"
	"os"

	"github.com/go-plot/plot"
)

func main() {
	x := make([]float64, 200)
	for i := range x {
		x[i] = float64(i) / 199 * 10
	}
	sin, damped := make([]float64, len(x)), make([]float64, len(x))
	for i, v := range x {
		sin[i] = math.Sin(v)
		damped[i] = math.Exp(-v/4) * math.Cos(2*v)
	}

	fig, axs := plot.Subplots(1, 1, plot.FigSize(6.4, 4))
	ax := axs[0][0]
	ax.Plot(x, sin, plot.Label(`$\sin t$`))
	ax.Plot(x, damped, plot.Label(`$e^{-t/4}\cos 2t$`), plot.LineStyle("--"))
	ax.FillBetween(x, damped, nil, plot.Alpha(0.2))
	ax.SetTitle(`Two signals, labelled in TeX`)
	ax.SetXLabel(`time $t$ (s)`)
	ax.SetYLabel("amplitude")
	ax.Grid(true)
	ax.Legend()
	if err := fig.SaveSVG(os.Stdout); err != nil {
		panic(err)
	}
}
