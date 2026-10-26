// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

// textMode says whether a string is read for TeX math, as matplotlib's
// Text.parse_math does: inherit the figure's default (rcParams
// "text.parse_math"), or force one way.
type textMode uint8

const (
	modeInherit textMode = iota
	modePlain
	modeMath
)

// PlainText sets a string as it is, with no TeX, as matplotlib's
// parse_math=False: "$5 and $6" is two dollar signs, a backslash is a
// backslash, and nothing reaches the TeX typesetter. Use it for strings that
// come from data.
//
// It applies to what the call that takes it draws: SetTitle, SetXLabel,
// SetYLabel, Text, Annotate, Legend (every entry), and Pie's labels and
// legend entries. ParseMath(false) on the figure does the same for every
// string at once.
func PlainText() Option { return func(p *props) { p.textMode = modePlain } }

// MathText reads a string for TeX math between dollar signs even on a
// figure made with ParseMath(false), as matplotlib's parse_math=True on one
// Text.
func MathText() Option { return func(p *props) { p.textMode = modeMath } }

// ParseMath sets the figure's default, as matplotlib's rcParams
// "text.parse_math" (true by default): with false, every string a caller
// gives — titles, axis labels, tick labels given to SetXTicks/SetYTicks,
// legend entries, texts, annotations, pie labels — is set as it is, unless
// its own call says MathText. The tick labels go-plot writes itself (the
// formatters' $\mathdefault{10^{3}}$) stay math: they are not data.
func ParseMath(on bool) FigOption { return func(f *Figure) { f.plainText = !on } }

// SetParseMath changes the figure's default after it is made; see
// ParseMath.
func (f *Figure) SetParseMath(on bool) { f.plainText = !on }

// plain resolves a string's mode against the figure's default.
func (f *Figure) plain(m textMode) bool {
	switch m {
	case modePlain:
		return true
	case modeMath:
		return false
	}
	return f.plainText
}

// labelText is a caller's string and how to read it.
type labelText struct {
	s    string
	mode textMode
}

func newLabel(s string, opts []Option) labelText {
	return labelText{s: s, mode: collect(opts).textMode}
}
