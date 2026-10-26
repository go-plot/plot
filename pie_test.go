// Copyright (c) 2026 the go-plot/plot authors. All rights reserved.
// Use of this source code is governed by a BSD-3-Clause license that can be
// found in the LICENSE file at the root of this repository.

package plot

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"math"
	"strings"
	"testing"

	"github.com/go-plot/plot/colors"
)

// pie.json and pie_*.png are written by tools/gen_pie.py: what matplotlib's
// ax.pie builds for each case below, and which inputs it refuses.
var (
	//go:embed testdata/pie.json
	pieJSON []byte
	//go:embed testdata/pie_basic.png
	pieBasicPNG []byte
	//go:embed testdata/pie_donut.png
	pieDonutPNG []byte
	//go:embed testdata/pie_clockwise.png
	pieClockwisePNG []byte
)

// pieCases builds the same pies as tools/gen_pie.py.
var pieCases = map[string]func(a *Axes) error{
	"basic": func(a *Axes) error {
		return a.Pie([]float64{15, 30, 45, 10}, Labels("Rent", "Food", "Travel", "Other"), AutoPct("%1.1f%%"))
	},
	"clockwise": func(a *Axes) error {
		return a.Pie([]float64{15, 30, 45, 10}, Labels("a", "b", "c", "d"), StartAngle(90),
			CounterClock(false), Explode(0, 0.1, 0, 0), Colors("#4e79a7", "#f28e2b", "#e15759"))
	},
	"donut": func(a *Axes) error {
		return a.Pie([]int{3, 2, 5}, Labels("x", "y", "z"), AutoPct("%.0f%%"), PctDistance(0.85), WedgeWidth(0.3), EdgeColor("w"))
	},
	"single": func(a *Axes) error { return a.Pie([]float64{5}, Labels("only"), AutoPct("%1.1f%%")) },
	"zeros":  func(a *Axes) error { return a.Pie([]float64{0, 3, 0, 1}, Labels("z1", "three", "z2", "one")) },
	"tiny": func(a *Axes) error {
		return a.Pie([]float64{1000, 1, 1, 1, 500}, Labels("Large", "tiny one", "tiny two", "tiny three", "Medium"))
	},
	"rotated": func(a *Axes) error {
		return a.Pie([]float64{1, 2, 3}, Labels("p", "q", "r"), RotateLabels(), LabelDistance(0.7),
			Center(2, 3), Radius(0.8), StartAngle(30))
	},
	"partial": func(a *Axes) error {
		return a.Pie([]float64{0.2, 0.3}, Labels("a", "b"), Normalize(false), AutoPct("%.0f%%"))
	},
	"legend": func(a *Axes) error {
		err := a.Pie([]float64{4, 3, 2, 1}, Labels("one", "two", "three", "four"), LabelDistance(math.NaN()))
		a.Legend(Loc("center left"))
		return err
	},
	"full_donut": func(a *Axes) error { return a.Pie([]float64{1}, WedgeWidth(0.4)) },
}

type pieFixture struct {
	Version string `json:"version"`
	Cases   map[string]struct {
		Position []float64 `json:"position"`
		Xlim     []float64 `json:"xlim"`
		Ylim     []float64 `json:"ylim"`
		Frame    bool      `json:"frame"`
		Wedges   []struct {
			Vertices [][2]float64 `json:"vertices"`
			Codes    []uint8      `json:"codes"`
			Face     [4]float64   `json:"face"`
			Label    string       `json:"label"`
		} `json:"wedges"`
		Texts []struct {
			S        string     `json:"s"`
			X        float64    `json:"x"`
			Y        float64    `json:"y"`
			HA       string     `json:"ha"`
			VA       string     `json:"va"`
			Rotation float64    `json:"rotation"`
			Size     float64    `json:"size"`
			Extent   [4]float64 `json:"extent"`
		} `json:"texts"`
	} `json:"cases"`
	Refused map[string]struct {
		X     []any  `json:"x"`
		Error string `json:"error"`
	} `json:"refused"`
}

func loadPieFixture(t *testing.T) pieFixture {
	t.Helper()
	var fx pieFixture
	if err := json.Unmarshal(pieJSON, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) != len(pieCases) {
		t.Fatalf("%d fixture cases for %d builders", len(fx.Cases), len(pieCases))
	}
	return fx
}

