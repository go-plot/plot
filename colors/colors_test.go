// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package colors

import (
	_ "embed"
	"encoding/json"
	"math"
	"slices"
	"testing"
)

// colormaps.json is written by tools/gen_colormaps.py from matplotlib,
// independently of the lookup tables: every colormap called on 1008 values
// (1001 evenly spaced in [0, 1], the ends, out-of-range values), as floats and
// as bytes, plus the property cycle and every named colour.
//
//go:embed testdata/colormaps.json
var fixtureJSON []byte

type fixture struct {
	Version string    `json:"version"`
	X       []float64 `json:"x"`
	Maps    map[string]struct {
		RGBA  [][4]float64 `json:"rgba"`
		Bytes [][4]int     `json:"bytes"`
	} `json:"maps"`
	Cycle []string              `json:"cycle"`
	Named map[string][4]float64 `json:"named"`
}

func load(t *testing.T) fixture {
	t.Helper()
	var f fixture
	if err := json.Unmarshal(fixtureJSON, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// TestColormapsMatchMatplotlib checks every sample of every colormap for
// exact equality, floats and bytes, and that every one of the 256 table
// entries was reached by some sample.
func TestColormapsMatchMatplotlib(t *testing.T) {
	f := load(t)
	want := []string{"RdBu", "cividis", "coolwarm", "gray", "inferno", "magma", "plasma", "viridis"}
	if got := Names(); !slices.Equal(got, want) {
		t.Fatalf("Names() = %v", got)
	}
	if len(f.Maps) != len(want) {
		t.Fatalf("fixture has %d maps", len(f.Maps))
	}
	for name, m := range f.Maps {
		cm := MustGet(name)
		seen := map[int]bool{}
		for i, x := range f.X {
			c := cm.At(x)
			if got := [4]float64{c.R, c.G, c.B, c.A}; got != m.RGBA[i] {
				t.Errorf("%s(%v) = %v, want %v", name, x, got, m.RGBA[i])
			}
			b := cm.BytesAt(x)
			if got := [4]int{int(b[0]), int(b[1]), int(b[2]), int(b[3])}; got != m.Bytes[i] {
				t.Errorf("%s(%v, bytes) = %v, want %v", name, x, got, m.Bytes[i])
			}
			seen[Index(x)] = true
		}
		for i := range 256 {
			if !seen[i] {
				t.Errorf("%s: entry %d never checked", name, i)
			}
		}
	}
	t.Logf("8 colormaps x %d samples equal to matplotlib %s", len(f.X), f.Version)
}

func TestReversedAndBad(t *testing.T) {
	v, r := MustGet("viridis"), MustGet("viridis_r")
	for _, x := range []float64{0, 0.25, 0.5, 0.999} {
		if v.At(x) != r.At(1-x-1e-12) && v.At(x) != r.At(1-x) {
			t.Errorf("viridis_r(%v) mismatch", x)
		}
	}
	if r.Under != v.Over || r.Over != v.Under {
		t.Error("reversed under/over")
	}
	if (v.At(math.NaN()) != RGBA{}) {
		t.Error("NaN is not transparent")
	}
	if _, err := Get("jet"); err == nil {
		t.Error("jet should not exist")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustGet did not panic")
		}
	}()
	MustGet("nope")
}

func TestNamedMatchMatplotlib(t *testing.T) {
	f := load(t)
	if !slices.Equal(f.Cycle, Cycle) {
		t.Errorf("cycle %v, want %v", Cycle, f.Cycle)
	}
	for name, want := range f.Named {
		c, err := Parse(name)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got := [4]float64{c.R, c.G, c.B, c.A}; got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	if len(f.Named) < 150 {
		t.Errorf("only %d names", len(f.Named))
	}
}

func TestParseErrorsAndHelpers(t *testing.T) {
	for _, bad := range []string{"#12", "#ggg", "1.5", "notacolor", "#1234567"} {
		if _, err := Parse(bad); err == nil {
			t.Errorf("Parse(%q) accepted", bad)
		}
	}
	if CycleColor(10) != CycleColor(0) || CycleColor(-1) != CycleColor(9) {
		t.Error("cycle wrap")
	}
	c := MustParse("#ff800080")
	if c.Hex() != "#ff800080" || c.WithAlpha(1).Hex() != "#ff8000" {
		t.Errorf("hex %s", c.Hex())
	}
	if (RGBA{2, -1, 0.5, 1}).Bytes() != [4]uint8{255, 0, 128, 255} {
		t.Error("clamp")
	}
	defer func() {
		if recover() == nil {
			t.Error("MustParse did not panic")
		}
	}()
	MustParse("bogus")
}

func TestNorms(t *testing.T) {
	n := Normalize{2, 4}
	if n.Normalize(3) != 0.5 || (Normalize{1, 1}).Normalize(5) != 0 {
		t.Error("Normalize")
	}
	if lo, hi := n.Range(); lo != 2 || hi != 4 {
		t.Error("Range")
	}
	l := LogNorm{1, 100}
	if l.Normalize(10) != 0.5 || !math.IsNaN(l.Normalize(0)) || (LogNorm{3, 3}).Normalize(3) != 0 {
		t.Error("LogNorm")
	}
	if lo, hi := l.Range(); lo != 1 || hi != 100 {
		t.Error("LogNorm Range")
	}
}
