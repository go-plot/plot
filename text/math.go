// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package text

import (
	"math"
	"strconv"
	"strings"

	"github.com/go-plot/plot/render"
)

// texSize is the pixel size go-tex/math typesets at; its integer sizes are
// scaled to the label's fractional size afterwards.
const texSize = 100

// math lays out one piece of TeX math and returns it with its advance.
func (d *Default) math(tex string, sizePx float64) (Layout, float64) {
	tex = strings.TrimSpace(tex)
	if inner, ok := strings.CutPrefix(tex, `\mathdefault{`); ok && strings.HasSuffix(inner, "}") {
		if l, ok := d.mathDefault(inner[:len(inner)-1], sizePx); ok {
			return l, l.Width
		}
	}
	src := strings.ReplaceAll(tex, `\mathdefault{`, `{`)
	svg, m, err := d.tex.RenderSVGMetrics(src, texSize)
	if err != nil {
		// matplotlib would refuse the whole figure; showing the source is
		// more useful than failing a render.
		r := d.setText("$"+tex+"$", sizePx, 0, 0)
		return Layout{Path: r.path, Width: r.advance, Ascent: -r.inkTop, Descent: math.Max(0, r.inkBottom)}, r.advance
	}
	k := sizePx / texSize
	p := parseTeXSVG(svg, render.Translate(0, -m.Height).Then(render.Scale(k, k)))
	return Layout{Path: p, Width: m.Width * k, Ascent: m.Height * k, Descent: m.Depth * k}, m.Width * k
}

// mathDefault sets the body of a \mathdefault{...} in the text font, the
// way matplotlib's mathtext sets tick labels such as 10^{-3} and
// 2\times10^{1}: characters, \times, \cdot, and one level of ^{...} and
// _{...} applied to the preceding character. It reports false for anything
// else, which then goes to the TeX typesetter.
func (d *Default) mathDefault(s string, sizePx float64) (Layout, bool) {
	xh := d.xh / d.upem * sizePx
	out := Layout{Path: &render.Path{}}
	pen := 0.0
	add := func(text string, size, shift float64) {
		r := d.setText(text, size, pen, shift)
		out.Path.Append(r.path, render.Identity)
		if r.hasInk {
			out.Ascent = math.Max(out.Ascent, -(r.inkTop + shift))
			out.Descent = math.Max(out.Descent, r.inkBottom+shift)
		}
		pen += r.advance
	}
	for len(s) > 0 {
		switch {
		case strings.HasPrefix(s, `\times`):
			add("×", sizePx, 0)
			s = s[len(`\times`):]
		case strings.HasPrefix(s, `\cdot`):
			add("⋅", sizePx, 0)
			s = s[len(`\cdot`):]
		case s[0] == '^' || s[0] == '_':
			if len(s) < 2 || s[1] != '{' {
				return Layout{}, false
			}
			end := strings.IndexByte(s, '}')
			if end < 0 {
				return Layout{}, false
			}
			body := s[2:end]
			if strings.ContainsAny(body, `\{^_`) {
				return Layout{}, false
			}
			body = strings.ReplaceAll(body, "-", "−")
			pen += 0.025 * xh // consts.delta * x_height
			small := sizePx * 0.7
			if s[0] == '^' {
				// shift_up = max(sup1 * x_height, depth + x_height / 4)
				probe := d.setText(body, small, 0, 0)
				up := math.Max(845.824/1120*xh, math.Max(0, probe.inkBottom)+xh/4)
				add(body, small, -up)
			} else {
				probe := d.setText(body, small, 0, 0)
				down := math.Max(307.199/1120*xh, -probe.inkTop-xh*4/5)
				add(body, small, down)
			}
			pen += 0.05 * xh // consts.script_space * x_height
			s = s[end+1:]
		case s[0] == '\\' || s[0] == '{' || s[0] == '}':
			return Layout{}, false
		default:
			ch := s[0:1]
			if ch == "-" {
				ch = "−"
			}
			add(ch, sizePx, 0)
			s = s[1:]
		}
	}
	out.Width = pen
	return out, true
}