func newPie(t *testing.T, name string) (*Figure, *Axes) {
	t.Helper()
	f := NewFigure(FigSize(4, 4), DPI(50))
	a := f.Subplots(1, 1)[0][0]
	if err := pieCases[name](a); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return f, a
}

// pieTexts returns the texts of an Axes in the order they were added.
func pieTexts(a *Axes) []*textArt {
	var out []*textArt
	for _, art := range a.artists {
		switch v := art.(type) {
		case *textArt:
			out = append(out, v)
		case *pieLabels:
			out = append(out, v.texts...)
		}
	}
	return out
}

// TestPieMatchesMatplotlib compares, case by case, the Axes box and limits
// (1e-9 relative), every wedge's path vertex by vertex (1e-12) with its
// codes and colour, and every text's string, position, alignment, rotation
// and size exactly; its window extent within 1.5 pixels (the text box is
// measured on outlines here, on hinted glyphs in matplotlib).
func TestPieMatchesMatplotlib(t *testing.T) {
	fx := loadPieFixture(t)
	for name := range pieCases {
		want := fx.Cases[name]
		f, a := newPie(t, name)
		f.layout()
		pos := []float64{a.active.x, a.active.y, a.active.w, a.active.h}
		for k := range pos {
			if !closeRel(pos[k], want.Position[k]) {
				t.Errorf("%s: position %v, want %v", name, pos, want.Position)
				break
			}
		}
		x0, x1 := a.XLim()
		y0, y1 := a.YLim()
		if !closeRel(x0, want.Xlim[0]) || !closeRel(x1, want.Xlim[1]) || !closeRel(y0, want.Ylim[0]) || !closeRel(y1, want.Ylim[1]) {
			t.Errorf("%s: lims %v %v %v %v, want %v %v", name, x0, x1, y0, y1, want.Xlim, want.Ylim)
		}
		if want.Frame || !a.noFrame {
			t.Errorf("%s: frame on", name)
		}
		var wedges []*wedge
		for _, art := range a.artists {
			if w, ok := art.(*wedge); ok {
				wedges = append(wedges, w)
			}
		}
		if len(wedges) != len(want.Wedges) {
			t.Fatalf("%s: %d wedges, want %d", name, len(wedges), len(want.Wedges))
		}
		for i, w := range wedges {
			ww := want.Wedges[i]
			v, c := w.vertices()
			if len(v) != len(ww.Vertices) || string(c) != string(ww.Codes) {
				t.Errorf("%s wedge %d: codes %v, want %v", name, i, c, ww.Codes)
				continue
			}
			for k := range v {
				if math.Abs(v[k].X-ww.Vertices[k][0]) > 1e-12 || math.Abs(v[k].Y-ww.Vertices[k][1]) > 1e-12 {
					t.Errorf("%s wedge %d vertex %d: %v, want %v", name, i, k, v[k], ww.Vertices[k])
					break
				}
			}
			face := w.style.face
			if !closeRel(face.R, ww.Face[0]) || !closeRel(face.G, ww.Face[1]) || !closeRel(face.B, ww.Face[2]) || !closeRel(face.A, ww.Face[3]) {
				t.Errorf("%s wedge %d: face %v, want %v", name, i, face, ww.Face)
			}
			if w.label != ww.Label {
				t.Errorf("%s wedge %d: label %q, want %q", name, i, w.label, ww.Label)
			}
		}
		texts := pieTexts(a)
		if len(texts) != len(want.Texts) {
			t.Fatalf("%s: %d texts, want %d", name, len(texts), len(want.Texts))
		}
		for i, tx := range texts {
			wt := want.Texts[i]
			rot := math.Mod(tx.rot, 360)
			if rot < 0 {
				rot += 360
			}
			if tx.s != wt.S || tx.ha != wt.HA || tx.va != wt.VA || tx.size != wt.Size ||
				math.Abs(tx.x-wt.X) > 1e-12 || math.Abs(tx.y-wt.Y) > 1e-12 || math.Abs(rot-wt.Rotation) > 1e-9 {
				t.Errorf("%s text %d: %q at %v %v %s/%s rot %v size %v, want %q at %v %v %s/%s rot %v size %v", name, i,
					tx.s, tx.x, tx.y, tx.ha, tx.va, rot, tx.size, wt.S, wt.X, wt.Y, wt.HA, wt.VA, wt.Rotation, wt.Size)
			}
			b := f.layoutText(tx.spec(a)).bbox
			got := [4]float64{b.x0, b.y0, b.x1, b.y1}
			for k := range got {
				if math.Abs(got[k]-wt.Extent[k]) > 1.5 {
					t.Errorf("%s text %q: extent %v, want %v", name, tx.s, got, wt.Extent)
					break
				}
			}
		}
	}
}

