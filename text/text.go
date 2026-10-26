// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

// Package text turns label strings into glyph outlines. Plain text is set in
// DejaVu Sans (matplotlib's default font) through go-opentype; TeX between
// dollar signs is typeset by go-tex/math, a real TeX math typesetter, except
// for matplotlib's \mathdefault{...} (what tick labels use), which is set in
// the text font with matplotlib's own superscript and subscript rules so that
// tick labels look like matplotlib's.
package text

import (
	"encoding/binary"
	"math"
	"strings"
	"sync"

	"github.com/go-opentype/fonts/dejavusans"
	"github.com/go-opentype/opentype"
	texmath "github.com/go-tex/math"

	"github.com/go-plot/plot/render"
)

// Layout is a laid-out line of text. Path holds the glyph outlines in pixels
// with the pen origin at (0, 0) on the baseline and y pointing down.
//
// Width, Ascent and Descent are the metrics matplotlib aligns text by. The
// box starts at the pen origin. For plain text the width is the advance of
// the string (kerned), which is what FreeType's bounding box of the
// rendered string comes to in matplotlib within a pixel or two (matplotlib's
// is hinted, each glyph a whole number of pixels); the ascent and descent
// are those of the ink. For math, the TeX box.
type Layout struct {
	Path    *render.Path
	Width   float64
	Ascent  float64 // above the baseline
	Descent float64 // below the baseline
}

// A Renderer lays out one line of text at a size in pixels.
type Renderer interface {
	Layout(s string, sizePx float64) Layout
	// LineMetrics returns the font's typographic ascent, descent and line gap
	// at sizePx: the minimum box matplotlib gives every line.
	LineMetrics(sizePx float64) (ascent, descent, gap float64)
}

// Default is the built-in Renderer: DejaVu Sans for text, go-tex/math for TeX.
type Default struct {
	font    *opentype.Font
	face    *opentype.Face
	upem    float64
	asc     float64 // OS/2 sTypoAscender, font units
	desc    float64 // -sTypoDescender
	gap     float64
	xh      float64 // x-height, font units
	texOnce sync.Once
	tex     *texmath.Renderer
	mu      sync.Mutex
	glyphs  map[opentype.GlyphIndex]glyph
}

type glyph struct {
	segs    []opentype.Segment
	advance float64 // font units
	ok      bool
}

var (
	defaultOnce sync.Once
	defaultR    *Default
)

// NewDefault returns the shared default Renderer.
func NewDefault() *Default {
	defaultOnce.Do(func() {
		f := must(opentype.Parse(dejavusans.TTF))
		d := &Default{
			font:   f,
			face:   f.NewFace(f.UnitsPerEm()),
			upem:   float64(f.UnitsPerEm()),
			glyphs: map[opentype.GlyphIndex]glyph{},
		}
		// matplotlib's x-height is the height of the ink of "x".
		gid, _ := f.GlyphIndex('x')
		for _, sg := range d.glyph(gid).segs {
			for _, p := range sg.P {
				d.xh = math.Max(d.xh, p.Y)
			}
		}
		// matplotlib reads the line metrics from the OS/2 table.
		os2, _ := f.Table("OS/2")
		d.asc = float64(int16(binary.BigEndian.Uint16(os2[68:])))
		d.desc = -float64(int16(binary.BigEndian.Uint16(os2[70:])))
		d.gap = float64(int16(binary.BigEndian.Uint16(os2[72:])))
		defaultR = d
	})
	return defaultR
}

// texRenderer returns the TeX typesetter, loading its font (STIX Two Math)
// the first time a label needs it.
func (d *Default) texRenderer() *texmath.Renderer {
	d.texOnce.Do(func() {
		d.tex = must(texmath.New(texmath.DefaultFont()))
	})
	return d.tex
}

// must returns v, or panics with err: the fonts it guards are embedded, so a
// failure is a broken build, not a condition to handle.
func must[T any](v T, err error) T {
	if err != nil {
		panic("text: embedded font: " + err.Error())
	}
	return v
}

// LineMetrics implements Renderer.
func (d *Default) LineMetrics(sizePx float64) (float64, float64, float64) {
	s := sizePx / d.upem
	return d.asc * s, d.desc * s, d.gap * s
}

func (d *Default) glyph(gid opentype.GlyphIndex) glyph {
	d.mu.Lock()
	defer d.mu.Unlock()
	g, ok := d.glyphs[gid]
	if !ok {
		g.segs, g.ok = d.face.GlyphOutline(gid)
		g.advance = d.face.AdvanceIndexUnits(gid)
		d.glyphs[gid] = g
	}
	return g
}

