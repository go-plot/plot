// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ticker

import (
	_ "embed"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

// units.json is written by tools/gen_ticker_units.py: matplotlib's locators
// and formatters called directly with non-default parameters.
//
//go:embed testdata/units.json
var unitsJSON []byte

type units struct {
	Maxn []struct {
		Params struct {
			Nbins     int       `json:"nbins"`
			Steps     []float64 `json:"steps"`
			Integer   bool      `json:"integer"`
			Symmetric bool      `json:"symmetric"`
			Prune     string    `json:"prune"`
			MinN      int       `json:"min_n_ticks"`
		} `json:"params"`
		Vmin, Vmax float64
		Locs       []float64
	} `json:"maxn"`
	Log []struct {
		Params struct {
			Base     any `json:"base"`
			Subs     any `json:"subs"`
			NumTicks int `json:"numticks"`
		} `json:"params"`
		Vmin, Vmax float64
		Locs       []float64
	} `json:"log"`
	Scalar []struct {
		Params struct {
			UseMathText bool  `json:"useMathText"`
			UseOffset   *bool `json:"useOffset"`
			PowerLimits []int `json:"powerlimits"`
			Scientific  *bool `json:"scientific"`
			Unicode     *bool `json:"unicode"`
		} `json:"params"`
		Locs   []float64
		View   []float64
		Labels []string
		Offset string
	} `json:"scalar"`
	Logfmt []struct {
		Params struct {
			LabelOnlyBase   bool  `json:"labelOnlyBase"`
			Base            int   `json:"base"`
			MinorThresholds []any `json:"minor_thresholds"`
		} `json:"params"`
		View   []float64
		Vals   []float64
		Labels []string
	} `json:"logfmt"`
	Nonsingular []struct {
		Args []any
		Out  []float64
	} `json:"nonsingular"`
	Fmtg []struct {
		V    float64
		Prec int
		S    string
	} `json:"fmtg"`
}

func num(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		if x == "e" {
			return math.E
		}
		return math.Inf(1)
	}
	return 0
}

func TestUnitsMatchMatplotlib(t *testing.T) {
	var u units
	if err := json.Unmarshal(unitsJSON, &u); err != nil {
		t.Fatal(err)
	}
	for _, c := range u.Maxn {
		p := c.Params
		steps := p.Steps
		if steps == nil {
			// MaxNLocator's own default; nil here means AutoLocator's.
			steps = []float64{1, 1.5, 2, 2.5, 3, 4, 5, 6, 8, 10}
		}
		l := &MaxNLocator{Nbins: p.Nbins, Steps: steps, Integer: p.Integer, Symmetric: p.Symmetric, Prune: p.Prune, MinNTicks: max(2, p.MinN)}
		if got := l.TickValues(c.Vmin, c.Vmax); !slices.Equal(got, c.Locs) {
			t.Errorf("MaxN %+v [%v, %v]: %v, want %v", p, c.Vmin, c.Vmax, got, c.Locs)
		}
	}
	for _, c := range u.Log {
		p := c.Params
		l := &LogLocator{Base: num(p.Base), NumTicks: p.NumTicks}
		switch s := p.Subs.(type) {
		case string:
			l.SubsAuto, l.SubsAll = s == "auto", s == "all"
		case []any:
			for _, v := range s {
				l.Subs = append(l.Subs, num(v))
			}
		}
		got := l.TickValues(c.Vmin, c.Vmax)
		if l.Base == math.E {
			// NumPy's e**n and Go's math.Pow(e, n) can differ in the last
			// bit; base e is compared to 1e-15.
			if len(got) == len(c.Locs) {
				for i := range got {
					if math.Abs(got[i]-c.Locs[i]) <= 1e-15*c.Locs[i] {
						got[i] = c.Locs[i]
					}
				}
			}
		}
		if !slices.Equal(got, c.Locs) && !(len(got) == 0 && len(c.Locs) == 0) {
			t.Errorf("Log %+v [%v, %v]: %v, want %v", p, c.Vmin, c.Vmax, got, c.Locs)
		}
	}
	for _, c := range u.Scalar {
		p := c.Params
		f := NewScalarFormatter()
		f.UseMathText = p.UseMathText
		if p.UseOffset != nil {
			f.UseOffset = *p.UseOffset
		}
		if p.PowerLimits != nil {
			f.PowerLimits = [2]int{p.PowerLimits[0], p.PowerLimits[1]}
		}
		if p.Scientific != nil {
			f.Scientific = *p.Scientific
		}
		if p.Unicode != nil {
			f.UnicodeMinus = *p.Unicode
		}
		got := FormatTicks(f, c.Locs, c.View[0], c.View[1])
		if !slices.Equal(got, c.Labels) && !(len(got) == 0 && len(c.Labels) == 0) || f.Offset() != c.Offset {
			t.Errorf("Scalar %+v %v %v: %q %q, want %q %q", p, c.Locs, c.View, got, f.Offset(), c.Labels, c.Offset)
		}
	}
	for _, c := range u.Logfmt {
		p := c.Params
		f := NewLogFormatterSciNotation()
		f.LabelOnlyBase = p.LabelOnlyBase
		f.Base = float64(p.Base)
		if p.MinorThresholds != nil {
			f.MinorThresholds = [2]float64{num(p.MinorThresholds[0]), num(p.MinorThresholds[1])}
		}
		got := FormatTicks(f, c.Vals, c.View[0], c.View[1])
		if !slices.Equal(got, c.Labels) || f.Offset() != "" {
			t.Errorf("LogFmt %+v %v %v: %q, want %q", p, c.Vals, c.View, got, c.Labels)
		}
	}
	for _, c := range u.Nonsingular {
		a, b := Nonsingular(num(c.Args[0]), num(c.Args[1]), num(c.Args[2]), num(c.Args[3]), c.Args[4].(bool))
		if a != c.Out[0] || b != c.Out[1] {
			t.Errorf("Nonsingular %v: %v %v, want %v", c.Args, a, b, c.Out)
		}
	}
	for _, c := range u.Fmtg {
		if got := fmtG(c.V, c.Prec); got != c.S {
			t.Errorf("fmtG(%v, %d) = %q, want %q", c.V, c.Prec, got, c.S)
		}
	}
}

