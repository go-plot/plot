// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"fmt"
	"image"

	"github.com/go-ndarray/ndarray"
)

// Values converts plot data to []float64. It accepts []float64, []float32,
// []int, []int64, []int32, a single float64 or int, any *ndarray.Array (of
// any dtype, flattened in row-major order; complex values keep their real
// part, as NumPy's astype does) and nil (which gives nil). Anything else panics: passing a type a plot cannot draw
// is a programming error, like an index out of range.
func Values(v any) []float64 {
	switch d := v.(type) {
	case nil:
		return nil
	case float64:
		return []float64{d}
	case int:
		return []float64{float64(d)}
	case []float64:
		return d
	case []float32:
		return convert(d)
	case []int:
		return convert(d)
	case []int64:
		return convert(d)
	case []int32:
		return convert(d)
	case *ndarray.Array:
		if f, ok := ndarray.Data[float64](d); ok {
			return f
		}
		f, _ := ndarray.Data[float64](d.AsType(ndarray.Float64))
		return f
	}
	panic(fmt.Sprintf("plot: cannot plot data of type %T", v))
}

func convert[T float32 | int | int64 | int32](s []T) []float64 {
	out := make([]float64, len(s))
	for i, v := range s {
		out[i] = float64(v)
	}
	return out
}

// gridData is a 2-D array of values in row-major order: Rows x Cols.
type gridData struct {
	Rows, Cols int
	Z          []float64
}

// grid converts 2-D plot data: a 2-D *ndarray.Array of any dtype, or
// [][]float64 with rows of equal length.
func grid(v any) gridData {
	switch d := v.(type) {
	case [][]float64:
		g := gridData{Rows: len(d)}
		if g.Rows > 0 {
			g.Cols = len(d[0])
		}
		g.Z = make([]float64, 0, g.Rows*g.Cols)
		for _, row := range d {
			if len(row) != g.Cols {
				panic("plot: ragged [][]float64")
			}
			g.Z = append(g.Z, row...)
		}
		return g
	case *ndarray.Array:
		sh := d.Shape()
		if len(sh) != 2 {
			panic(fmt.Sprintf("plot: need a 2-D array, got shape %v", sh))
		}
		return gridData{Rows: sh[0], Cols: sh[1], Z: Values(d)}
	}
	panic(fmt.Sprintf("plot: cannot use %T as 2-D data", v))
}

// indices returns 0, 1, ..., n-1.
func indices(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = float64(i)
	}
	return out
}

// mustSameLen panics unless x and y have the same length.
func mustSameLen(what string, x, y []float64) {
	if len(x) != len(y) {
		panic(fmt.Sprintf("plot: %s: x and y must have the same length, got %d and %d", what, len(x), len(y)))
	}
}

// rgbImage takes a (rows, cols, 3 or 4) *ndarray.Array as an image: uint8
// values as they are, anything else as floats in [0, 1].
func rgbImage(v any, p props) (*image.RGBA, bool) {
	d, ok := v.(*ndarray.Array)
	if !ok {
		return nil, false
	}
	sh := d.Shape()
	if len(sh) != 3 || (sh[2] != 3 && sh[2] != 4) {
		return nil, false
	}
	rows, cols, ch := sh[0], sh[1], sh[2]
	img := image.NewRGBA(image.Rect(0, 0, cols, rows))
	var px func(i int) uint8
	if b, ok := ndarray.Data[uint8](d); ok {
		px = func(i int) uint8 { return b[i] }
	} else {
		f := Values(d)
		px = func(i int) uint8 { return uint8(min(1, max(0, f[i]))*255 + 0.5) }
	}
	for k := range rows * cols {
		for c := range ch {
			img.Pix[k*4+c] = px(k*ch + c)
		}
		if ch == 3 {
			img.Pix[k*4+3] = 255
		}
		if p.setAlpha {
			img.Pix[k*4+3] = uint8(float64(img.Pix[k*4+3])*p.alpha + 0.5)
		}
	}
	return img, true
}