// run is text set in the text font: its outline, its advance and its ink.
type run struct {
	path              *render.Path
	advance           float64
	inkX0, inkX1      float64
	inkTop, inkBottom float64 // y down: top is negative above the baseline
	hasInk            bool
}

// setText lays out s in the text font at sizePx with the pen at (x, y).
func (d *Default) setText(s string, sizePx, x, y float64) run {
	sc := sizePx / d.upem
	r := run{path: &render.Path{}, inkX0: math.Inf(1), inkX1: math.Inf(-1), inkTop: math.Inf(1), inkBottom: math.Inf(-1)}
	pen := 0.0
	var prev opentype.GlyphIndex
	for i, ch := range s {
		gid, _ := d.font.GlyphIndex(ch)
		if i > 0 {
			pen += d.face.KernIndexUnits(prev, gid)
		}
		prev = gid
		g := d.glyph(gid)
		for _, sg := range g.segs {
			var pts [2]render.Pt
			for k := range sg.P {
				pts[k] = render.Pt{X: x + (pen+sg.P[k].X)*sc, Y: y - sg.P[k].Y*sc}
			}
			switch sg.Op {
			case opentype.SegMoveTo:
				r.path.MoveTo(pts[0].X, pts[0].Y)
			case opentype.SegLineTo:
				r.path.LineTo(pts[0].X, pts[0].Y)
			case opentype.SegQuadTo:
				r.path.QuadTo(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y)
			case opentype.SegClose:
				r.path.Close()
				continue
			}
			n := 1
			if sg.Op == opentype.SegQuadTo {
				n = 2
			}
			for _, p := range pts[:n] {
				r.hasInk = true
				r.inkX0, r.inkX1 = math.Min(r.inkX0, p.X-x), math.Max(r.inkX1, p.X-x)
				r.inkTop, r.inkBottom = math.Min(r.inkTop, p.Y-y), math.Max(r.inkBottom, p.Y-y)
			}
		}
		pen += g.advance
	}
	r.advance = pen * sc
	if !r.hasInk {
		r.inkX0, r.inkX1, r.inkTop, r.inkBottom = 0, 0, 0, 0
	}
	return r
}

// A PlainRenderer can also lay out a string as it is, with no TeX: dollar
// signs and backslashes are characters. Text that comes from data (a
// category name, a customer's label) goes through it, so that "$5 and $6"
// is not typeset as math and no input reaches the TeX typesetter.
type PlainRenderer interface {
	Renderer
	LayoutPlain(s string, sizePx float64) Layout
}

// LayoutPlain implements PlainRenderer.
func (d *Default) LayoutPlain(s string, sizePx float64) Layout {
	r := d.setText(s, sizePx, 0, 0)
	return Layout{
		Path:    r.path,
		Width:   r.advance,
		Ascent:  -r.inkTop,
		Descent: math.Max(0, r.inkBottom),
	}
}

// Layout implements Renderer.
func (d *Default) Layout(s string, sizePx float64) Layout {
	parts := splitMath(s)
	if len(parts) == 1 && !parts[0].math {
		// parts[0].s, not s: "cost \$5" is "cost $5" (v0.1 drew the
		// backslash).
		return d.LayoutPlain(parts[0].s, sizePx)
	}
	out := Layout{Path: &render.Path{}}
	pen := 0.0
	for _, p := range parts {
		var l Layout
		var adv float64
		if p.math {
			l, adv = d.math(p.s, sizePx)
		} else {
			r := d.setText(p.s, sizePx, 0, 0)
			l = Layout{Path: r.path, Ascent: -r.inkTop, Descent: math.Max(0, r.inkBottom)}
			adv = r.advance
		}
		out.Path.Append(l.Path, render.Translate(pen, 0))
		out.Ascent = math.Max(out.Ascent, l.Ascent)
		out.Descent = math.Max(out.Descent, l.Descent)
		pen += adv
	}
	out.Width = pen
	return out
}

type part struct {
	s    string
	math bool
}

// splitMath splits s at unescaped dollar signs into text and math parts, as
// matplotlib does: an odd number of dollars means no math at all, and \$ is a
// literal dollar.
func splitMath(s string) []part {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == '$' {
			n++
		}
	}
	if n == 0 || n%2 == 1 {
		return []part{{s: strings.ReplaceAll(s, `\$`, "$")}}
	}
	var out []part
	var cur strings.Builder
	inMath := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) && s[i+1] == '$' {
			if inMath {
				cur.WriteString(`\$`)
			} else {
				cur.WriteByte('$')
			}
			i++
			continue
		}
		if c == '$' {
			if cur.Len() > 0 {
				out = append(out, part{s: cur.String(), math: inMath})
			}
			cur.Reset()
			inMath = !inMath
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, part{s: cur.String()})
	}
	return out
}