// TestPieRefusesWhatMatplotlibRefuses: every input matplotlib refuses
// returns the matching error and adds nothing to the Axes.
func TestPieRefusesWhatMatplotlibRefuses(t *testing.T) {
	fx := loadPieFixture(t)
	want := map[string]error{
		"Wedge sizes 'x' must be non negative values": ErrPieNegative,
		"Wedge sizes must be finite numbers":          ErrPieNonFinite,
		"All wedge sizes are zero":                    ErrPieZero,
	}
	if len(fx.Refused) < 5 {
		t.Fatalf("%d refused cases", len(fx.Refused))
	}
	for name, c := range fx.Refused {
		xs := make([]float64, len(c.X))
		for i, v := range c.X {
			switch v := v.(type) {
			case float64:
				xs[i] = v
			case string:
				xs[i] = map[string]float64{"nan": math.NaN(), "inf": math.Inf(1)}[v]
			}
		}
		werr, ok := want[c.Error]
		if !ok {
			t.Fatalf("%s: matplotlib said %q, which this test does not know", name, c.Error)
		}
		a := NewFigure().Subplots(1, 1)[0][0]
		if err := a.Pie(xs); !errors.Is(err, werr) {
			t.Errorf("%s %v: %v, want %v", name, xs, err, werr)
		}
		if len(a.artists) != 0 || a.noFrame {
			t.Errorf("%s: a refused pie changed the Axes", name)
		}
	}
}

// classify returns the index of the palette colour nearest to c.
func classify(c [3]float64, palette [][3]float64) int {
	best, bi := math.Inf(1), 0
	for i, p := range palette {
		d := (c[0]-p[0])*(c[0]-p[0]) + (c[1]-p[1])*(c[1]-p[1]) + (c[2]-p[2])*(c[2]-p[2])
		if d < best {
			best, bi = d, i
		}
	}
	return bi
}

func rgbAt(img image.Image, x, y int) [3]float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	return [3]float64{float64(r) / 0xffff, float64(g) / 0xffff, float64(b) / 0xffff}
}

// pixelAgreement draws a case and matplotlib's PNG of it, puts every pixel
// in the class of its nearest colour (white, black, each wedge's face) and
// returns the fraction of pixels in the same class in both.
func pixelAgreement(t *testing.T, name string, ref []byte) float64 {
	t.Helper()
	want, err := png.Decode(bytes.NewReader(ref))
	if err != nil {
		t.Fatal(err)
	}
	f, a := newPie(t, name)
	got := f.Image().Img
	if got.Bounds() != want.Bounds() {
		t.Fatalf("%s: %v, matplotlib %v", name, got.Bounds(), want.Bounds())
	}
	palette := [][3]float64{{1, 1, 1}, {0, 0, 0}}
	for _, art := range a.artists {
		if w, ok := art.(*wedge); ok {
			palette = append(palette, [3]float64{w.style.face.R, w.style.face.G, w.style.face.B})
		}
	}
	same, n := 0, 0
	b := got.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			n++
			if classify(rgbAt(got, x, y), palette) == classify(rgbAt(want, x, y), palette) {
				same++
			}
		}
	}
	return float64(same) / float64(n)
}

