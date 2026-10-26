// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"fmt"
	"math"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
)

// legendEntry is what an artist shows in the legend: a label and a handle
// drawn like the artist.
type legendEntry struct {
	label   string
	line    *lineStyle
	patch   *patchStyle
	scatter *scatterArt
	plain   bool // the label is set without TeX
}

type legendSpec struct {
	loc      string
	fontSize float64
}

// legendLocs are matplotlib's location codes 1..10 and their anchors.
var legendLocs = []struct {
	name string
	ax   float64 // anchor x: 0 west, 0.5 centre, 1 east
	ay   float64 // anchor y: 0 top, 0.5 centre, 1 bottom (pixels, y down)
}{
	{"upper right", 1, 0}, {"upper left", 0, 0}, {"lower left", 0, 1}, {"lower right", 1, 1},
	{"right", 1, 0.5}, {"center left", 0, 0.5}, {"center right", 1, 0.5},
	{"lower center", 0.5, 1}, {"upper center", 0.5, 0}, {"center", 0.5, 0.5},
}

// Loc sets where Legend goes: "best" (the default: the place covering the
// least data, as matplotlib chooses it) or one of matplotlib's location
// names ("upper right", "lower center", ...).
func Loc(s string) Option { return func(p *props) { p.loc = s } }

// Legend adds a legend of every labelled artist, as matplotlib's
// ax.legend(): 10-point labels in a rounded, translucent frame.
//
// Options: Loc, FontSize.
func (a *Axes) Legend(opts ...Option) {
	p := collect(opts)
	loc := p.loc
	if loc == "" {
		loc = "best"
	}
	ok := loc == "best"
	for _, l := range legendLocs {
		ok = ok || l.name == loc
	}
	if !ok {
		panic(fmt.Sprintf("plot: unknown legend location %q", loc))
	}
	fs := defaultFontSize
	if p.fontSize > 0 {
		fs = p.fontSize
	}
	a.legend = &legendSpec{loc: loc, fontSize: fs}
}

type legendRow struct {
	e       *legendEntry
	txt     placedText
	asc, de float64 // text box extents above and below the baseline
	w       float64
}

// drawLegend lays out and draws the legend (z-order 5, after everything).
func (a *Axes) drawLegend(r render.Renderer) {
	f := a.fig
	var rows []legendRow
	for _, art := range a.artists {
		if e := art.legendEntry(); e != nil {
			rows = append(rows, legendRow{e: e})
		}
	}
	if len(rows) == 0 {
		return
	}
	F := f.pt(a.legend.fontSize)
	hl, hh, htp, ls, bp, bap := 2*F, 0.7*F, 0.8*F, 0.5*F, 0.4*F, 0.5*F
	textW := 0.0
	height := 0.0
	for i := range rows {
		pt := f.layoutText(textSpec{S: rows[i].e.label, Size: F, HA: "left", VA: "baseline", Plain: rows[i].e.plain})
		rows[i].txt = pt
		rows[i].asc = math.Max(hh, -pt.bbox.y0)
		rows[i].de = math.Max(0, pt.bbox.y1)
		rows[i].w = pt.bbox.x1 - pt.bbox.x0
		textW = math.Max(textW, rows[i].w)
		height += rows[i].asc + rows[i].de
	}
	height += float64(len(rows)-1)*ls + 2*bp
	width := hl + htp + textW + 2*bp

	x0, y0 := a.legendPosition(width, height, bap)
	// Frame: rounded, 0.8 alpha white face and #cccccc edge.
	frame := roundRect(x0, y0, width, height, 0.2*F)
	r.DrawPath(frame, render.Style{
		Fill:   colors.RGBA{R: 1, G: 1, B: 1, A: 0.8},
		Stroke: colors.RGBA{R: 0.8, G: 0.8, B: 0.8, A: 0.8},
		Width:  f.pt(1), Join: render.JoinMiter, Snap: true,
	})
	y := y0 + bp
	for _, row := range rows {
		base := y + row.asc
		hx := x0 + bp
		a.drawHandle(r, row.e, hx, base-hh, hl, hh)
		tx := hx + hl + htp
		var p render.Path
		p.Append(row.txt.path, render.Translate(math.Round(tx), math.Round(base)))
		r.DrawPath(&p, render.Style{Fill: black})
		y = base + row.de + ls
	}
}

// legendPosition picks the legend's top-left corner.
func (a *Axes) legendPosition(w, h, pad float64) (float64, float64) {
	cx0, cy0, cx1, cy1 := a.px.x0+pad, a.px.y0+pad, a.px.x1-pad, a.px.y1-pad
	at := func(k int) (float64, float64) {
		l := legendLocs[k]
		return cx0 + (cx1-cx0-w)*l.ax, cy0 + (cy1-cy0-h)*l.ay
	}
	if a.legend.loc != "best" {
		for k, l := range legendLocs {
			if l.name == a.legend.loc {
				return at(k)
			}
		}
	}
	verts, segs, boxes := a.legendObstacles()
	best, bestK := math.MaxInt, 0
	for k := range legendLocs {
		x, y := at(k)
		b := box{x, y, x + w, y + h}
		bad := 0
		for _, v := range verts {
			if v.X > b.x0 && v.X < b.x1 && v.Y > b.y0 && v.Y < b.y1 {
				bad++
			}
		}
		for _, o := range boxes {
			if o.x0 < b.x1 && o.x1 > b.x0 && o.y0 < b.y1 && o.y1 > b.y0 {
				bad++
			}
		}
		for _, s := range segs {
			if pathIntersectsBox(s, b) {
				bad++
			}
		}
		if bad == 0 {
			return x, y
		}
		if bad < best {
			best, bestK = bad, k
		}
	}
	return at(bestK)
}

