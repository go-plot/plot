// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ticker

import (
	"math"
	"strconv"
	"strings"
)

// The tick algorithms are ported from matplotlib, and matplotlib's results
// depend on exactly how Python and NumPy do floating-point arithmetic: floor
// division, correctly rounded powers of ten, round-half-even. The helpers in
// this file reproduce those semantics, so that a port that reads the same also
// computes the same.
//
// One trap is Go's own: on arm64, ppc64le and s390x the compiler fuses x*y+z
// into a single fused multiply-add, rounding once where Python rounds twice.
// Every product that is then added to is therefore converted explicitly
// (float64(x*y)), which the Go specification defines as a rounding point.

// pow10 returns 10**n correctly rounded, as Python's 10.0**n is. math.Pow10
// is not: math.Pow10(-23) != 1e-23.
func pow10(n int) float64 {
	v, _ := strconv.ParseFloat("1e"+strconv.Itoa(n), 64)
	return v
}

// log10 returns the base-10 logarithm of x, exact at the powers of ten, which
// math.Log10 is not (math.Log10(1e15) is 14.999999999999998) and the C library
// that NumPy and Python call is.
func log10(x float64) float64 {
	l := math.Log10(x)
	if r := math.Round(l); math.Abs(r) < 400 && pow10(int(r)) == x {
		return r
	}
	return l
}

// pyDivmod is Python's divmod for floats: the quotient is floored and the
// remainder takes the sign of the divisor.
func pyDivmod(x, y float64) (div, mod float64) {
	mod = math.Mod(x, y)
	div = (x - mod) / y
	if mod != 0 {
		if (y < 0) != (mod < 0) {
			mod += y
			div -= 1
		}
	} else {
		mod = math.Copysign(0, y)
	}
	if div != 0 {
		fl := math.Floor(div)
		if div-fl > 0.5 {
			fl++
		}
		div = fl
	} else {
		div = math.Copysign(0, x/y)
	}
	return div, mod
}

// floorDiv is Python's (and NumPy's) x // y for floats.
func floorDiv(x, y float64) float64 {
	d, _ := pyDivmod(x, y)
	return d
}

// npRound is numpy.round(x, decimals) for decimals >= 0: scale, round half to
// even, unscale.
func npRound(x float64, decimals int) float64 {
	p := pow10(decimals)
	return math.RoundToEven(float64(x*p)) / p
}

// pyRound is Python's round(x, n): correctly rounded to n decimal places.
func pyRound(x float64, n int) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', n, 64), 64)
	return v
}

// isCloseToInt is matplotlib's _is_close_to_int: math.isclose(x, round(x)).
func isCloseToInt(x float64) bool {
	r := math.RoundToEven(x)
	return math.Abs(x-r) <= 1e-9*math.Max(math.Abs(x), math.Abs(r))
}

// fmtF is Python's '%1.<prec>f' % x.
func fmtF(x float64, prec int) string { return strconv.FormatFloat(x, 'f', prec, 64) }

// fmtG is Python's '%1.<prec>g' % x (and '%g' with prec 6): the shorter of
// fixed and exponent notation by C's rule, trailing zeros removed, and an
// exponent of at least two digits.
func fmtG(x float64, prec int) string {
	if x == 0 {
		if math.Signbit(x) {
			return "-0"
		}
		return "0"
	}
	// The exponent C uses is that of the value after rounding to prec digits.
	e := strconv.FormatFloat(x, 'e', prec-1, 64)
	mant, expPart, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expPart)
	if exp < -4 || exp >= prec {
		mant = trimZeros(mant)
		sign := "+"
		if exp < 0 {
			sign = "-"
			exp = -exp
		}
		es := strconv.Itoa(exp)
		if len(es) < 2 {
			es = "0" + es
		}
		return mant + "e" + sign + es
	}
	return trimZeros(strconv.FormatFloat(x, 'f', prec-1-exp, 64))
}

// trimZeros removes trailing zeros after a decimal point, and the point too if
// nothing is left after it.
func trimZeros(s string) string {
	if !strings.Contains(s, ".") {
		return s
	}
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

// FixMinus replaces every hyphen-minus with U+2212 MINUS SIGN, as matplotlib
// does when axes.unicode_minus is on (its default).
func FixMinus(s string) string { return strings.ReplaceAll(s, "-", "−") }

// Nonsingular is matplotlib's transforms.nonsingular: it widens an interval
// that is empty, tiny or not finite so that a scale can be built on it. With
// increasing set the result is ordered; otherwise a swapped input stays
// swapped.
func Nonsingular(vmin, vmax, expander, tiny float64, increasing bool) (float64, float64) {
	if math.IsInf(vmin, 0) || math.IsInf(vmax, 0) || math.IsNaN(vmin) || math.IsNaN(vmax) {
		return -expander, expander
	}
	swapped := false
	if vmax < vmin {
		vmin, vmax = vmax, vmin
		swapped = true
	}
	// Python ints would not overflow here; these are floats already.
	maxabs := math.Max(math.Abs(vmin), math.Abs(vmax))
	if maxabs < (1e6/tiny)*2.2250738585072014e-308 {
		vmin, vmax = -expander, expander
	} else if vmax-vmin <= maxabs*tiny {
		if vmax == 0 && vmin == 0 {
			vmin, vmax = -expander, expander
		} else {
			vmin -= float64(expander * math.Abs(vmin))
			vmax += float64(expander * math.Abs(vmax))
		}
	}
	if swapped && !increasing {
		vmin, vmax = vmax, vmin
	}
	return vmin, vmax
}