// TestPiePixelsMatchMatplotlib is the rendering judge: at least 99% of the
// pixels are the same colour class as in matplotlib's own PNG. The
// remainder is anti-aliasing of edges and text. Its control: the same
// comparison against a different case must fail, or the judge sees nothing.
func TestPiePixelsMatchMatplotlib(t *testing.T) {
	for _, c := range []struct {
		name string
		ref  []byte
	}{{"basic", pieBasicPNG}, {"donut", pieDonutPNG}, {"clockwise", pieClockwisePNG}} {
		if got := pixelAgreement(t, c.name, c.ref); got < 0.99 {
			t.Errorf("%s: %.4f of the pixels agree with matplotlib, want >= 0.99", c.name, got)
		} else {
			t.Logf("%s: %.4f of the pixels agree with matplotlib", c.name, got)
		}
	}
	if got := pixelAgreement(t, "basic", pieClockwisePNG); got > 0.9 {
		t.Errorf("control: a different pie agrees %.4f, the judge cannot tell pies apart", got)
	}
}

// TestPieSVGIsSafeToInline: labels that look like markup, script or TeX
// come out as glyph outlines; the document has no script, no event handler,
// no style attribute and no external reference, so it can be inlined in a
// page served under a strict Content-Security-Policy.
func TestPieSVGIsSafeToInline(t *testing.T) {
	f := NewFigure()
	a := f.Subplots(1, 1)[0][0]
	evil := []string{`</svg><script>alert(1)</script>`, `" onload="alert(1)`, `<img src=x onerror=alert(1)>`, `$\input{/etc/passwd}$ & $5`}
	if err := a.Pie([]float64{1, 2, 3, 4}, Labels(evil...), AutoPct("%.1f%%"), PlainText()); err != nil {
		t.Fatal(err)
	}
	a.Legend()
	a.SetTitle("<b>title</b>")
	var buf bytes.Buffer
	if err := f.SaveSVG(&buf); err != nil {
		t.Fatal(err)
	}
	doc := strings.ToLower(buf.String())
	for _, bad := range []string{"<script", "onload", "onerror", "alert", "style=", "href", "<img", "passwd", "<b>", "<text", "<foreignobject", "javascript:"} {
		if strings.Contains(doc, bad) {
			t.Errorf("the SVG contains %q", bad)
		}
	}
	for i, s := range strings.Split(doc, "url(") {
		if i > 0 && !strings.HasPrefix(s, "#") {
			t.Errorf("an external url(%.20s", s)
		}
	}
}

