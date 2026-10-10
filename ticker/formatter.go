// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ticker

import (
	"math"
	"strconv"
)

// A Formatter turns tick locations into label strings. SetLocs is called once
// with every location the locator returned (including those outside the
// view) and the view interval; Format is then called per tick, and Offset
// returns the text shown at the end of the axis (an offset or a power of ten
// common to every label), or "".
//
// Labels may contain TeX math between dollar signs, as matplotlib's do.
type Formatter interface {
	SetLocs(locs []float64, vmin, vmax float64)
	Format(x float64) string
	Offset() string
}

// FormatTicks is matplotlib's Formatter.format_ticks: SetLocs, then Format
// each location.
func FormatTicks(f Formatter, locs []float64, vmin, vmax float64) []string {
	f.SetLocs(locs, vmin, vmax)
	out := make([]string, len(locs))
	for i, x := range locs {
		out[i] = f.Format(x)
	}
	return out
}

// ScalarFormatter is matplotlib's default formatter for linear axes: plain
// decimals with just enough digits to tell the ticks apart, an offset when
// every tick shares a long common prefix, and scientific notation (a common
// power of ten shown at the end of the axis) outside PowerLimits.
type ScalarFormatter struct {
	UseOffset       bool
	OffsetThreshold int    // digits an offset must save before it is used
	PowerLimits     [2]int // scientific notation for exponents <= [0] or >= [1]
	Scientific      bool
	UseMathText     bool // write the offset as a TeX ×10^n
	UnicodeMinus    bool // write U+2212 rather than a hyphen

	locs       []float64
	vmin, vmax float64
	offset     float64
	oom        int
	prec       int
}

// NewScalarFormatter returns a ScalarFormatter with matplotlib's defaults.
func NewScalarFormatter() *ScalarFormatter {
	return &ScalarFormatter{
		UseOffset:       true,
		OffsetThreshold: 4,
		PowerLimits:     [2]int{-5, 6},
		Scientific:      true,
		UnicodeMinus:    true,
	}
}

func (f *ScalarFormatter) minus(s string) string {
	if f.UnicodeMinus {
		return FixMinus(s)
	}
	return s
}

// SetLocs implements Formatter.
func (f *ScalarFormatter) SetLocs(locs []float64, vmin, vmax float64) {
	if vmin > vmax {
		vmin, vmax = vmax, vmin
	}
	f.locs, f.vmin, f.vmax = locs, vmin, vmax
	f.offset, f.oom = 0, 0
	if len(locs) == 0 {
		return
	}
	if f.UseOffset {
		f.computeOffset()
	}
	f.setOrderOfMagnitude()
	f.setFormat()
}

// visible returns the locations inside the view interval.
func (f *ScalarFormatter) visible() []float64 {
	var out []float64
	for _, l := range f.locs {
		if f.vmin <= l && l <= f.vmax {
			out = append(out, l)
		}
	}
	return out
}

func (f *ScalarFormatter) computeOffset() {
	locs := f.visible()
	if len(locs) == 0 {
		return
	}
	lmin, lmax := locs[0], locs[0]
	for _, l := range locs {
		lmin, lmax = math.Min(lmin, l), math.Max(lmax, l)
	}
	if lmin == lmax || (lmin <= 0 && 0 <= lmax) {
		return
	}
	absMin, absMax := math.Abs(lmin), math.Abs(lmax)
	if absMin > absMax {
		absMin, absMax = absMax, absMin
	}
	sign := math.Copysign(1, lmin)
	oomMax := int(math.Ceil(log10(absMax)))
	oom := oomMax
	for floorDiv(absMin, pow10(oom)) == floorDiv(absMax, pow10(oom)) {
		oom--
	}
	oom++
	if (absMax-absMin)/pow10(oom) <= 1e-2 {
		oom = oomMax
		for floorDiv(absMax, pow10(oom))-floorDiv(absMin, pow10(oom)) <= 1 {
			oom--
		}
		oom++
	}
	if q := floorDiv(absMax, pow10(oom)); q >= pow10(f.OffsetThreshold-1) {
		f.offset = float64(sign*q) * pow10(oom)
	}
}