// legendObstacles collects what "best" avoids, as matplotlib's
// _auto_legend_data: the vertices and paths of lines and polygons, the boxes
// of bars and texts, and the points of scatter plots.
func (a *Axes) legendObstacles() (verts []render.Pt, paths [][]render.Pt, boxes []box) {
	for _, art := range a.artists {
		switch v := art.(type) {
		case *line2D:
			pts := v.pixels(a)
			verts = append(verts, pts...)
			paths = append(paths, pts)
		case *polygons:
			for _, pg := range v.polys {
				var pts []render.Pt
				for _, q := range pg {
					x, y := a.toPx(q.X, q.Y)
					pts = append(pts, render.Pt{X: x, Y: y})
				}
				if len(pts) > 0 {
					pts = append(pts, pts[0])
				}
				verts = append(verts, pts...)
				paths = append(paths, pts)
			}
		case *bars:
			for _, rc := range v.rects {
				x0, y0 := a.toPx(rc[0], rc[1])
				x1, y1 := a.toPx(rc[0]+rc[2], rc[1]+rc[3])
				boxes = append(boxes, box{math.Min(x0, x1), math.Min(y0, y1), math.Max(x0, x1), math.Max(y0, y1)})
			}
		case *scatterArt:
			for i := range v.x {
				x, y := a.toPx(v.x[i], v.y[i])
				verts = append(verts, render.Pt{X: x, Y: y})
			}
		case *textArt:
			boxes = append(boxes, a.fig.layoutText(v.spec(a)).bbox)
		case *wedge:
			// matplotlib counts a non-rectangular patch as a line: its
			// vertices, and its path crossing the legend box.
			pts := v.pixels(a)
			verts = append(verts, pts...)
			paths = append(paths, pts)
		case *pieLabels:
			for _, t := range v.texts {
				boxes = append(boxes, a.fig.layoutText(t.spec(a)).bbox)
			}
		}
	}
	return verts, paths, boxes
}

// pathIntersectsBox reports whether a polyline touches the inside of b.
func pathIntersectsBox(pts []render.Pt, b box) bool {
	for i := 1; i < len(pts); i++ {
		if segmentIntersectsBox(pts[i-1], pts[i], b) {
			return true
		}
	}
	return false
}

// segmentIntersectsBox clips the segment to the box (Liang–Barsky).
func segmentIntersectsBox(p, q render.Pt, b box) bool {
	if math.IsNaN(p.X) || math.IsNaN(q.X) {
		return false
	}
	t0, t1 := 0.0, 1.0
	dx, dy := q.X-p.X, q.Y-p.Y
	for _, c := range [4][2]float64{{-dx, p.X - b.x0}, {dx, b.x1 - p.X}, {-dy, p.Y - b.y0}, {dy, b.y1 - p.Y}} {
		pp, qq := c[0], c[1]
		if pp == 0 {
			if qq < 0 {
				return false
			}
			continue
		}
		t := qq / pp
		if pp < 0 {
			t0 = math.Max(t0, t)
		} else {
			t1 = math.Min(t1, t)
		}
	}
	return t0 <= t1
}

// drawHandle draws an entry's handle in the box (x, y, w, h), y down.
func (a *Axes) drawHandle(r render.Renderer, e *legendEntry, x, y, w, h float64) {
	f := a.fig
	switch {
	case e.line != nil:
		cy := y + h/2
		if e.line.dash != "" {
			r.DrawPath(line(x, cy, x+w, cy), e.line.stroke(f))
		}
		e.line.drawMarkers(f, r, []render.Pt{{X: x + w/2, Y: cy}})
	case e.patch != nil:
		var p render.Path
		p.Rect(x, y, w, h)
		r.DrawPath(&p, e.patch.render(f))
	default:
		s := e.scatter
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, v := range s.sizes {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
		size := f.pt(math.Sqrt((lo + hi) / 2))
		face := s.faces[0]
		edge := face
		if s.edges != nil {
			edge = s.edges[0]
		}
		st := render.Style{Stroke: edge, Width: f.pt(s.lw), Join: render.JoinMiter}
		if s.marker.filled {
			st.Fill = face
		}
		r.DrawPath(s.marker.at(x+w/2, y+h/2, size), st)
	}
}

// roundRect is a rectangle with circular corners of radius rad.
func roundRect(x, y, w, h, rad float64) *render.Path {
	var p render.Path
	k := 0.5522847498307936 * rad
	p.MoveTo(x+rad, y)
	p.LineTo(x+w-rad, y)
	p.CubicTo(x+w-rad+k, y, x+w, y+rad-k, x+w, y+rad)
	p.LineTo(x+w, y+h-rad)
	p.CubicTo(x+w, y+h-rad+k, x+w-rad+k, y+h, x+w-rad, y+h)
	p.LineTo(x+rad, y+h)
	p.CubicTo(x+rad-k, y+h, x, y+h-rad+k, x, y+h-rad)
	p.LineTo(x, y+rad)
	p.CubicTo(x, y+rad-k, x+rad-k, y, x+rad, y)
	p.Close()
	return &p
}
