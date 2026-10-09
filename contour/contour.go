// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package contour traces iso-lines and fills iso-bands of a function sampled
// on a rectangular grid, by marching squares with linear interpolation along
// cell edges.
//
// A grid is Z[i*nx+j] at (X[j], Y[i]); X and Y must be monotonic. Saddle
// cells (opposite corners on the same side of the level) are resolved by the
// value at the cell centre, taken as the mean of the four corners. NaN
// corners make a cell contribute nothing.
package contour

import "math"

// Pt is a point in grid (data) coordinates.
type Pt struct{ X, Y float64 }

// Grid is a function sampled on a rectangular grid.
type Grid struct {
	X, Y []float64 // len nx and ny
	Z    []float64 // ny*nx, row-major: Z[i*nx+j] is at (X[j], Y[i])
}

func (g Grid) nx() int { return len(g.X) }
func (g Grid) ny() int { return len(g.Y) }

func (g Grid) z(i, j int) float64 { return g.Z[i*g.nx()+j] }

// edge identifies a cell edge globally: horizontal edges (between (i,j) and
// (i,j+1)) and vertical edges (between (i,j) and (i+1,j)).
type edge struct {
	i, j int
	vert bool
}

// Lines returns the iso-lines at level as polylines. Lines that close on
// themselves repeat their first point at the end.
func Lines(g Grid, level float64) [][]Pt {
	nx, ny := g.nx(), g.ny()
	// Each cell yields up to two segments between edges; chain them.
	next := map[edge][]edge{}
	link := func(a, b edge) {
		next[a] = append(next[a], b)
		next[b] = append(next[b], a)
	}
	for i := 0; i < ny-1; i++ {
		for j := 0; j < nx-1; j++ {
			z00, z01 := g.z(i, j), g.z(i, j+1)
			z10, z11 := g.z(i+1, j), g.z(i+1, j+1)
			if math.IsNaN(z00) || math.IsNaN(z01) || math.IsNaN(z10) || math.IsNaN(z11) {
				continue
			}
			code := 0
			if z00 > level {
				code |= 1
			}
			if z01 > level {
				code |= 2
			}
			if z11 > level {
				code |= 4
			}
			if z10 > level {
				code |= 8
			}
			bottom := edge{i, j, false}   // (i,j)-(i,j+1)
			right := edge{i, j + 1, true} // (i,j+1)-(i+1,j+1)
			top := edge{i + 1, j, false}  // (i+1,j)-(i+1,j+1)
			left := edge{i, j, true}      // (i,j)-(i+1,j)
			switch code {
			case 0, 15:
			case 1, 14:
				link(left, bottom)
			case 2, 13:
				link(bottom, right)
			case 3, 12:
				link(left, right)
			case 4, 11:
				link(right, top)
			case 6, 9:
				link(bottom, top)
			case 7, 8:
				link(left, top)
			case 5, 10:
				centre := (z00 + z01 + z10 + z11) / 4
				// code 5: corners 00 and 11 above. If the centre is above,
				// the above corners connect through the middle.
				if (centre > level) == (code == 5) {
					link(left, top)
					link(bottom, right)
				} else {
					link(left, bottom)
					link(right, top)
				}
			}
		}
	}
	point := func(e edge) Pt {
		i, j := e.i, e.j
		i2, j2 := i, j+1
		if e.vert {
			i2, j2 = i+1, j
		}
		za, zb := g.z(i, j), g.z(i2, j2)
		t := (level - za) / (zb - za)
		return Pt{
			X: g.X[j] + t*(g.X[j2]-g.X[j]),
			Y: g.Y[i] + t*(g.Y[i2]-g.Y[i]),
		}
	}
	unlink := func(a, b edge) {
		for k, e := range next[a] {
			if e == b {
				next[a] = append(next[a][:k:k], next[a][k+1:]...)
				return
			}
		}
	}
	var out [][]Pt
	// Every edge has at most two links (one per cell it borders), so a walk
	// that removes what it crosses traces one line and stops at its end, or
	// back at its start for a loop.
	walk := func(start edge) {
		line := []Pt{point(start)}
		for cur := start; len(next[cur]) > 0; {
			nxt := next[cur][0]
			unlink(cur, nxt)
			unlink(nxt, cur)
			line = append(line, point(nxt))
			cur = nxt
		}
		out = append(out, line)
	}
	// Open lines start at an edge with one link (the grid boundary); what is
	// left after them is closed loops.
	for _, pass := range []int{1, 2} {
		for i := 0; i < ny; i++ {
			for j := 0; j < nx; j++ {
				for _, v := range []bool{false, true} {
					e := edge{i, j, v}
					n := len(next[e])
					if (pass == 1 && n == 1) || (pass == 2 && n == 2) {
						walk(e)
					}
				}
			}
		}
	}
	return out
}

// vertex is a polygon corner carrying the function value there.
type vertex struct{ x, y, z float64 }

// clip keeps the part of a convex-or-not polygon where keep(z) holds,
// interpolating z linearly along edges (Sutherland–Hodgman against the
// half-space z >= level, or z <= level when below is set).
func clip(poly []vertex, level float64, below bool) []vertex {
	in := func(v vertex) bool {
		if below {
			return v.z <= level
		}
		return v.z >= level
	}
	var out []vertex
	for k := range poly {
		a, b := poly[k], poly[(k+1)%len(poly)]
		ina, inb := in(a), in(b)
		if ina {
			out = append(out, a)
		}
		if ina != inb {
			t := (level - a.z) / (b.z - a.z)
			out = append(out, vertex{a.x + t*(b.x-a.x), a.y + t*(b.y-a.y), level})
		}
	}
	return out
}

// Band returns the region where lo <= z <= hi as polygons, one per grid
// cell that the band touches, all wound counter-clockwise in (x, y). Filled
// together under the nonzero rule they make the band without seams.
func Band(g Grid, lo, hi float64) [][]Pt {
	nx, ny := g.nx(), g.ny()
	var out [][]Pt
	for i := 0; i < ny-1; i++ {
		for j := 0; j < nx-1; j++ {
			quad := []vertex{
				{g.X[j], g.Y[i], g.z(i, j)},
				{g.X[j+1], g.Y[i], g.z(i, j+1)},
				{g.X[j+1], g.Y[i+1], g.z(i+1, j+1)},
				{g.X[j], g.Y[i+1], g.z(i+1, j)},
			}
			skip := false
			mn, mx := math.Inf(1), math.Inf(-1)
			for _, v := range quad {
				if math.IsNaN(v.z) {
					skip = true
				}
				mn, mx = math.Min(mn, v.z), math.Max(mx, v.z)
			}
			if skip || mx < lo || mn > hi {
				continue
			}
			p := quad
			if mn < lo {
				p = clip(p, lo, false)
			}
			if mx > hi && len(p) > 0 {
				p = clip(p, hi, true)
			}
			if len(p) < 3 {
				continue
			}
			pts := make([]Pt, len(p))
			for k, v := range p {
				pts[k] = Pt{v.x, v.y}
			}
			if area(pts) < 0 {
				for l, r := 0, len(pts)-1; l < r; l, r = l+1, r-1 {
					pts[l], pts[r] = pts[r], pts[l]
				}
			}
			out = append(out, pts)
		}
	}
	return out
}

// area is the signed area of a polygon (positive counter-clockwise).
func area(p []Pt) float64 {
	s := 0.0
	for k := range p {
		a, b := p[k], p[(k+1)%len(p)]
		s += a.X*b.Y - b.X*a.Y
	}
	return s / 2
}
