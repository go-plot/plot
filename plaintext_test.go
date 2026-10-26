// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/go-plot/plot/text"
)

// spyText records, for each string laid out, whether it went through TeX
// (Layout) or not (LayoutPlain).
type spyText struct {
	*text.Default
	mu    sync.Mutex
	plain map[string]bool
	math  map[string]bool
}

func newSpy() *spyText {
	return &spyText{Default: text.NewDefault(), plain: map[string]bool{}, math: map[string]bool{}}
}

func (s *spyText) Layout(str string, size float64) text.Layout {
	s.mu.Lock()
	s.math[str] = true
	s.mu.Unlock()
	return s.Default.Layout(str, size)
}

func (s *spyText) LayoutPlain(str string, size float64) text.Layout {
	s.mu.Lock()
	s.plain[str] = true
	s.mu.Unlock()
	return s.Default.LayoutPlain(str, size)
}

const (
	evilTeX    = `$\input{/etc/passwd}$`
	evilScript = `<script>alert(1)</script>`
	dollar     = `100$`
)

// dataFigure draws every caller string go-plot renders, each a different
// data-like value, with opts given to every call that takes options.
func dataFigure(spy *spyText, figOpts []FigOption, opts ...Option) (*Figure, []string) {
	f := NewFigure(figOpts...)
	f.Text = spy
	axs := f.Subplots(1, 3)
	bars, logs, pie := axs[0][0], axs[0][1], axs[0][2]
	names := []string{"title " + evilTeX, "xlabel " + evilScript, "ylabel " + dollar,
		"cat " + evilTeX, "cat " + dollar, "legend " + evilTeX, "text " + evilTeX, "note " + evilScript,
		"ylab2 " + evilTeX, "ytick " + evilTeX}
	bars.SetTitle(names[0], opts...)
	bars.SetXLabel(names[1], opts...)
	bars.SetYLabel(names[2], opts...)
	bars.Bar([]float64{0, 1}, []float64{3, 5}, Label(names[5]))
	bars.SetXTickLabels([]float64{0, 1}, []string{names[3], names[4]}, opts...)
	bars.Legend(opts...)
	bars.Text(0, 1, names[6], opts...)
	bars.Annotate(names[7], 0, 1, 1, 2, append([]Option{Arrow()}, opts...)...)
	bars.TwinX().SetYLabel(names[8], opts...)
	// The formatters' own labels on a log axis are TeX ($\mathdefault{...}$).
	logs.Plot([]float64{1, 10, 1000}, []float64{1, 2, 3})
	logs.SetXScale("log")
	logs.SetYTickLabels([]float64{1, 2}, []string{names[9], "two"}, opts...)
	_ = pie.Pie([]float64{1, 2}, append([]Option{Labels("pie "+evilTeX, "pie "+dollar)}, opts...)...)
	pie.Legend()
	return f, append(names, "pie "+evilTeX, "pie "+dollar)
}

func svgOf(t *testing.T, f *Figure) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := f.SaveSVG(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// checkPlain: every caller string went through LayoutPlain and none through
// TeX, while go-plot's own math tick labels still did.
func checkPlain(t *testing.T, how string, spy *spyText, names []string) {
	t.Helper()
	for _, n := range names {
		if !spy.plain[n] || spy.math[n] {
			t.Errorf("%s: %q plain=%v math=%v", how, n, spy.plain[n], spy.math[n])
		}
	}
	own := false
	for s := range spy.math {
		own = own || strings.Contains(s, `\mathdefault`)
	}
	if !own {
		t.Errorf("%s: the log axis' own labels did not go through TeX", how)
	}
}

func checkSafeSVG(t *testing.T, svg []byte) {
	t.Helper()
	doc := strings.ToLower(string(svg))
	for _, bad := range []string{"<script", "alert", "passwd", "style=", "<text", "href", "onload"} {
		if strings.Contains(doc, bad) {
			t.Errorf("the SVG contains %q", bad)
		}
	}
}

// TestParseMathFalse: ParseMath(false) on the figure sets every caller's
// string without TeX: titles, axis labels (a twin's too), tick labels given
// to SetXTickLabels/SetYTickLabels, legend entries, texts, annotations and
// pie labels.
func TestParseMathFalse(t *testing.T) {
	spy := newSpy()
	f, names := dataFigure(spy, []FigOption{ParseMath(false)})
	f.TightLayout()
	svg := svgOf(t, f)
	checkPlain(t, "ParseMath(false)", spy, names)
	checkSafeSVG(t, svg)
}

// TestPlainTextPerCall: the same with PlainText on each call and the
// figure's default left alone. Legend(PlainText()) covers the bar's entry;
// Pie's own PlainText covers its entries.
func TestPlainTextPerCall(t *testing.T) {
	spy := newSpy()
	f, names := dataFigure(spy, nil, PlainText())
	checkSafeSVG(t, svgOf(t, f))
	checkPlain(t, "PlainText", spy, names)
}

// TestMathByDefault is the control: without either, the same strings go to
// TeX, so the two tests above see a real difference. MathText on one call
// overrides ParseMath(false).
func TestMathByDefault(t *testing.T) {
	spy := newSpy()
	fig, names := dataFigure(spy, nil)
	svgOf(t, fig)
	for _, n := range names {
		if strings.Contains(n, evilTeX) && !spy.math[n] {
			t.Errorf("default: %q did not go through TeX", n)
		}
	}
	spy = newSpy()
	fig, names = dataFigure(spy, []FigOption{ParseMath(false)}, MathText())
	svgOf(t, fig)
	for _, n := range names[:5] {
		if !spy.math[n] {
			t.Errorf("MathText: %q did not go through TeX", n)
		}
	}
	fig.SetParseMath(true)
	if fig.plainText {
		t.Error("SetParseMath(true)")
	}
}

// TestPlainDollar: "100$" set plain is the string's own glyphs, a dollar
// included, with no backslash and no math.
func TestPlainDollar(t *testing.T) {
	d := text.NewDefault()
	f := NewFigure(ParseMath(false))
	p := f.layoutText(textSpec{S: dollar, Size: 20, Plain: true})
	want := d.LayoutPlain(dollar, 20)
	if len(p.path.Segs) != len(want.Path.Segs) || len(want.Path.Segs) == 0 {
		t.Errorf("100$: %d segments, want %d", len(p.path.Segs), len(want.Path.Segs))
	}
	tex := d.LayoutPlain(`$\input{x}$`, 20)
	if tex.Width <= d.Layout(`$x$`, 20).Width {
		t.Error("plain TeX source is not wider than its typeset form: was it typeset?")
	}
}