// parseTeXSVG reads the subset of SVG go-tex/math writes — nested groups
// with translate or scale transforms, paths of absolute M/L/H/V/Q/C/Z
// commands, and rectangles — into one path, transformed by base.
func parseTeXSVG(svg string, base render.Affine) *render.Path {
	out := &render.Path{}
	stack := []render.Affine{base}
	for {
		i := strings.IndexByte(svg, '<')
		if i < 0 {
			return out
		}
		svg = svg[i+1:]
		j := strings.IndexByte(svg, '>')
		tag := svg[:j]
		svg = svg[j+1:]
		cur := stack[len(stack)-1]
		switch {
		case strings.HasPrefix(tag, "/g"):
			stack = stack[:len(stack)-1]
		case strings.HasPrefix(tag, "g"):
			m := parseTransform(attr(tag, "transform")).Then(cur)
			if strings.HasSuffix(tag, "/") {
				continue
			}
			stack = append(stack, m)
		case strings.HasPrefix(tag, "path"):
			appendPathData(out, attr(tag, "d"), parseTransform(attr(tag, "transform")).Then(cur))
		case strings.HasPrefix(tag, "rect"):
			x, _ := strconv.ParseFloat(attr(tag, "x"), 64)
			y, _ := strconv.ParseFloat(attr(tag, "y"), 64)
			w, _ := strconv.ParseFloat(attr(tag, "width"), 64)
			h, _ := strconv.ParseFloat(attr(tag, "height"), 64)
			var r render.Path
			r.Rect(x, y, w, h)
			out.Append(&r, cur)
		}
	}
}

// attr returns the value of attribute name in a tag, or "".
func attr(tag, name string) string {
	k := strings.Index(tag, " "+name+`="`)
	if k < 0 {
		return ""
	}
	v := tag[k+len(name)+3:]
	return v[:strings.IndexByte(v, '"')]
}

// parseTransform reads a sequence of translate(...) and scale(...) as SVG
// composes them (the first applies last). Anything else is ignored.
func parseTransform(s string) render.Affine {
	m := render.Identity
	for {
		open := strings.IndexByte(s, '(')
		if open < 0 {
			return m
		}
		name := strings.TrimSpace(s[:open])
		closing := strings.IndexByte(s, ')')
		args := strings.FieldsFunc(s[open+1:closing], func(r rune) bool { return r == ',' || r == ' ' })
		s = s[closing+1:]
		v := make([]float64, len(args))
		for i, a := range args {
			v[i], _ = strconv.ParseFloat(a, 64)
		}
		var t render.Affine
		switch {
		case name == "translate" && len(v) == 2:
			t = render.Translate(v[0], v[1])
		case name == "translate" && len(v) == 1:
			t = render.Translate(v[0], 0)
		case name == "scale" && len(v) == 2:
			t = render.Scale(v[0], v[1])
		case name == "scale" && len(v) == 1:
			t = render.Scale(v[0], v[0])
		default:
			continue
		}
		m = t.Then(m)
	}
}

// appendPathData appends an SVG path "d" of absolute commands.
func appendPathData(out *render.Path, d string, m render.Affine) {
	f := strings.FieldsFunc(d, func(r rune) bool { return r == ' ' || r == ',' })
	// Split letters glued to numbers ("M3.4" -> "M", "3.4").
	var tok []string
	for _, t := range f {
		for len(t) > 0 {
			k := strings.IndexAny(t[1:], "MLHVQCZz")
			if k < 0 {
				tok = append(tok, t)
				break
			}
			tok = append(tok, t[:k+1])
			t = t[k+1:]
		}
	}
	var cmd byte
	var cx, cy float64
	nums := func(n int) []float64 {
		v := make([]float64, n)
		for i := range n {
			if len(tok) == 0 {
				return nil
			}
			v[i], _ = strconv.ParseFloat(tok[0], 64)
			tok = tok[1:]
		}
		return v
	}
	pt := func(x, y float64) render.Pt { return m.Apply(render.Pt{X: x, Y: y}) }
	for len(tok) > 0 {
		t := tok[0]
		if c := t[0]; len(t) == 1 && strings.IndexByte("MLHVQCZz", c) >= 0 {
			cmd = c
			tok = tok[1:]
			if c == 'Z' || c == 'z' {
				out.Close()
				continue
			}
		}
		switch cmd {
		case 'M', 'L':
			v := nums(2)
			if v == nil {
				return
			}
			cx, cy = v[0], v[1]
			p := pt(cx, cy)
			if cmd == 'M' {
				out.MoveTo(p.X, p.Y)
				cmd = 'L'
			} else {
				out.LineTo(p.X, p.Y)
			}
		case 'H', 'V':
			v := nums(1)
			if v == nil {
				return
			}
			if cmd == 'H' {
				cx = v[0]
			} else {
				cy = v[0]
			}
			p := pt(cx, cy)
			out.LineTo(p.X, p.Y)
		case 'Q':
			v := nums(4)
			if v == nil {
				return
			}
			a, b := pt(v[0], v[1]), pt(v[2], v[3])
			cx, cy = v[2], v[3]
			out.QuadTo(a.X, a.Y, b.X, b.Y)
		case 'C':
			v := nums(6)
			if v == nil {
				return
			}
			a, b, c := pt(v[0], v[1]), pt(v[2], v[3]), pt(v[4], v[5])
			cx, cy = v[4], v[5]
			out.CubicTo(a.X, a.Y, b.X, b.Y, c.X, c.Y)
		default:
			return // numbers before any command, or a closed path followed by numbers
		}
	}
}
