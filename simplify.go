// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import "github.com/go-plot/plot/render"

// simplifyThreshold is matplotlib's path.simplify_threshold, in pixels.
const simplifyThreshold = 1.0 / 9.0

// simplify is matplotlib's PathSimplifier, applied to a polyline in pixels:
// runs of segments that stay within simplifyThreshold of one direction are
// merged into the longest excursion forward and the longest backward, which
// leaves the drawn line indistinguishable and turns a million points across
// a few hundred pixels into a few thousand. Like matplotlib it is applied to
// lines of 128 points or more.
func simplify(pts []render.Pt) []render.Pt {
	if len(pts) < 128 {
		return pts
	}
	thr := simplifyThreshold * simplifyThreshold
	out := make([]render.Pt, 0, 256)
	out = append(out, pts[0])
	last := pts[0]
	var (
		origdx, origdy, origNorm2 float64
		fwdMax, bwdMax            float64
		lastFwdMax, lastBwdMax    bool
		start, next, nextBackward render.Pt
	)
	push := func(p render.Pt) {
		if bwdMax > 0 {
			if lastFwdMax {
				out = append(out, nextBackward, next)
			} else {
				out = append(out, next, nextBackward)
			}
		} else {
			out = append(out, next)
		}
		if !lastFwdMax && !lastBwdMax {
			out = append(out, last)
		}
		origdx, origdy = p.X-last.X, p.Y-last.Y
		origNorm2 = origdx*origdx + origdy*origdy
		fwdMax, lastFwdMax = origNorm2, true
		start = out[len(out)-1]
		last, next = p, p
		bwdMax, lastBwdMax = 0, false
	}
	for _, p := range pts[1:] {
		if origNorm2 == 0 {
			// No direction yet (the start, or a repeated point): this
			// segment sets it.
			origdx, origdy = p.X-last.X, p.Y-last.Y
			origNorm2 = origdx*origdx + origdy*origdy
			fwdMax, bwdMax = origNorm2, 0
			lastFwdMax, lastBwdMax = true, false
			start = last
			next, last = p, p
			continue
		}
		totdx, totdy := p.X-start.X, p.Y-start.Y
		totdot := origdx*totdx + origdy*totdy
		paradx, parady := totdot*origdx/origNorm2, totdot*origdy/origNorm2
		perpdx, perpdy := totdx-paradx, totdy-parady
		if perpdx*perpdx+perpdy*perpdy < thr {
			paraNorm2 := paradx*paradx + parady*parady
			lastFwdMax, lastBwdMax = false, false
			if totdot > 0 {
				if paraNorm2 > fwdMax {
					lastFwdMax, fwdMax, next = true, paraNorm2, p
				}
			} else if paraNorm2 > bwdMax {
				lastBwdMax, bwdMax, nextBackward = true, paraNorm2, p
			}
			last = p
			continue
		}
		push(p)
	}
	if origNorm2 != 0 {
		if bwdMax > 0 {
			if lastFwdMax {
				out = append(out, nextBackward, next)
			} else {
				out = append(out, next, nextBackward)
			}
		} else {
			out = append(out, next)
		}
		if !lastFwdMax && !lastBwdMax {
			out = append(out, last)
		}
	}
	return out
}
