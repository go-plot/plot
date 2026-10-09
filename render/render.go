// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package render

import (
	"image"
	"math"

	"github.com/go-plot/plot/colors"
)

// Cap is how an open stroke ends.
type Cap uint8

// Caps, as matplotlib names them.
const (
	CapButt       Cap = iota // stops at the end point
	CapRound                 // a half disc
	CapProjecting            // carries on half a width
)

// Join is how two segments of a stroke meet.
type Join uint8

// Joins, as matplotlib names them.
const (
	JoinMiter Join = iota
	JoinRound
	JoinBevel
)

// Style says how a path is painted. A zero Fill.A paints no fill; a zero
// Width or Stroke.A paints no stroke. The fill is painted first.
type Style struct {
	Fill    colors.RGBA
	EvenOdd bool // fill rule; nonzero otherwise

	Stroke colors.RGBA
	Width  float64   // pixels
	Cap    Cap       // ignored for closed subpaths
	Join   Join      //
	Dash   []float64 // on/off lengths in pixels; nil is solid
	Offset float64   // dash offset in pixels

	// Snap rounds the vertices of a path made only of horizontal and vertical
	// segments to the pixel grid so that its strokes are crisp, as
	// matplotlib's path snapping does for spines, ticks and grid lines.
	Snap bool
}

// Image is a raster placed in the figure: Pix is drawn into the rectangle
// (X, Y, W, H) in pixels, with nearest-neighbour sampling.
type Image struct {
	X, Y, W, H float64
	Pix        *image.RGBA
}

// A Renderer draws a figure. Coordinates are pixels, y down, origin at the
// top-left corner of the figure.
type Renderer interface {
	// Size is the figure size in pixels.
	Size() (w, h int)
	// DrawPath paints a path.
	DrawPath(p *Path, s Style)
	// DrawImage paints a raster.
	DrawImage(im Image)
	// PushClip restricts drawing to a rectangle (intersected with any
	// enclosing clip) until the matching PopClip.
	PushClip(x, y, w, h float64)
	PopClip()
	// Group brackets a logical object (an artist) for formats that keep
	// structure; id is a stable identifier such as "line2d_3". Raster
	// formats ignore it.
	BeginGroup(id string)
	EndGroup()
}

// snapPath returns p with every vertex snapped to the pixel grid when every
// segment is horizontal or vertical: to pixel centres for an odd stroke
// width (rounded to whole pixels) and to pixel edges for an even one, which
// is what matplotlib's PathSnapper does. A path that is only filled snaps with
// width 0, to pixel edges. Paths with more than 1024 vertices, curves, or a
// slanted segment are returned unchanged.
func snapPath(p *Path, width float64) *Path {
	if len(p.Segs) > 1024 {
		return p
	}
	var px, py float64
	for _, s := range p.Segs {
		switch s.Op {
		case MoveTo:
			px, py = s.P[0].X, s.P[0].Y
		case LineTo:
			if s.P[0].X != px && s.P[0].Y != py {
				return p
			}
			px, py = s.P[0].X, s.P[0].Y
		case QuadTo, CubicTo:
			return p
		}
	}
	off := 0.0
	if int(math.Floor(width+0.5))%2 == 1 {
		off = 0.5
	}
	q := &Path{Segs: make([]Seg, len(p.Segs))}
	for i, s := range p.Segs {
		if s.Op != Close {
			s.P[0] = Pt{snap(s.P[0].X, off), snap(s.P[0].Y, off)}
		}
		q.Segs[i] = s
	}
	return q
}

// snap is matplotlib's floor(v + 0.5) + off.
func snap(v, off float64) float64 { return math.Floor(v+0.5) + off }
