// Command perf times go-plot on the two benchmark figures: a line plot of a
// million points and a 1000 x 1000 imshow, each saved as PNG and as SVG.
// It prints one JSON line per measurement: {"case":..., "seconds":...}.
// The data are built before the clock starts; the clock covers building
// the figure and writing the file (to memory).
//
//	go run ./perf -case line_png
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"math"
	"os"
	"time"

	"github.com/go-plot/plot"
)

func main() {
	which := flag.String("case", "line_png", "line_png, line_svg, imshow_png or imshow_svg")
	write := flag.String("write", "", "also write the file here")
	flag.Parse()
	// The same deterministic data as perf.py.
	n := 1_000_000
	x := make([]float64, n)
	y := make([]float64, n)
	s := uint32(3)
	acc := 0.0
	for i := range n {
		s = 1664525*s + 1013904223
		acc += float64(s)/4294967296.0 - 0.5
		x[i], y[i] = float64(i), acc
	}
	img := make([][]float64, 1000)
	for i := range img {
		img[i] = make([]float64, 1000)
		for j := range img[i] {
			img[i][j] = math.Sin(float64(i)/37) * math.Cos(float64(j)/23)
		}
	}
	var buf bytes.Buffer
	start := time.Now()
	fig, axs := plot.Subplots(1, 1)
	switch *which {
	case "line_png", "line_svg":
		axs[0][0].Plot(x, y)
	default:
		axs[0][0].Imshow(img)
	}
	var err error
	if *which == "line_png" || *which == "imshow_png" {
		err = fig.SavePNG(&buf)
	} else {
		err = fig.SaveSVG(&buf)
	}
	el := time.Since(start).Seconds()
	if err != nil {
		panic(err)
	}
	if *write != "" {
		if err := os.WriteFile(*write, buf.Bytes(), 0o644); err != nil {
			panic(err)
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"case": *which, "seconds": el, "bytes": buf.Len()})
}
