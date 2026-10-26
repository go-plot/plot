// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"math"
	"strings"

	"github.com/go-plot/plot/colors"
	"github.com/go-plot/plot/render"
	"github.com/go-plot/plot/text"
)

// box is a rectangle in pixels, y down.
type box struct{ x0, y0, x1, y1 float64 }

// textSpec is a string to place: anchored at (X, Y) pixels, size in pixels,
// aligned and rotated as a matplotlib Text with rotation_mode "default"
// (Anchor false) or "anchor".
type textSpec struct {
	S      string
	X, Y   float64
	Size   float64
	HA, VA string
	Rot    float64
	Anchor bool
	Color  colors.RGBA
	Plain  bool // no TeX: set through text.PlainRenderer when the renderer is one
}

// placedText is a laid-out string: its glyph outlines in figure pixels and
// the box matplotlib reports as its window extent.
type placedText struct {
	path *render.Path
	bbox box
}

// rotatePt rotates (x, y) by deg counter-clockwise, y up, as matplotlib's
// _rotate_point does.
func rotatePt(deg, x, y float64) (float64, float64) {
	s, c := math.Sincos(deg * math.Pi / 180)
	return c*x - s*y, s*x + c*y
}

// layoutText is matplotlib's Text._get_layout followed by drawing: the
// lines are stacked with the font's line metrics, the block aligned on the
// anchor, and each line's glyphs placed at its baseline.
func (f *Figure) layoutText(t textSpec) placedText {
	lines := strings.Split(t.S, "\n")
	minA, minD, gap := f.Text.LineMetrics(t.Size)
	if len(lines) == 1 {
		gap = 0
	}
	type line struct {
		w, a, d, y float64
		path       *render.Path
	}
	ls := make([]line, len(lines))
	thisy := 0.0
	width := 0.0
	for i, s := range lines {
		var l line
		if s != "" {
			var lay text.Layout
			if pr, ok := f.Text.(text.PlainRenderer); ok && t.Plain {
				lay = pr.LayoutPlain(s, t.Size)
			} else {
				lay = f.Text.Layout(s, t.Size)
			}
			l = line{w: lay.Width, a: lay.Ascent, d: lay.Descent, path: lay.Path}
		}
		l.a = math.Max(l.a, minA) + gap/2
		l.d = math.Max(l.d, minD) + gap/2
		thisy -= l.a
		l.y = thisy
		thisy -= l.d
		width = math.Max(width, l.w)
		ls[i] = l
	}
	baseline := ls[0].a
	descent := ls[len(ls)-1].d
	ymin := ls[len(ls)-1].y - descent
	// corners of the unrotated box, y up
	cx := [4]float64{0, 0, width, width}
	cy := [4]float64{ymin, 0, 0, ymin}
	var rx, ry [4]float64
	xmin, xmax, rymin, rymax := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for i := range cx {
		rx[i], ry[i] = rotatePt(t.Rot, cx[i], cy[i])
		xmin, xmax = math.Min(xmin, rx[i]), math.Max(xmax, rx[i])
		rymin, rymax = math.Min(rymin, ry[i]), math.Max(rymax, ry[i])
	}
	var ox, oy float64
	if !t.Anchor {
		switch t.HA {
		case "left":
			ox = xmin
		case "right":
			ox = xmax
		default:
			ox = (xmin + xmax) / 2
		}
		switch t.VA {
		case "bottom":
			oy = rymin
		case "top":
			oy = rymax
		case "center":
			oy = (rymin + rymax) / 2
		case "baseline":
			oy = rymin + descent
		default: // center_baseline
			oy = rymin + (rymax - rymin) - baseline/2
		}
	} else {
		switch t.HA {
		case "left":
			ox = 0
		case "right":
			ox = width
		default:
			ox = width / 2
		}
		switch t.VA {
		case "bottom":
			oy = ymin
		case "top":
			oy = 0
		case "center":
			oy = ymin / 2
		case "baseline":
			oy = -baseline
		default:
			oy = -baseline / 2
		}
		ox, oy = rotatePt(t.Rot, ox, oy)
	}
	out := placedText{
		path: &render.Path{},
		bbox: box{t.X + xmin - ox, t.Y - (rymax - oy), t.X + xmax - ox, t.Y - (rymin - oy)},
	}
	rot := render.Rotate(t.Rot)
	for _, l := range ls {
		if l.path == nil {
			continue
		}
		var x0 float64
		switch t.HA {
		case "left":
		case "right":
			x0 = width - l.w
		default:
			x0 = width/2 - l.w/2
		}
		lx, ly := rotatePt(t.Rot, x0, l.y)
		px, py := t.X+lx-ox, t.Y-(ly-oy)
		// Agg draws text at whole pixels; doing the same keeps upright text
		// as crisp as matplotlib's and costs a vector output nothing visible.
		px, py = math.Round(px), math.Round(py)
		out.path.Append(l.path, rot.Then(render.Translate(px, py)))
	}
	return out
}

// drawText lays out and paints t, returning its window extent.
func (f *Figure) drawText(r render.Renderer, t textSpec) box {
	p := f.layoutText(t)
	if f.collect != nil {
		*f.collect = append(*f.collect, p.bbox)
	}
	r.DrawPath(p.path, render.Style{Fill: t.Color})
	return p.bbox
}