// TestPieEdgeCases covers what the fixture does not: overflow, option
// errors, Donut, Colors with Alpha, the toolkit's orientation and the
// TeX-free labels.
func TestPieEdgeCases(t *testing.T) {
	ax := func() *Axes { return NewFigure().Subplots(1, 1)[0][0] }
	// A sum that overflows still gives the right fractions.
	a := ax()
	if err := a.Pie([]float64{math.MaxFloat64, math.MaxFloat64}); err != nil {
		t.Fatal(err)
	}
	if w := a.artists[0].(*wedge); math.Abs(w.theta2-w.theta1-180) > 1e-9 {
		t.Errorf("overflowing sum: wedge of %v degrees", w.theta2-w.theta1)
	}
	for name, err := range map[string]error{
		"labels":       ax().Pie([]float64{1, 2}, Labels("a")),
		"explode":      ax().Pie([]float64{1, 2}, Explode(1)),
		"radius":       ax().Pie([]float64{1}, Radius(0)),
		"radius inf":   ax().Pie([]float64{1}, Radius(math.Inf(1))),
		"start":        ax().Pie([]float64{1}, StartAngle(math.NaN())),
		"center":       ax().Pie([]float64{1}, Center(math.Inf(1), 0)),
		"explode nan":  ax().Pie([]float64{1}, Explode(math.NaN())),
		"width":        ax().Pie([]float64{1}, WedgeWidth(2)),
		"width neg":    ax().Pie([]float64{1}, WedgeWidth(-1)),
		"width nan":    ax().Pie([]float64{1}, WedgeWidth(math.NaN())),
		"unnormalized": ax().Pie([]float64{0.7, 0.7}, Normalize(false)),
		"donut":        ax().Donut([]float64{1}, 0),
		"-inf":         ax().Pie([]float64{1, math.Inf(-1)}),
	} {
		if err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	if !errors.Is(ax().Pie([]float64{1, math.Inf(-1)}), ErrPieNegative) {
		t.Error("-inf is negative first, as in matplotlib")
	}
	// Donut puts the percentages in the middle of the ring.
	a = ax()
	if err := a.Donut([]float64{1, 3}, 0.4, AutoPct("%.0f%%"), Radius(2)); err != nil {
		t.Fatal(err)
	}
	ts := pieTexts(a)
	if len(ts) != 2 || math.Abs(math.Hypot(ts[0].x, ts[0].y)-1.8) > 1e-12 || ts[0].s != "25%" {
		t.Errorf("donut percentages: %+v", ts)
	}
	// Colors cycle, and Alpha applies to them.
	a = ax()
	if err := a.Pie([]float64{1, 1, 1}, Colors("r", "g"), Alpha(0.5)); err != nil {
		t.Fatal(err)
	}
	if w := a.artists[2].(*wedge); w.style.face != (colors.RGBA{R: 1, A: 0.5}) {
		t.Errorf("third wedge %v", w.style.face)
	}
	// One colour for every wedge.
	a = ax()
	_ = a.Pie([]float64{1, 1}, Color("k"), ZOrder(4))
	if w := a.artists[1].(*wedge); w.style.face != (colors.RGBA{A: 1}) || w.z != 4 {
		t.Errorf("Color: %v z %v", w.style.face, w.z)
	}
	// The toolkit's layout: from 12 o'clock, clockwise.
	a = ax()
	_ = a.Pie([]float64{1, 3}, StartAngle(90), CounterClock(false))
	if w := a.artists[0].(*wedge); w.theta1 != 0 || w.theta2 != 90 {
		t.Errorf("first wedge %v..%v, want 0..90 (12 to 3 o'clock)", w.theta1, w.theta2)
	}
	// AutoPctFunc, "" skipped; FontSize and TextColor reach the texts.
	a = ax()
	_ = a.Pie([]float64{99, 1}, AutoPctFunc(func(p float64) string {
		if p < 5 {
			return ""
		}
		return "big"
	}), FontSize(7), TextColor("w"), LabelDistance(math.NaN()))
	if ts := pieTexts(a); len(ts) != 1 || ts[0].size != 7 || ts[0].color != (colors.RGBA{R: 1, G: 1, B: 1, A: 1}) {
		t.Errorf("AutoPctFunc texts %+v", ts)
	}
	// Legend entries skip "_" labels; PlainText reaches them.
	a = ax()
	_ = a.Pie([]float64{1, 1}, Labels("_skip", "$5 and $6"), PlainText())
	if e := a.artists[0].legendEntry(); e != nil {
		t.Error("an _ label in the legend")
	}
	if e := a.artists[1].legendEntry(); e == nil || !e.plain {
		t.Errorf("legend entry %+v", e)
	}
	if ts := pieTexts(a); !ts[1].plain {
		t.Error("PlainText did not reach the label")
	}
	// Rotated labels left of x = 0 turn by another half turn, to read
	// left to right, as in matplotlib.
	a = ax()
	_ = a.Pie([]float64{1, 1}, Labels("right", "left"), RotateLabels())
	if ts := pieTexts(a); math.Abs(ts[0].rot-90) > 1e-9 || math.Abs(ts[1].rot-450) > 1e-9 || ts[1].va != "top" || ts[0].va != "bottom" {
		t.Errorf("rotated labels %v %s, %v %s", ts[0].rot, ts[0].va, ts[1].rot, ts[1].va)
	}
	a = ax()
	_ = a.Pie([]float64{1, 1}, Labels("_skip", "$5 and $6"), PlainText())
	// A wedge's limits are its path's.
	if l := a.artists[0].limits(); l.x0 < -1-1e-12 || l.x1 > 1+1e-12 {
		t.Errorf("limits %+v", l)
	}
	if x, y := a.artists[0].sticky(); x != nil || y != nil {
		t.Error("sticky")
	}
}

// FuzzPie: any sizes and labels either return an error or draw a
// well-formed figure with no script in it; nothing panics.
func FuzzPie(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0xf0, 0x3f, 0, 0, 0, 0, 0, 0, 0, 0x40}, "a\x00b", 90.0, 0.3, true)
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0xf8, 0x7f}, "$x$", 0.0, 0.0, false)
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xef, 0x7f, 1, 0, 0, 0, 0, 0, 0, 0}, "<script>", -720.0, 1.0, true)
	f.Fuzz(func(t *testing.T, raw []byte, label string, start, width float64, spread bool) {
		if len(raw) > 8*64 {
			raw = raw[:8*64]
		}
		var xs []float64
		var labels []string
		for i := 0; i+8 <= len(raw); i += 8 {
			xs = append(xs, math.Float64frombits(uint64(raw[i])|uint64(raw[i+1])<<8|uint64(raw[i+2])<<16|uint64(raw[i+3])<<24|
				uint64(raw[i+4])<<32|uint64(raw[i+5])<<40|uint64(raw[i+6])<<48|uint64(raw[i+7])<<56))
			labels = append(labels, label)
		}
		fig := NewFigure(FigSize(3, 3), DPI(30))
		a := fig.Subplots(1, 1)[0][0]
		opts := []Option{Labels(labels...), AutoPct("%.1f%%"), StartAngle(start), WedgeWidth(width), PlainText()}
		if spread {
			opts = append(opts, SpreadLabels())
		}
		if err := a.Pie(xs, opts...); err != nil {
			return
		}
		a.Legend()
		var buf bytes.Buffer
		if err := fig.SaveSVG(&buf); err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(buf.Bytes(), []byte("<script")) || !bytes.HasSuffix(buf.Bytes(), []byte("</svg>\n")) {
			t.Fatal("bad document")
		}
	})
}

