// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package ticker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
)

// ticks.json is written by tools/gen_ticks.py from matplotlib: for each view
// interval, on real axes of real figures, the locator output, the ticks drawn
// and their labels, and the offset text.
//
//go:embed testdata/ticks.json
var ticksJSON []byte

type tickCase struct {
	Axis         string    `json:"axis"`
	Scale        string    `json:"scale"`
	TickSpace    int       `json:"tick_space"`
	LengthPt     float64   `json:"length_pt"`
	Vmin         float64   `json:"vmin"`
	Vmax         float64   `json:"vmax"`
	Locs         []float64 `json:"locs"`
	Visible      []float64 `json:"visible"`
	Labels       []string  `json:"labels"`
	Offset       string    `json:"offset"`
	MinorVisible []float64 `json:"minor_visible"`
	MinorLabels  []string  `json:"minor_labels"`
	Figsize      []float64 `json:"figsize"`
}

func loadCases(t *testing.T) (string, []tickCase) {
	t.Helper()
	var fx struct {
		Version string     `json:"version"`
		Cases   []tickCase `json:"cases"`
	}
	if err := json.Unmarshal(ticksJSON, &fx); err != nil {
		t.Fatal(err)
	}
	return fx.Version, fx.Cases
}

// TestMatchesMatplotlib checks every case: tick locations bit for bit, the
// ticks drawn, every label and the offset text, character for character.
func TestMatchesMatplotlib(t *testing.T) {
	version, cases := loadCases(t)
	if len(cases) < 300 {
		t.Fatalf("only %d cases", len(cases))
	}
	t.Logf("%d cases from matplotlib %s", len(cases), version)
	for _, c := range cases {
		name := fmt.Sprintf("%s/%s/%gx%g/[%g,%g]", c.Scale, c.Axis, c.Figsize[0], c.Figsize[1], c.Vmin, c.Vmax)
		var cfg Config
		if c.Scale == "log" {
			cfg = LogConfig(c.TickSpace, c.Vmin)
		} else {
			cfg = LinearConfig(c.TickSpace)
		}
		if got := cfg.Major.TickValues(c.Vmin, c.Vmax); !slices.Equal(got, c.Locs) {
			t.Errorf("%s: locs\n got %v\nwant %v", name, got, c.Locs)
			continue
		}
		ticks := cfg.Compute(c.Vmin, c.Vmax)
		var locs []float64
		var labels []string
		for _, tk := range ticks.Major {
			locs = append(locs, tk.Loc)
			labels = append(labels, tk.Label)
		}
		if !slices.Equal(locs, c.Visible) || !slices.Equal(labels, c.Labels) {
			t.Errorf("%s: visible\n got %v %q\nwant %v %q", name, locs, labels, c.Visible, c.Labels)
		}
		if ticks.Offset != c.Offset {
			t.Errorf("%s: offset %q, want %q", name, ticks.Offset, c.Offset)
		}
		locs, labels = nil, nil
		for _, tk := range ticks.Minor {
			locs = append(locs, tk.Loc)
			labels = append(labels, tk.Label)
		}
		if !slices.Equal(locs, c.MinorVisible) || !slices.Equal(labels, c.MinorLabels) {
			t.Errorf("%s: minor\n got %v %q\nwant %v %q", name, locs, labels, c.MinorVisible, c.MinorLabels)
		}
	}
}
