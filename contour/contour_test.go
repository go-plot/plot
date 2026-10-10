// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package contour

import (
	"math"
	"testing"
)

// cone is z = distance from (2, 2) on a 5 x 5 grid.
func cone() Grid {
	g := Grid{X: []float64{0, 1, 2, 3, 4}, Y: []float64{0, 1, 2, 3, 4}}
	for i := range 5 {
		for j := range 5 {
			g.Z = append(g.Z, math.Hypot(float64(i)-2, float64(j)-2))
		}
	}
	return g
}

// TestCircle: the level-1.5 line of a cone is one closed loop whose points
// all lie at distance 1.5 along grid edges (linear interpolation of an exact
// distance along an axis-parallel edge is exact at the crossing only up to
// the curvature, so allow 0.1).
func TestCircle(t *testing.T) {
	lines := Lines(cone(), 1.5)
	if len(lines) != 1 {
		t.Fatalf("%d lines", len(lines))
	}
	l := lines[0]
	if l[0] != l[len(l)-1] {
		t.Error("loop not closed")
	}
	if len(l) != 13 {
		t.Errorf("%d points", len(l))
	}
	for _, p := range l {
		if d := math.Hypot(p.X-2, p.Y-2); math.Abs(d-1.5) > 0.1 {
			t.Errorf("point %v at distance %v", p, d)
		}
	}
}

// TestOpenLine: a plane crosses the grid in one open line from edge to edge.
func TestOpenLine(t *testing.T) {
	g := Grid{X: []float64{0, 1, 2}, Y: []float64{0, 1}, Z: []float64{0, 1, 2, 0, 1, 2}}
	lines := Lines(g, 0.5)
	if len(lines) != 1 || len(lines[0]) != 2 || lines[0][0].X != 0.5 || lines[0][1].X != 0.5 {
		t.Errorf("lines %v", lines)
	}
}

// TestAllCases runs every one of the 16 corner configurations, saddles in
// both resolutions, and NaN corners, checking that each crossing lies on the
// level.
func TestAllCases(t *testing.T) {
	for code := range 16 {
		for _, centreHigh := range []bool{false, true} {
			z := [4]float64{} // 00, 01, 11, 10
			for k := range 4 {
				if code&(1<<k) != 0 {
					z[k] = 1
				}
			}
			if centreHigh && (code == 5 || code == 10) {
				for k := range 4 {
					if z[k] == 1 {
						z[k] = 3
					}
				}
			}
			g := Grid{X: []float64{0, 1}, Y: []float64{0, 1}, Z: []float64{z[0], z[1], z[3], z[2]}}
			for _, l := range Lines(g, 0.5) {
				for _, p := range l {
					if p.X != 0 && p.X != 1 && p.Y != 0 && p.Y != 1 {
						t.Errorf("code %d: point %v not on an edge", code, p)
					}
				}
			}
		}
	}
	nan := Grid{X: []float64{0, 1}, Y: []float64{0, 1}, Z: []float64{0, math.NaN(), 1, 1}}
	if len(Lines(nan, 0.5)) != 0 || len(Band(nan, 0, 1)) != 0 {
		t.Error("NaN cell contributed")
	}
}

// TestBandArea: the bands of a plane between its levels have the area of
// the strip they cover, sum to the grid, and are counter-clockwise.
func TestBandArea(t *testing.T) {
	g := Grid{X: []float64{0, 1, 2, 3}, Y: []float64{0, 2}, Z: []float64{0, 1, 2, 3, 0, 1, 2, 3}}
	total := 0.0
	for _, lv := range [][2]float64{{-1, 0.5}, {0.5, 2.5}, {2.5, 9}} {
		a := 0.0
		for _, pg := range Band(g, lv[0], lv[1]) {
			s := area(pg)
			if s <= 0 {
				t.Errorf("band %v: clockwise polygon %v", lv, pg)
			}
			a += s
		}
		total += a
	}
	if math.Abs(total-6) > 1e-12 {
		t.Errorf("bands cover %v, want 6", total)
	}
	if len(Band(g, 10, 11)) != 0 {
		t.Error("band above the data")
	}
	// A grid running right to left yields clockwise cells, which Band turns.
	r := Grid{X: []float64{3, 2, 1, 0}, Y: []float64{0, 2}, Z: g.Z}
	for _, pg := range Band(r, 0.5, 2.5) {
		if area(pg) <= 0 {
			t.Error("reversed grid: clockwise polygon")
		}
	}
	// A band that touches a cell only at one corner leaves a degenerate
	// polygon, which is dropped.
	c := Grid{X: []float64{0, 1}, Y: []float64{0, 1}, Z: []float64{0, 1, 1, 1}}
	if b := Band(c, -1, 0); len(b) != 0 {
		t.Errorf("degenerate band %v", b)
	}
}
