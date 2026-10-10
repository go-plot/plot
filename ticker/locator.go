// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package ticker chooses where the ticks of an axis go and how they are
// labelled. It is a port of the parts of matplotlib.ticker that a default
// matplotlib figure uses — AutoLocator (MaxNLocator), ScalarFormatter,
// LogLocator and LogFormatterSciNotation — and its results are tested to be
// equal to matplotlib's, tick for tick and label for label.
package ticker

import "math"

// A Locator returns tick locations for a view interval. The locations may
// extend beyond the interval; the axis keeps only those inside it.
type Locator interface {
	TickValues(vmin, vmax float64) []float64
}

// MaxNLocator finds up to Nbins+1 nicely rounded tick locations, nice meaning
// a multiple of one of Steps times a power of ten. It is matplotlib's
// MaxNLocator; NewAutoLocator returns it configured as matplotlib's
// AutoLocator, the default for linear axes.
type MaxNLocator struct {
	// Nbins is the maximum number of intervals. Zero means "auto": TickSpace,
	// clipped to [max(1, MinNTicks-1), 9], or 9 when TickSpace is zero.
	Nbins int
	// TickSpace is the number of ticks that fit on the axis, as the axis
	// estimates it (matplotlib's Axis.get_tick_space). Used when Nbins is zero.
	TickSpace int
	// Steps are the acceptable multiples of a power of ten, in [1, 10],
	// increasing. Nil means AutoLocator's {1, 2, 2.5, 5, 10}.
	Steps []float64
	// Integer restricts ticks to integer values.
	Integer bool
	// Symmetric makes the range symmetric about zero.
	Symmetric bool
	// Prune drops the "lower", "upper" or "both" extreme ticks.
	Prune string
	// MinNTicks is the minimum number of ticks in the interval; at least 1.
	MinNTicks int
}

// NewAutoLocator returns matplotlib's AutoLocator: nbins "auto", steps
// {1, 2, 2.5, 5, 10}, at least two ticks.
func NewAutoLocator(tickSpace int) *MaxNLocator {
	return &MaxNLocator{TickSpace: tickSpace, MinNTicks: 2}
}

// extendedSteps returns matplotlib's _staircase of the validated steps:
// 0.1*steps[:-1], steps, 10*steps[1].
func (l *MaxNLocator) extendedSteps() []float64 {
	steps := l.Steps
	if steps == nil {
		steps = []float64{1, 2, 2.5, 5, 10}
	}
	if steps[0] != 1 {
		steps = append([]float64{1}, steps...)
	}
	if steps[len(steps)-1] != 10 {
		steps = append(append([]float64(nil), steps...), 10)
	}
	out := make([]float64, 0, 2*len(steps))
	for _, s := range steps[:len(steps)-1] {
		out = append(out, 0.1*s)
	}
	out = append(out, steps...)
	return append(out, 10*steps[1])
}

func (l *MaxNLocator) minN() int { return max(1, l.MinNTicks) }

func (l *MaxNLocator) nbins() int {
	if l.Nbins > 0 {
		return l.Nbins
	}
	if l.TickSpace <= 0 {
		return 9
	}
	return min(max(l.TickSpace, max(1, l.minN()-1)), 9)
}

// scaleRange is matplotlib's scale_range.
func scaleRange(vmin, vmax float64, n int) (scale, offset float64) {
	dv := math.Abs(vmax - vmin)
	meanv := (vmax + vmin) / 2
	if math.Abs(meanv)/dv >= 100 {
		offset = math.Copysign(pow10(int(math.Floor(log10(math.Abs(meanv))))), meanv)
	}
	scale = pow10(int(math.Floor(log10(dv / float64(n)))))
	return scale, offset
}

// edgeInteger is matplotlib's _Edge_integer.
type edgeInteger struct{ step, offset float64 }

func (e edgeInteger) closeto(ms, edge float64) bool {
	tol := 1e-10
	if e.offset > 0 {
		digits := math.Log10(e.offset / e.step)
		tol = math.Min(0.4999, math.Max(1e-10, math.Pow(10, digits-12)))
	}
	return math.Abs(ms-edge) < tol
}

