// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"math"
	"strconv"

	"github.com/go-plot/plot/colors"
)

// SVG renders a figure as an SVG document. Coordinates are written in
// pixels at the figure's dpi, in a viewBox whose physical size is given in
// points, as matplotlib does; text arrives here already turned into glyph
// outlines, so the document needs no font.
type SVG struct {
	w, h   int
	dpi    float64
	body   bytes.Buffer
	defs   bytes.Buffer
	clipN  int
	scratc []byte
}

// NewSVG returns an SVG renderer for a w x h pixel figure at dpi.
func NewSVG(w, h int, dpi float64) *SVG { return &SVG{w: w, h: h, dpi: dpi} }

// Size implements Renderer.
func (s *SVG) Size() (int, int) { return s.w, s.h }

// BeginGroup implements Renderer.
func (s *SVG) BeginGroup(id string) { fmt.Fprintf(&s.body, "<g id=%q>\n", id) }

// EndGroup implements Renderer.
func (s *SVG) EndGroup() { s.body.WriteString("</g>\n") }

// PushClip implements Renderer.
func (s *SVG) PushClip(x, y, w, h float64) {
	s.clipN++
	id := "c" + strconv.Itoa(s.clipN)
	s.defs.WriteString(`<clipPath id="` + id + `"><rect x="`)
	s.defs.Write(num(nil, x))
	s.defs.WriteString(`" y="`)
	s.defs.Write(num(nil, y))
	s.defs.WriteString(`" width="`)
	s.defs.Write(num(nil, w))
	s.defs.WriteString(`" height="`)
	s.defs.Write(num(nil, h))
	s.defs.WriteString("\"/></clipPath>\n")
	s.body.WriteString(`<g clip-path="url(#` + id + `)">` + "\n")
}

// PopClip implements Renderer.
func (s *SVG) PopClip() { s.body.WriteString("</g>\n") }

// num appends v rounded to 1/1000 pixel, without trailing zeros.
func num(b []byte, v float64) []byte {
	v = math.Round(v*1000) / 1000
	if v == 0 {
		v = 0 // no "-0"
	}
	return strconv.AppendFloat(b, v, 'f', -1, 64)
}

func colour(c colors.RGBA) string { return c.WithAlpha(1).Hex() }

// pathData appends a path's "d" attribute value.
func pathData(b []byte, p *Path) []byte {
	cmd := [...]byte{'M', 'L', 'Q', 'C', 'z'}
	for i, sg := range p.Segs {
		if i > 0 {
			b = append(b, ' ')
		}
		b = append(b, cmd[sg.Op])
		for j, q := range sg.P[:sg.Op.points()] {
			if j > 0 {
				b = append(b, ' ')
			}
			b = num(b, q.X)
			b = append(b, ' ')
			b = num(b, q.Y)
		}
	}
	return b
}

// DrawPath implements Renderer.
func (s *SVG) DrawPath(p *Path, st Style) {
	if len(p.Segs) == 0 {
		return
	}
	b := s.scratc[:0]
	b = append(b, `<path d="`...)
	b = pathData(b, p)
	b = append(b, `" style="`...)
	if st.Fill.A > 0 {
		b = append(b, "fill:"+colour(st.Fill)...)
		if st.Fill.A < 1 {
			b = append(b, ";fill-opacity:"...)
			b = num(b, st.Fill.A)
		}
		if st.EvenOdd {
			b = append(b, ";fill-rule:evenodd"...)
		}
	} else {
		b = append(b, "fill:none"...)
	}
	if st.Width > 0 && st.Stroke.A > 0 {
		b = append(b, ";stroke:"+colour(st.Stroke)...)
		if st.Stroke.A < 1 {
			b = append(b, ";stroke-opacity:"...)
			b = num(b, st.Stroke.A)
		}
		b = append(b, ";stroke-width:"...)
		b = num(b, st.Width)
		b = append(b, [...]string{";stroke-linecap:butt", ";stroke-linecap:round", ";stroke-linecap:square"}[st.Cap]...)
		b = append(b, [...]string{";stroke-linejoin:miter", ";stroke-linejoin:round", ";stroke-linejoin:bevel"}[st.Join]...)
		if len(st.Dash) > 0 {
			b = append(b, ";stroke-dasharray:"...)
			for i, d := range st.Dash {
				if i > 0 {
					b = append(b, ',')
				}
				b = num(b, d)
			}
			if st.Offset != 0 {
				b = append(b, ";stroke-dashoffset:"...)
				b = num(b, st.Offset)
			}
		}
	}
	b = append(b, "\"/>\n"...)
	s.body.Write(b)
	s.scratc = b
}

// DrawImage implements Renderer. The image is embedded as a PNG at its own
// resolution and scaled by the viewer without smoothing.
func (s *SVG) DrawImage(im Image) {
	var buf bytes.Buffer
	img := im.Pix
	if img.Rect.Min != (image.Point{}) {
		c := image.NewRGBA(image.Rect(0, 0, img.Rect.Dx(), img.Rect.Dy()))
		draw.Draw(c, c.Rect, img, img.Rect.Min, draw.Src)
		img = c
	}
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	_ = enc.Encode(&buf, img) // writing to memory cannot fail
	b := s.scratc[:0]
	b = append(b, `<image x="`...)
	b = num(b, im.X)
	b = append(b, `" y="`...)
	b = num(b, im.Y)
	b = append(b, `" width="`...)
	b = num(b, im.W)
	b = append(b, `" height="`...)
	b = num(b, im.H)
	b = append(b, `" preserveAspectRatio="none" style="image-rendering:pixelated;image-rendering:optimizeSpeed" xlink:href="data:image/png;base64,`...)
	b = base64.StdEncoding.AppendEncode(b, buf.Bytes())
	b = append(b, "\"/>\n"...)
	s.body.Write(b)
	s.scratc = b
}

// WriteTo writes the finished document.
func (s *SVG) WriteTo(w io.Writer) (int64, error) {
	var out bytes.Buffer
	out.WriteString(`<?xml version="1.0" encoding="utf-8" standalone="no"?>` + "\n")
	out.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1" width="`)
	out.Write(num(nil, float64(s.w)*72/s.dpi))
	out.WriteString(`pt" height="`)
	out.Write(num(nil, float64(s.h)*72/s.dpi))
	out.WriteString(`pt" viewBox="0 0 ` + strconv.Itoa(s.w) + " " + strconv.Itoa(s.h) + "\">\n")
	if s.defs.Len() > 0 {
		out.WriteString("<defs>\n")
		out.Write(s.defs.Bytes())
		out.WriteString("</defs>\n")
	}
	out.Write(s.body.Bytes())
	out.WriteString("</svg>\n")
	return out.WriteTo(w)
}
