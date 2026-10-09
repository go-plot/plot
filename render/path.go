// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package render is the drawing seam between the figure model and its output
// formats. A figure is laid out once, in pixels with y pointing down, and
// drawn through the Renderer interface; Raster and SVG implement it.
package render

import "math"

// Op is a path operation.
type Op uint8

// Path operations. A Seg's P holds 1 point for MoveTo and LineTo, 2 for
// QuadTo (control, end), 3 for CubicTo and none for Close.
const (
	MoveTo Op = iota
	LineTo
	QuadTo
	CubicTo
	Close
)

// Pt is a point in pixels.
type Pt struct{ X, Y float64 }

// Seg is one path segment.
type Seg struct {
	Op Op
	P  [3]Pt
}

// Path is an outline in pixel coordinates.
type Path struct{ Segs []Seg }

// MoveTo starts a new subpath.
func (p *Path) MoveTo(x, y float64) { p.Segs = append(p.Segs, Seg{Op: MoveTo, P: [3]Pt{{x, y}}}) }

// LineTo adds a straight segment.
func (p *Path) LineTo(x, y float64) { p.Segs = append(p.Segs, Seg{Op: LineTo, P: [3]Pt{{x, y}}}) }

// QuadTo adds a quadratic Bézier segment.
func (p *Path) QuadTo(cx, cy, x, y float64) {
	p.Segs = append(p.Segs, Seg{Op: QuadTo, P: [3]Pt{{cx, cy}, {x, y}}})
}

// CubicTo adds a cubic Bézier segment.
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) {
	p.Segs = append(p.Segs, Seg{Op: CubicTo, P: [3]Pt{{c1x, c1y}, {c2x, c2y}, {x, y}}})
}

// Close closes the current subpath.
func (p *Path) Close() { p.Segs = append(p.Segs, Seg{Op: Close}) }

// Rect adds a closed rectangle.
func (p *Path) Rect(x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.Close()
}

// kappa is the cubic Bézier control distance for a quarter circle.
const kappa = 0.5522847498307936

// Circle adds a closed circle of radius r about (cx, cy), as four cubics.
func (p *Path) Circle(cx, cy, r float64) {
	k := kappa * r
	p.MoveTo(cx+r, cy)
	p.CubicTo(cx+r, cy+k, cx+k, cy+r, cx, cy+r)
	p.CubicTo(cx-k, cy+r, cx-r, cy+k, cx-r, cy)
	p.CubicTo(cx-r, cy-k, cx-k, cy-r, cx, cy-r)
	p.CubicTo(cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	p.Close()
}

// Append adds every segment of q, transformed by m.
func (p *Path) Append(q *Path, m Affine) {
	for _, s := range q.Segs {
		for i := range s.P {
			s.P[i] = m.Apply(s.P[i])
		}
		p.Segs = append(p.Segs, s)
	}
}

// Bounds returns the bounding box of the path's points (control points
// included), or ok false for an empty path.
func (p *Path) Bounds() (x0, y0, x1, y1 float64, ok bool) {
	x0, y0, x1, y1 = math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, s := range p.Segs {
		for _, q := range s.P[:s.Op.points()] {
			x0, y0 = math.Min(x0, q.X), math.Min(y0, q.Y)
			x1, y1 = math.Max(x1, q.X), math.Max(y1, q.Y)
		}
	}
	return x0, y0, x1, y1, x0 <= x1
}

// points returns how many of a Seg's points op uses.
func (op Op) points() int {
	switch op {
	case MoveTo, LineTo:
		return 1
	case QuadTo:
		return 2
	case CubicTo:
		return 3
	}
	return 0
}

// Affine is the transform x' = A*x + C*y + E, y' = B*x + D*y + F.
type Affine struct{ A, B, C, D, E, F float64 }

// Identity is the identity transform.
var Identity = Affine{A: 1, D: 1}

// Translate returns a translation.
func Translate(dx, dy float64) Affine { return Affine{A: 1, D: 1, E: dx, F: dy} }

// Scale returns a scaling about the origin.
func Scale(sx, sy float64) Affine { return Affine{A: sx, D: sy} }

// Rotate returns a rotation by deg degrees counter-clockwise on screen (y
// pointing down), the sense in which matplotlib rotates text.
func Rotate(deg float64) Affine {
	s, c := math.Sincos(deg * math.Pi / 180)
	return Affine{A: c, B: -s, C: s, D: c}
}

// Then returns the transform that applies m, then n.
func (m Affine) Then(n Affine) Affine {
	return Affine{
		A: n.A*m.A + n.C*m.B,
		B: n.B*m.A + n.D*m.B,
		C: n.A*m.C + n.C*m.D,
		D: n.B*m.C + n.D*m.D,
		E: n.A*m.E + n.C*m.F + n.E,
		F: n.B*m.E + n.D*m.F + n.F,
	}
}

// Apply transforms a point.
func (m Affine) Apply(p Pt) Pt {
	return Pt{m.A*p.X + m.C*p.Y + m.E, m.B*p.X + m.D*p.Y + m.F}
}