func (e edgeInteger) le(x float64) float64 {
	d, m := pyDivmod(x, e.step)
	if e.closeto(m/e.step, 1) {
		return d + 1
	}
	return d
}

func (e edgeInteger) ge(x float64) float64 {
	d, m := pyDivmod(x, e.step)
	if e.closeto(m/e.step, 0) {
		return d
	}
	return d + 1
}

// rawTicks is matplotlib's MaxNLocator._raw_ticks.
func (l *MaxNLocator) rawTicks(vmin, vmax float64) []float64 {
	nbins := l.nbins()
	scale, offset := scaleRange(vmin, vmax, nbins)
	vmin0 := vmin - offset
	vmax0 := vmax - offset
	var steps []float64
	for _, s := range l.extendedSteps() {
		s = float64(s * scale)
		if l.Integer && !(s < 1 || math.Abs(s-math.RoundToEven(s)) < 0.001) {
			continue
		}
		steps = append(steps, s)
	}
	rawStep := (vmax0 - vmin0) / float64(nbins)
	istep := len(steps) - 1
	for i, s := range steps {
		if s >= rawStep {
			istep = i
			break
		}
	}
	var ticks []float64
	for i := istep; i >= 0; i-- {
		step := steps[i]
		if l.Integer && math.Floor(vmax0)-math.Ceil(vmin0) >= float64(l.minN()-1) {
			step = math.Max(1, step)
		}
		bestVmin := float64(floorDiv(vmin0, step) * step)
		edge := edgeInteger{step: step, offset: math.Abs(offset)}
		low := edge.le(vmin0 - bestVmin)
		high := edge.ge(vmax0 - bestVmin)
		ticks = ticks[:0]
		n := 0
		for k := low; k < high+1; k++ {
			t := float64(k*step) + bestVmin
			ticks = append(ticks, t)
			if t <= vmax0 && t >= vmin0 {
				n++
			}
		}
		if n >= l.minN() {
			break
		}
	}
	out := make([]float64, len(ticks))
	for i, t := range ticks {
		out[i] = t + offset
	}
	return out
}

// TickValues returns the tick locations for [vmin, vmax].
func (l *MaxNLocator) TickValues(vmin, vmax float64) []float64 {
	if l.Symmetric {
		vmax = math.Max(math.Abs(vmin), math.Abs(vmax))
		vmin = -vmax
	}
	vmin, vmax = Nonsingular(vmin, vmax, 1e-13, 1e-14, true)
	locs := l.rawTicks(vmin, vmax)
	switch l.Prune {
	case "lower":
		locs = locs[1:]
	case "upper":
		locs = locs[:len(locs)-1]
	case "both":
		locs = locs[1 : len(locs)-1]
	}
	return locs
}

// LogLocator places ticks at Subs[j] * Base**i. It is matplotlib's
// LogLocator; with Subs nil it places the major ticks of a log axis, with
// SubsAuto the minor ones.
type LogLocator struct {
	Base float64 // zero means 10
	// Subs are the multiples of the powers of Base to tick at. Nil means {1}.
	Subs []float64
	// SubsAuto is matplotlib's subs="auto": ticks between the powers of Base
	// (2..Base-1) when few decades are shown, none otherwise.
	SubsAuto bool
	// SubsAll is matplotlib's subs="all": like SubsAuto, but the powers of
	// Base themselves are included, and when too many decades are shown it
	// falls back to the powers alone.
	SubsAll bool
	// NumTicks is the maximum number of ticks; zero means "auto" (TickSpace
	// clipped to [2, 9], or 9 when TickSpace is zero).
	NumTicks  int
	TickSpace int
	// MinPos is the smallest positive data value, used when vmin <= 0.
	MinPos float64
}

func (l *LogLocator) base() float64 {
	if l.Base == 0 {
		return 10
	}
	return l.Base
}