// TestPieSpreadLabels: thin wedges side by side have overlapping labels
// where matplotlib puts them (the fixture's "tiny" case); SpreadLabels moves
// them apart, joins the moved ones to their wedges and leaves the rest
// where they were.
func TestPieSpreadLabels(t *testing.T) {
	overlaps := func(f *Figure, a *Axes, specs []textSpec) int {
		n := 0
		for i := range specs {
			for j := i + 1; j < len(specs); j++ {
				p, q := f.layoutText(specs[i]).bbox, f.layoutText(specs[j]).bbox
				if p.x0 < q.x1 && q.x0 < p.x1 && p.y0 < q.y1 && q.y0 < p.y1 {
					n++
				}
			}
		}
		return n
	}
	values := []float64{1000, 1, 1, 1, 500}
	labels := Labels("Large", "tiny one", "tiny two", "tiny three", "Medium")

	f, a := newPie(t, "tiny")
	f.layout()
	var plain []textSpec
	for _, tx := range pieTexts(a) {
		plain = append(plain, tx.spec(a))
	}
	if overlaps(f, a, plain) == 0 {
		t.Fatal("control: the tiny labels do not overlap without SpreadLabels; the test proves nothing")
	}

	f = NewFigure(FigSize(4, 4), DPI(50))
	a = f.Subplots(1, 1)[0][0]
	if err := a.Pie(values, labels, SpreadLabels()); err != nil {
		t.Fatal(err)
	}
	f.layout()
	g := a.artists[len(a.artists)-1].(*pieLabels)
	if g.zorder() != 3 || g.legendEntry() != nil {
		t.Error("label group")
	}
	if x, y := g.sticky(); x != nil || y != nil || g.limits() != emptyLim() {
		t.Error("label group limits")
	}
	specs, moved := g.place(a)
	if overlaps(f, a, specs) != 0 {
		t.Errorf("labels still overlap after SpreadLabels")
	}
	nMoved := 0
	for i, m := range moved {
		if m {
			nMoved++
		} else if specs[i] != plain[i] {
			t.Errorf("label %q moved without being marked", specs[i].S)
		}
	}
	if nMoved == 0 || nMoved == len(moved) {
		t.Errorf("%d of %d labels moved", nMoved, len(moved))
	}
	if len(specs) != len(values) {
		t.Errorf("%d texts placed", len(specs))
	}
	// The legend's "best" sees the group.
	a.Legend()
	var buf bytes.Buffer
	if err := f.SaveSVG(&buf); err != nil {
		t.Fatal(err)
	}
	png, svg := renderBoth(t, f)
	dumpFigure(t, "pie_spread", png.Bytes(), svg)
}