func (f *ScalarFormatter) setOrderOfMagnitude() {
	if !f.Scientific {
		return
	}
	if f.PowerLimits[0] == f.PowerLimits[1] && f.PowerLimits[0] != 0 {
		f.oom = f.PowerLimits[0]
		return
	}
	locs := f.visible()
	if len(locs) == 0 {
		return
	}
	var oom int
	if f.offset != 0 {
		oom = int(math.Floor(log10(f.vmax - f.vmin)))
	} else {
		val := 0.0
		for _, l := range locs {
			val = math.Max(val, math.Abs(l))
		}
		if val != 0 {
			oom = int(math.Floor(log10(val)))
		}
	}
	if oom <= f.PowerLimits[0] || oom >= f.PowerLimits[1] {
		f.oom = oom
	}
}

func (f *ScalarFormatter) setFormat() {
	src := f.locs
	if len(src) < 2 {
		src = append(append([]float64(nil), src...), f.vmin, f.vmax)
	}
	locs := make([]float64, len(src))
	p := pow10(f.oom)
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, l := range src {
		locs[i] = (l - f.offset) / p
		lo, hi = math.Min(lo, locs[i]), math.Max(hi, locs[i])
	}
	locRange := hi - lo
	if locRange == 0 {
		locRange = math.Max(math.Abs(lo), math.Abs(hi))
	}
	if locRange == 0 {
		locRange = 1
	}
	if len(f.locs) < 2 {
		locs = locs[:len(locs)-2]
	}
	rangeOOM := int(math.Floor(log10(locRange)))
	sigfigs := max(0, 3-rangeOOM)
	thresh := 1e-3 * pow10(rangeOOM)
	for sigfigs >= 0 {
		worst := 0.0
		for _, l := range locs {
			worst = math.Max(worst, math.Abs(l-npRound(l, sigfigs)))
		}
		if worst >= thresh {
			break
		}
		sigfigs--
	}
	f.prec = sigfigs + 1
}

// Format implements Formatter.
func (f *ScalarFormatter) Format(x float64) string {
	if len(f.locs) == 0 {
		return ""
	}
	xp := (x - f.offset) / pow10(f.oom)
	if math.Abs(xp) < 1e-8 {
		xp = 0
	}
	s := fmtF(xp, f.prec)
	if f.UseMathText {
		s = `$\mathdefault{` + s + `}$`
	}
	return f.minus(s)
}

// formatData is matplotlib's ScalarFormatter.format_data, the full-precision
// form of a value used for the offset text.
func (f *ScalarFormatter) formatData(v float64) string {
	e := int(math.Floor(log10(math.Abs(v))))
	s := pyRound(v/pow10(e), 10)
	var sig string
	if math.Mod(s, 1) == 0 {
		sig = strconv.Itoa(int(s))
	} else {
		sig = fmtG(s, 10)
	}
	if e == 0 {
		return sig
	}
	exp := strconv.Itoa(e)
	if f.UseMathText {
		exp = "10^{" + exp + "}"
		if s == 1 {
			return exp
		}
		return sig + ` \times ` + exp
	}
	return sig + "e" + exp
}

// Offset implements Formatter.
func (f *ScalarFormatter) Offset() string {
	if len(f.locs) == 0 || (f.oom == 0 && f.offset == 0) {
		return ""
	}
	var offsetStr, sciStr string
	if f.offset != 0 {
		offsetStr = f.formatData(f.offset)
		if f.offset > 0 {
			offsetStr = "+" + offsetStr
		}
	}
	if f.oom != 0 {
		if f.UseMathText {
			sciStr = f.formatData(pow10(f.oom))
		} else {
			sciStr = "1e" + strconv.Itoa(f.oom)
		}
	}
	if f.UseMathText {
		if sciStr != "" {
			sciStr = `\times\mathdefault{` + sciStr + `}`
		}
		return f.minus(`$` + sciStr + `\mathdefault{` + offsetStr + `}$`)
	}
	return f.minus(sciStr + offsetStr)
}

