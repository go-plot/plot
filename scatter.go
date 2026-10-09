// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// scatterArt is a set of markers with a size and colour each, as
// matplotlib's PathCollection from ax.scatter.
type scatterArt struct {
	x, y   []float64
	sizes  []float64 // points squared, one or per point
	faces  []colors.RGBA
	edges  []colors.RGBA // nil: the face colour ("face")
	marker *markerShape
	lw     float64 // points
	label  string
	z      float64
}

func (s *scatterArt) zorder() float64 { return s.z }

func (s *scatterArt) limits() lim {
	lm := emptyLim()
	for i := range s.x {
		lm.addXY(s.x[i], s.y[i])
	}
	return lm
}

func (s *scatterArt) sticky() (x, y []float64) { return nil, nil }

func pick[T any](v []T, i int) T {
	if len(v) == 1 {
		return v[0]
	}
	return v[i]
}

func (s *scatterArt) draw(a *Axes, r render.Renderer) {
	a.clipToAxes(r)
	defer r.PopClip()
	for i := range s.x {
		x, y := s.x[i], s.y[i]
		if math.IsNaN(x) || math.IsNaN(y) || (a.x.log && x <= 0) || (a.y.log && y <= 0) {
			continue
		}
		px, py := a.toPx(x, y)
		size := a.fig.pt(math.Sqrt(pick(s.sizes, i)))
		face := pick(s.faces, i)
		edge := face
		if s.edges != nil {
			edge = pick(s.edges, i)
		}
		st := render.Style{Stroke: edge, Width: a.fig.pt(s.lw), Join: render.JoinMiter}
		if s.marker.filled {
			st.Fill = face
		}
		r.DrawPath(s.marker.at(px, py, size), st)
	}
}

func (s *scatterArt) legendEntry() *legendEntry {
	if s.label == "" {
		return nil
	}
	return &legendEntry{label: s.label, scatter: s}
}

// Scatter draws a marker at each (x, y), as matplotlib's ax.scatter: circles
// of 36 points² in the next colour of the fill cycle, edged in their own
// colour. With CValues the colours come from a colormap, and the returned
// Mappable can be given to Colorbar; otherwise it is nil.
//
// Options: Sizes, CValues, Cmap, Vmin, Vmax, LogNorm, Color, Colors, Marker,
// EdgeColor, LineWidth (edge), Alpha, Label, ZOrder.
func (a *Axes) Scatter(x, y any, opts ...Option) *Mappable {
	xs, ys := Values(x), Values(y)
	mustSameLen("Scatter", xs, ys)
	p := collect(opts)
	s := &scatterArt{x: xs, y: ys, sizes: []float64{36}, lw: 1, label: p.label, z: 1, marker: getMarker("o")}
	if p.setMarker && p.marker != "" {
		s.marker = getMarker(p.marker)
	}
	if p.sizes != nil {
		s.sizes = broadcast(p.sizes, len(xs), "Scatter sizes")
	}
	if p.lineWidth > 0 {
		s.lw = p.lineWidth
	}
	if p.setZorder {
		s.z = p.zorder
	}
	var m *Mappable
	switch {
	case p.cvalues != nil:
		cv := broadcast(p.cvalues, len(xs), "Scatter CValues")
		m = newMappable(p, cv)
		s.faces = make([]colors.RGBA, len(cv))
		for i, v := range cv {
			s.faces[i] = m.color(v)
		}
	case p.colorsPer != nil:
		s.faces = append([]colors.RGBA(nil), p.colorsPer...)
	case p.setColor:
		s.faces = []colors.RGBA{p.color}
	default:
		s.faces = []colors.RGBA{a.nextFillColor()}
	}
	if p.setAlpha {
		for i := range s.faces {
			s.faces[i].A = p.alpha
		}
	}
	if !s.marker.filled {
		// Unfilled markers ("+", "x") are drawn with their face colour.
		s.edges = s.faces
	} else if p.setEdge {
		s.edges = []colors.RGBA{p.edgeColor}
	}
	a.add(s)
	return m
}
