// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"fmt"
	"math"

	"github.com/go-plot/plot/render"
)

// markerShape is a marker as matplotlib defines it: an outline in units of
// the marker size (y up), and whether it is filled.
type markerShape struct {
	path   *render.Path // y up, unit = marker size
	filled bool
	pixel  bool // "," is one pixel whatever the size
}

// polygon returns matplotlib's unit_regular_polygon(n) (radius 1, a vertex
// at the top) or, with inner > 0, unit_regular_star(n, inner), scaled by s.
func polygon(n int, inner, s float64) *render.Path {
	var p render.Path
	k := n
	if inner > 0 {
		k = 2 * n
	}
	for i := 0; i <= k; i++ {
		th := 2*math.Pi/float64(k)*float64(i) + math.Pi/2
		r := 1.0
		if inner > 0 && i%2 == 1 {
			r = inner
		}
		x, y := r*math.Cos(th)*s, r*math.Sin(th)*s
		if i == 0 {
			p.MoveTo(x, y)
		} else if i < k {
			p.LineTo(x, y)
		}
	}
	p.Close()
	return &p
}

func transformed(p *render.Path, m render.Affine) *render.Path {
	var q render.Path
	q.Append(p, m)
	return &q
}

// getMarker returns the named marker, or panics for an unknown name.
func getMarker(name string) *markerShape {
	var p render.Path
	switch name {
	case "o":
		p.Circle(0, 0, 0.5)
	case ".":
		p.Circle(0, 0, 0.25)
	case ",":
		p.Rect(0, 0, 1, 1)
		return &markerShape{path: &p, filled: true, pixel: true}
	case "s":
		p.Rect(-0.5, -0.5, 1, 1)
	case "^", "v", "<", ">":
		var t render.Path
		t.MoveTo(0, 1)
		t.LineTo(-1, -1)
		t.LineTo(1, -1)
		t.Close()
		rot := map[string]float64{"^": 0, "v": 180, "<": 90, ">": 270}[name]
		// matplotlib rotates counter-clockwise, y up.
		s, c := math.Sincos(rot * math.Pi / 180)
		return &markerShape{path: transformed(&t, render.Scale(0.5, 0.5).Then(render.Affine{A: c, B: s, C: -s, D: c})), filled: true}
	case "D", "d":
		var q render.Path
		q.Rect(-0.5, -0.5, 1, 1)
		s, c := math.Sincos(math.Pi / 4)
		m := render.Affine{A: c, B: s, C: -s, D: c}
		if name == "d" {
			m = m.Then(render.Scale(0.6, 1))
		}
		return &markerShape{path: transformed(&q, m), filled: true}
	case "p":
		return &markerShape{path: polygon(5, 0, 0.5), filled: true}
	case "h":
		return &markerShape{path: polygon(6, 0, 0.5), filled: true}
	case "*":
		return &markerShape{path: polygon(5, 0.381966, 0.5), filled: true}
	case "+":
		p.MoveTo(-0.5, 0)
		p.LineTo(0.5, 0)
		p.MoveTo(0, -0.5)
		p.LineTo(0, 0.5)
		return &markerShape{path: &p}
	case "x":
		p.MoveTo(-0.5, -0.5)
		p.LineTo(0.5, 0.5)
		p.MoveTo(-0.5, 0.5)
		p.LineTo(0.5, -0.5)
		return &markerShape{path: &p}
	case "|":
		p.MoveTo(0, -0.5)
		p.LineTo(0, 0.5)
		return &markerShape{path: &p}
	case "_":
		p.MoveTo(-0.5, 0)
		p.LineTo(0.5, 0)
		return &markerShape{path: &p}
	default:
		panic(fmt.Sprintf("plot: unknown marker %q", name))
	}
	return &markerShape{path: &p, filled: true}
}

// at returns the marker scaled to sizePx pixels and centred on (x, y) pixels.
func (m *markerShape) at(x, y, sizePx float64) *render.Path {
	if m.pixel {
		return transformed(m.path, render.Translate(math.Floor(x), math.Floor(y)))
	}
	return transformed(m.path, render.Scale(sizePx, -sizePx).Then(render.Translate(x, y)))
}