// LogFormatterSciNotation is matplotlib's formatter for log axes: powers of
// the base as TeX ($10^{3}$), and other values as a coefficient times a
// power. Which non-decade ticks get a label at all depends on how many
// decades the view spans (MinorThresholds).
type LogFormatterSciNotation struct {
	Base          float64 // zero means 10
	LabelOnlyBase bool
	// MinorThresholds are matplotlib's minor_thresholds: label only the
	// powers of the base when more than [0] decade boundaries are in view; a
	// log-spaced subset when the view spans more than [1] decades; everything
	// otherwise. The zero value means matplotlib's default (1, 0.4).
	MinorThresholds [2]float64

	sublabels map[float64]bool // nil: label everything
}

// NewLogFormatterSciNotation returns the formatter matplotlib puts on a log
// axis: labelOnlyBase false, minor thresholds (1, 0.4). Its labels are TeX,
// where a minus is already a minus sign.
func NewLogFormatterSciNotation() *LogFormatterSciNotation {
	return &LogFormatterSciNotation{}
}

func (f *LogFormatterSciNotation) base() float64 {
	if f.Base == 0 {
		return 10
	}
	return f.Base
}

func (f *LogFormatterSciNotation) thresholds() [2]float64 {
	if f.MinorThresholds == [2]float64{} {
		return [2]float64{1, 0.4}
	}
	return f.MinorThresholds
}

// SetLocs implements Formatter. Only the view interval matters.
func (f *LogFormatterSciNotation) SetLocs(_ []float64, vmin, vmax float64) {
	th := f.thresholds()
	if math.IsInf(th[0], 1) {
		f.sublabels = nil
		return
	}
	if vmin > vmax {
		vmin, vmax = vmax, vmin
	}
	if vmin <= 0 {
		f.sublabels = map[float64]bool{1: true}
		return
	}
	b := f.base()
	lmin := math.Log(vmin) / math.Log(b)
	lmax := math.Log(vmax) / math.Log(b)
	numticks := math.Floor(lmax) - math.Floor(math.Nextafter(lmin, math.Inf(-1)))
	numdec := math.Abs(lmax - lmin)
	switch {
	case numticks > th[0]:
		f.sublabels = map[float64]bool{1: true}
	case numdec > th[1]:
		// np.round(np.geomspace(1, b, int(b)//2 + 1)): {1,2,3,4,6,10} for 10.
		n := int(b)/2 + 1
		f.sublabels = map[float64]bool{}
		for i := range n {
			v := math.Pow(b, float64(i)/float64(n-1))
			f.sublabels[math.RoundToEven(v)] = true
		}
	default:
		f.sublabels = map[float64]bool{}
		for c := 1.0; c <= b; c++ {
			f.sublabels[c] = true
		}
	}
}

// Format implements Formatter.
func (f *LogFormatterSciNotation) Format(x float64) string {
	if x == 0 {
		return `$\mathdefault{0}$`
	}
	sign := ""
	if x < 0 {
		sign = "-"
	}
	x = math.Abs(x)
	b := f.base()
	fx := math.Log(x) / math.Log(b)
	isDecade := isCloseToInt(fx)
	exponent := math.Floor(fx)
	if isDecade {
		exponent = math.RoundToEven(fx)
	}
	// matplotlib computes b**(fx - exponent), which lands within an ulp of
	// a half-integer for x = 6, b = 2 and rounds to either side depending on
	// the platform's pow; x / b**exponent is exact for the powers of 2 and
	// 10 and gives the answer every platform should.
	coeff := math.RoundToEven(x / math.Pow(b, exponent))
	if f.LabelOnlyBase && !isDecade {
		return ""
	}
	if f.sublabels != nil && !f.sublabels[coeff] {
		return ""
	}
	bs := fmtG(b, 17)
	var s string
	if isDecade {
		s = `$\mathdefault{` + sign + bs + `^{` + strconv.Itoa(int(math.RoundToEven(fx))) + `}}$`
	} else {
		e := math.Floor(fx)
		c := x / math.Pow(b, e)
		if isCloseToInt(c) {
			c = math.RoundToEven(c)
		}
		s = `$\mathdefault{` + sign + fmtG(c, 6) + `\times` + bs + `^{` + strconv.Itoa(int(e)) + `}}$`
	}
	return s
}

// Offset implements Formatter: log axes have none.
func (f *LogFormatterSciNotation) Offset() string { return "" }