func (l *LogLocator) logB(x float64) float64 {
	switch b := l.base(); b {
	case 10:
		return log10(x)
	case 2:
		return math.Log2(x)
	default:
		return math.Log(x) / math.Log(b)
	}
}

// TickValues returns the tick locations for [vmin, vmax]. A vmin <= 0 is
// replaced by MinPos; with no positive value at all it returns nil.
func (l *LogLocator) TickValues(vmin, vmax float64) []float64 {
	nRequest := l.NumTicks
	if nRequest == 0 {
		nRequest = 9
		if l.TickSpace > 0 {
			nRequest = min(max(l.TickSpace, 2), 9)
		}
	}
	b := l.base()
	if vmin <= 0 {
		vmin = l.MinPos
		if vmin <= 0 || math.IsInf(vmin, 0) || math.IsNaN(vmin) {
			return nil
		}
	}
	if vmax < vmin {
		vmin, vmax = vmax, vmin
	}
	efmin, efmax := l.logB(vmin), l.logB(vmax)
	emin := int(math.Ceil(efmin))
	emax := int(math.Floor(efmax))
	nAvail := emax - emin + 1
	subs := l.Subs
	if l.SubsAuto || l.SubsAll {
		subs = []float64{1}
		if nAvail < 10 && b >= 3 {
			first := 2.0
			if l.SubsAll {
				first = 1
			}
			subs = nil
			for s := first; s < b; s++ {
				subs = append(subs, s)
			}
		} else if l.SubsAuto {
			return nil
		}
	} else if subs == nil {
		subs = []float64{1}
	}

	stride := nAvail/(nRequest+1) + 1
	if nr := ceilDiv(nAvail, stride); nr <= nRequest {
		nRequest = nr
	}
	var decades []int
	switch nRequest {
	case 0:
		decades = []int{emin - 1, emax + 1}
		stride = decades[1] - decades[0]
	case 1:
		mid := int(math.RoundToEven((efmin + efmax) / 2))
		stride = max(mid-(emin-1), (emax+1)-mid)
		decades = []int{mid - stride, mid, mid + stride}
	default:
		stride = (nAvail - 1) / (nRequest - 1)
		if float64(stride) < float64(nAvail)/float64(nRequest) {
			stride = nAvail / nRequest
		}
		olo := max(nAvail-stride*nRequest, 0)
		ohi := min(nAvail-stride*(nRequest-1), stride)
		offset := pyMod(-emin, stride)
		if !(olo <= offset && offset < ohi) {
			offset = olo
		}
		for d := emin + offset - stride; d < emax+stride+1; d += stride {
			decades = append(decades, d)
		}
	}
	isMinor := len(subs) > 1 || subs[0] != 1
	var locs []float64
	if isMinor {
		if stride == 1 || nAvail <= 1 {
			for d := emin - 1; d < emax+1; d++ {
				p := powB(b, d)
				for _, s := range subs {
					locs = append(locs, s*p)
				}
			}
		}
	} else {
		for _, d := range decades {
			locs = append(locs, powB(b, d))
		}
	}
	if len(subs) > 1 && stride == 1 {
		n := len(decades) - 2
		for _, t := range locs {
			if vmin <= t && t <= vmax {
				n++
			}
		}
		if n <= 1 {
			// A minor locator that expects at least two ticks per decade
			// falls back to a fresh AutoLocator — one with no axis, so nine
			// bins, exactly as matplotlib does.
			return NewAutoLocator(0).TickValues(vmin, vmax)
		}
	}
	return locs
}

// powB is b**d as Python computes it: correctly rounded for base 10, which
// math.Pow is not.
func powB(b float64, d int) float64 {
	if b == 10 {
		return pow10(d)
	}
	return math.Pow(b, float64(d))
}

// ceilDiv is ceil(a/b) for positive b.
func ceilDiv(a, b int) int { return int(math.Ceil(float64(a) / float64(b))) }

// pyMod is Python's a % b for ints: the result takes the sign of b.
func pyMod(a, b int) int {
	m := a % b
	if m != 0 && (m < 0) != (b < 0) {
		m += b
	}
	return m
}