// TestComputeExtras covers what the fixtures cannot reach through a real
// axis: a reversed view, minor ticks that coincide with major ones, a minor
// locator without formatter, and log locations at or below zero.
func TestComputeExtras(t *testing.T) {
	cfg := Config{
		Major:       NewAutoLocator(0),
		Minor:       &MaxNLocator{Nbins: 20, MinNTicks: 2},
		MajorFormat: NewScalarFormatter(),
	}
	tk := cfg.Compute(1, 0)
	if len(tk.Major) != 6 || len(tk.Minor) != 15 {
		t.Errorf("reversed view: %d major, %d minor", len(tk.Major), len(tk.Minor))
	}
	lc := LogConfig(9, 0.5)
	lc.Major = fixed{0, 1, 10}
	if tk := lc.Compute(1, 10); len(tk.Major) != 2 {
		t.Errorf("log with zero location: %v", tk.Major)
	}
	if got := (&LogLocator{}).TickValues(-1, 10); got != nil {
		t.Errorf("no positive MinPos: %v", got)
	}
	if got := (&LogLocator{MinPos: 1}).TickValues(-1, 100); len(got) == 0 {
		t.Error("MinPos not used")
	}
	steps := (&MaxNLocator{Steps: []float64{2, 5}}).extendedSteps()
	if steps[0] != 0.1 || steps[len(steps)-1] != 20 {
		t.Errorf("extended steps %v", steps)
	}
	if NewScalarFormatter().Format(1) != "" {
		t.Error("format before SetLocs")
	}
	if powB(2, 3) != 8 || pyMod(-7, 3) != 2 || ceilDiv(7, 2) != 4 {
		t.Error("helpers")
	}
	if trimZeros("100") != "100" || FixMinus("-1") != "−1" {
		t.Error("strings")
	}
	if d, m := pyDivmod(-7, 2); d != -4 || m != 1 {
		t.Errorf("divmod %v %v", d, m)
	}
	if d, m := pyDivmod(6, -3); d != -2 || m != 0 || !math.Signbit(m) {
		t.Errorf("divmod %v %v", d, m)
	}
	if d, _ := pyDivmod(0.5, 3); d != 0 {
		t.Errorf("divmod small %v", d)
	}
	if pyRound(2.675, 2) != 2.67 || npRound(0.125, 2) != 0.12 {
		t.Error("rounding")
	}
}

type fixed []float64

func (f fixed) TickValues(_, _ float64) []float64 { return f }
