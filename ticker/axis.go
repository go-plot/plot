// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ticker

import "math"

// A Tick is one tick an axis draws: where, and the label it carries.
type Tick struct {
	Loc   float64
	Label string
}

// Ticks is what an axis draws for one view interval.
type Ticks struct {
	Major, Minor []Tick
	Offset       string // text at the end of the axis, or ""
}

// Config is the locators and formatters of one axis. Minor and MinorFormat
// may be nil (no minor ticks), as on a linear axis by default.
type Config struct {
	Major       Locator
	Minor       Locator
	MajorFormat Formatter
	MinorFormat Formatter
	Log         bool // the axis scale is logarithmic (base 10 transform)
}

// Compute returns the ticks of an axis whose view is [vmin, vmax], as
// matplotlib's Axis._update_ticks does: every major location is formatted (so
// that the format can depend on locations just outside the view), minor
// locations that coincide with major ones are dropped, and only the ticks
// within the view (to a relative 1e-10, in scale coordinates) are kept.
func (c Config) Compute(vmin, vmax float64) Ticks {
	tr := func(v float64) float64 { return v }
	if c.Log {
		tr = func(v float64) float64 {
			if v <= 0 {
				return math.Inf(-1)
			}
			return log10(v)
		}
	}
	lo, hi := tr(vmin), tr(vmax)
	if lo > hi {
		lo, hi = hi, lo
	}
	rtol := (hi - lo) * 1e-10
	inView := func(v float64) bool {
		t := tr(v)
		return lo-rtol <= t && t <= hi+rtol
	}

	var out Ticks
	major := c.Major.TickValues(vmin, vmax)
	labels := FormatTicks(c.MajorFormat, major, vmin, vmax)
	for i, l := range major {
		if inView(l) {
			out.Major = append(out.Major, Tick{Loc: l, Label: labels[i]})
		}
	}
	out.Offset = c.MajorFormat.Offset()
	if c.Minor == nil {
		return out
	}
	tol := (hi - lo) * 1e-5
	var minor []float64
	for _, m := range c.Minor.TickValues(vmin, vmax) {
		dup := false
		for _, l := range major {
			if math.Abs(tr(m)-tr(l)) <= tol {
				dup = true
				break
			}
		}
		if !dup {
			minor = append(minor, m)
		}
	}
	mlabels := make([]string, len(minor))
	if c.MinorFormat != nil {
		mlabels = FormatTicks(c.MinorFormat, minor, vmin, vmax)
	}
	for i, m := range minor {
		if inView(m) {
			out.Minor = append(out.Minor, Tick{Loc: m, Label: mlabels[i]})
		}
	}
	return out
}

// LinearConfig returns the configuration of a matplotlib linear axis on which
// tickSpace ticks fit: AutoLocator and ScalarFormatter, no minor ticks.
func LinearConfig(tickSpace int) Config {
	return Config{Major: NewAutoLocator(tickSpace), MajorFormat: NewScalarFormatter()}
}

// LogConfig returns the configuration of a matplotlib base-10 log axis on
// which tickSpace ticks fit; minPos is the smallest positive data value.
func LogConfig(tickSpace int, minPos float64) Config {
	return Config{
		Major:       &LogLocator{TickSpace: tickSpace, MinPos: minPos},
		Minor:       &LogLocator{SubsAuto: true, TickSpace: tickSpace, MinPos: minPos},
		MajorFormat: NewLogFormatterSciNotation(),
		MinorFormat: NewLogFormatterSciNotation(),
		Log:         true,
	}
}
