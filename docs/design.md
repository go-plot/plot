# go-plot v0: design

go-plot is the plotting library of a Go notebook ecosystem built on
[go-ndarray](https://github.com/go-ndarray/ndarray). Its model and its defaults
are matplotlib's, so that a matplotlib user can read and write it without a
manual, and so that a figure can be checked against matplotlib's rendering of
the same figure. This document says how it is built, what is equal to
matplotlib and how that is checked, and what is not.

## Packages

| package | what it does |
|---|---|
| `plot` | `Figure`, `Axes` and every plot type; layout, autoscaling, legend, colorbar |
| `ticker` | tick locations and labels: AutoLocator (MaxNLocator), ScalarFormatter, LogLocator, LogFormatterSciNotation |
| `colors` | colour parsing, the tab10 cycle, eight colormaps, `Normalize` and `LogNorm` |
| `contour` | marching squares: iso-lines and filled bands on a rectangular grid |
| `render` | the drawing seam: a `Renderer` interface, a raster back-end and an SVG back-end |
| `text` | strings to glyph outlines: DejaVu Sans for text, go-tex/math for TeX |

Everything is pure Go (`CGO_ENABLED=0`); it builds for the six 64-bit Linux
targets, 386, arm, `js/wasm` and `wasip1/wasm`.

## API decisions

**Shaped by what notebooks call.** A census of 339 scientific notebooks
(measured for this ecosystem) ranks plotting first among the things that keep
them from moving to Go, and gives the share of notebooks using each call:
`pyplot.subplots` 62 %, `Axes.plot` 56 %, `pyplot.show` 52 %,
`Axes.set_xlabel` 47 %, `Axes.legend` 46 %, `Axes.set_ylabel` 45 %,
`Axes.set_title` 37 %, `pyplot.figure` 27 %, `tight_layout` 27 %,
module-level `pyplot.plot`/`xlabel`/`ylabel` about 22 %, `Axes.set_ylim` 21 %,
`Axes.axhline` 20 %. Most write `fig, ax = plt.subplots(); ax.plot(...)`. So v0
is the object-oriented API, and every one of those operations exists under the
same name: `Subplots`, `Plot`, `SetXLabel`, `Legend` (with matplotlib's
location names and `"best"`), `SetYLabel`, `SetTitle`, `NewFigure`,
`Figure.TightLayout`, `SetYLim`, `Axhline`; `pyplot.show` is the notebook's
call to `Figure.Display`. The module-level, state-machine functions
(`plt.plot`, `plt.xlabel` acting on a hidden "current" Axes) are deliberately
not provided: they are the minority form, and hidden global state is what a
Go notebook kernel running cells concurrently should not have.

**matplotlib's object model, Go's argument style.** `Subplots(rows, cols)`
returns the figure and a `[][]*Axes`; plot types are `Axes` methods named as in
matplotlib (`Plot`, `Scatter`, `Bar`, `Barh`, `Hist`, `FillBetween`, `ErrorBar`,
`Imshow`, `Contour`, `Contourf`, `Axhline`, `Axvline`, `Text`, `Annotate`,
`Legend`, `SetXLim`, `SetYScale`, `TwinX`...). matplotlib's keyword arguments
become functional options — `ax.Plot(x, y, plot.Color("C1"), plot.LineStyle("--"),
plot.Label("data"))` — because Go has no keywords and a struct per method would
multiply types without adding safety. Each method documents the options it
honours; an option a method does not use is ignored, as matplotlib ignores
nothing but also never needs to.

**Data is `any`.** Every data argument accepts `[]float64`, `[]float32`,
`[]int`, `[]int32`, `[]int64`, a scalar, or a `*ndarray.Array` of any of
ndarray's thirteen dtypes (converted with `AsType(Float64)`, flattened in
row-major order; 2-D arrays for `Imshow` and contours, 3-D `(rows, cols, 3|4)`
arrays as RGB(A) images, `uint8` taken as bytes). A type that cannot be plotted
**panics**, as does a length mismatch: both are programming errors, exactly as
an index out of range is, and returning an error from every `Plot` call would
make every notebook cell three lines longer for a failure that is never
handled.

**Mappables.** `Imshow`, `Scatter` with `CValues`, `Contour` and `Contourf`
return a `*Mappable` (colormap + norm), which `Figure.Colorbar(m, ax)` draws,
taking its space from `ax` exactly as `fig.colorbar(m, ax=ax)` does.

**Output.** `SavePNG(w)`, `SaveSVG(w)`, and `Display() map[string][]byte`, the
notebook contract, returning `image/svg+xml` and `image/png`.

**Text is a seam.** `Figure.Text` is a `text.Renderer` (`Layout(s, size)`,
`LineMetrics(size)`); the default sets plain text in DejaVu Sans (matplotlib's
default font, from go-opentype/fonts) and TeX between dollar signs with
[go-tex/math](https://github.com/go-tex/math), a real TeX math typesetter using
the OpenType MATH table of STIX Two Math. Replacing the renderer changes fonts
for the whole figure.

## Reuse of the fleet

- **go-ndarray/ndarray** (pseudo-version of main at the dtypes merge, 3217f28):
  input data, through `ndarray.Data[T]` and `AsType`.
- **go-widgets/painter** v0.15.0: the raster back-end composites through a
  `PixelPainter` — coverage masks with `DrawMask`, images with `DrawImage`
  (nearest-neighbour scaling), clipping with its clip stack.
- **go-gfx/gfx/vector** v0.35.0: anti-aliased coverage of fills and of strokes
  with every cap, join and dash pattern matplotlib uses. The painter's own
  `StrokePath` draws round caps and joins only, which matplotlib's lines do not
  have (projecting caps for solid lines, butt caps for dashes and spines,
  miter joins for markers), so go-plot asks the rasteriser for coverage itself
  and hands the painter a mask.
- **go-opentype/opentype** v0.15.0 and **go-opentype/fonts** v0.12.0: glyph
  outlines, advances and kerning of DejaVu Sans. Text is drawn as outlines, so
  rotated labels and SVG output need no font.
- **go-tex/math** v0.50.0: TeX math to positioned glyph paths (its SVG output
  is parsed back into paths and scaled to the label size).
- **go-widgets/svg** is *not* used: it wraps a raster in an SVG envelope. go-plot
  writes vector SVG itself (paths, clip paths, embedded PNG for images), as
  matplotlib's SVG backend does with `svg.fonttype = "path"`.
- **go-widgets/window** (native and wasm windows) is for a later version.

## What equals matplotlib, and how it is checked

All fixtures are generated by scripts in `tools/` from matplotlib itself
(3.11.2), committed, `//go:embed`-ded in the tests, and compared exactly unless
stated.

| what | how it is checked |
|---|---|
| Tick locations, labels, offset text on linear and log axes | `ticker/testdata/ticks.json`: 408 axes of real figures (4 sizes, 37 linear and 14 log ranges, x and y): every location bit for bit, every label and offset character for character, minor log ticks and their labels included. |
| Locator and formatter parameters a default figure never uses | `ticker/testdata/units.json`: 45 MaxNLocator (nbins, steps, integer, symmetric, prune, min_n_ticks), 42 LogLocator (base 2, e, 10; subs list/auto/all; numticks), 72 ScalarFormatter (math text, no offset, fixed power limits, no scientific, ASCII minus, one or no location), 25 LogFormatterSciNotation, `nonsingular`, and `%g`. Exact, except base-e log locations, compared to 1e-15: NumPy's `e**n` and Go's `math.Pow(e, n)` differ in the last bit. |
| Colormaps | viridis, plasma, inferno, magma, cividis, gray, coolwarm, RdBu (and `_r`): the 256-entry tables are generated from matplotlib, and an independent fixture of 1008 samples per map (including the ends and out-of-range values) is compared as floats and as bytes; the test also checks that all 256 entries were sampled. |
| Named colours and the cycle | all 148 CSS4, 10 `tab:`, 8 single-letter colours, `C0`–`C9`, grey levels, short and long hex: exact. |
| Layout | `testdata/layout.json`: 15 kinds of figure at two sizes — line, bar, barh, hist, imshow, imshow + colorbar, scatter, log-log, fill + errorbar, contourf, 2x2 grid, twinx, offsets, axhline/axvline, inverted and singular limits: every Axes position (after aspect and colorbar) and view limits (after margins and sticky edges) to 1e-9 relative, and every tick label and offset text exactly. |
| Defaults | figure 6.4 x 4.8 in at 100 dpi, subplot parameters, font sizes, tick sizes and pads, line widths, marker sizes, dash patterns (scaled by line width), patch and grid colours, legend geometry, colorbar fraction/pad/aspect: transcribed from matplotlibrc and checked by the rendering comparison below. |
| Histogram binning | NumPy's algorithm (equal bins by arithmetic, corrected against the edges; explicit edges by `searchsorted`). |
| Contour levels | `ContourSet._autolev` (MaxNLocator(n+1, min_n_ticks=1), trimmed). |
| `TightLayout` | matplotlib's `_auto_adjust_subplotpars`, ported; `testdata/layout.json` holds matplotlib's positions after `fig.tight_layout()` for 6 kinds of figure at two sizes (single Axes, labelled, 2x2 labelled, twinx with two labels, offsets, log-log). Every Axes edge is within 0.005 of the figure (worst measured 0.0049, 2.4 pixels): the algorithm is the same, the measured width of text is not — matplotlib's is FreeType's hinted box, every glyph a whole number of pixels; go-plot's is the string's advance. |
| Dense lines | matplotlib's `PathSimplifier` (threshold 1/9 pixel), ported from `path_converters.h`. |

### Rendering comparison

`tools/compare` draws twelve reference figures with go-plot (`main.go`) and
with matplotlib (`figures.py`) — same data, size and dpi — and `compare.py`
measures each pair with **one detector applied to both images**: the axes
frames (columns and rows of long dark runs), the tick marks (dark for the three
pixels just outside each bottom and left spine, within the frame), and the
pixels of the first cycle colour inside the frames. Tolerances: frame and ticks
within 1 pixel, C0 pixel count within 10 %, overlap of C0 pixels (each side
dilated by 1 pixel) at least 0.90.

| figure | frames | ticks (go / mpl) | tick error | C0 pixels (go / mpl) | overlap | mean abs diff (0–255) |
|---|---|---|---|---|---|---|
| line | equal | 15 / 15 | 0 px | 1695 / 1736 | 1.000 | 1.01 |
| scatter + colorbar | equal | 11 / 11 | 0 px | – | – | 0.53 |
| bars (1x2) | equal | 26 / 26 | 0 px | 76024 / 76034 | 1.000 | 0.67 |
| hist | equal | 17 / 17 | 0 px | 49515 / 49515 | 1.000 | 0.63 |
| imshow + colorbar | equal | 16 / 16 | 0 px | 38429 / 38683 | 1.000 | 0.77 |
| contourf + contour + colorbar | equal | 16 / 16 | 0 px | – | – | 0.75 |
| log-log with grid | equal | 11 / 11 | 0 px | 770 / 775 | 1.000 | 0.64 |
| errorbar, fill, axhline/axvline, text, annotate | equal | 14 / 14 | 0 px | 1688 / 1558 | 0.994 | 1.03 |
| twinx | equal | 11 / 11 | 0 px | 881 / 891 | 1.000 | 0.94 |
| 2x2 grid, offsets | equal | 41 / 41 | 0 px | 4991 / 4944 | 0.997 | 1.79 |
| int32 ndarray with markers | equal | 14 / 14 | 0 px | 2380 / 2465 | 1.000 | 0.32 |
| 1000² imshow (perf figure) | equal | 10 / 10 | 0 px | 17426 / 17629 | 1.000 | 1.59 |
| **2x2 labelled, `TightLayout`** | **±2 px** | 14 / 14 | **2 px** | 713 / 700 | 0.970 | 6.71 |
| **1e6-point line (perf figure)** | equal | 13 / 13 | 0 px | **12032 / 15405** | 1.000 | 2.07 |

In twelve of the fourteen, every frame lands on the same pixel rows and
columns and every tick on the same pixel. The images were also looked at, side
by side (`sidebyside.py`, `crop.py`).

Two fail the tolerances, and are reported rather than loosened:

- **`TightLayout`**: frames and ticks within 2 pixels, not 1, for the reason
  given in the table above — the width of tick labels, which tight layout makes
  room for, is measured differently. Side by side the figures are not
  distinguishable by eye.
- **The dense line**: go-plot paints 22 % fewer *saturated* C0
pixels. Every go-plot pixel is within one pixel of a matplotlib one (overlap
1.000); the difference is how dark the edges of a stroke that crosses itself
thousands of times come out — Agg's scanline accumulates such a stroke
differently from go-gfx's coverage sweep. It is not the simplification: drawn
without it, go-plot's count changes by 0.3 %, while matplotlib's own
simplified and unsimplified renderings differ by 9 %.

**Negative controls.** Two deliberately perturbed go-plot figures are compared
with matplotlib's `line`, and both must fail: `line_shifted` (x limits widened
by 0.1 data units at the left: tick error 4 px, overlap 0.436)
and `line_1px` (the view slid by 0.03 data units, 1.35 pixels: tick error 2 px,
overlap 0.779). Both are caught; the smaller one by the tick detector and by the
overlap, so a one-pixel slip of the data does not pass.

## What does not equal matplotlib

- **Text rendering.** Glyphs are filled outlines (go-gfx coverage), not
  FreeType's hinted rasterisation with matplotlib's text antialiasing, so the
  pixels of text differ, and a string's box is its advance where matplotlib's
  is FreeType's hinted box (up to a few pixels wider on long labels); positions are matplotlib's
  (`Text._get_layout`: line metrics from the OS/2 table, alignment, rotation,
  anchoring), and like Agg the pen is put on a whole pixel. Kerning comes from
  DejaVu's GPOS/kern tables through go-opentype.
- **Math text is TeX, not mathtext.** `$...$` is typeset by go-tex/math with
  STIX Two Math — real TeX layout (fractions, radicals, big operators, limits,
  matrices, `\left...\right`), which matplotlib's mathtext only approximates.
  The exception is `\mathdefault{...}`, what matplotlib's tick formatters emit
  (`10^{3}`, `2\times10^{1}`): it is set in DejaVu Sans with mathtext's own
  superscript rules (shift = max(sup1·x-height, depth + x-height/4), scripts at
  0.7 size), so tick labels match matplotlib's look and position.
- **Strokes.** Anti-aliasing is go-gfx's, not Agg's; widths, caps, joins and
  dashes are matplotlib's. Pixel snapping follows matplotlib's rule (odd widths
  at pixel centres, even at edges) for paths made of horizontal and vertical
  segments. Strokes of more than 512 segments are drawn in pieces and get round
  ends (see `docs/perf.md`).
- **Images** are resampled nearest-neighbour when enlarged and box-averaged
  when reduced; matplotlib's default (`interpolation="auto"`, antialiased
  resampling in data space) differs when an image is shown smaller than its
  resolution, or less than three times enlarged.
- **Contours** are marching squares with the saddle resolved by the cell mean,
  not contourpy's `serial` algorithm (`corner_mask`, chunking): levels and
  colours are matplotlib's, line vertices differ slightly. Filled bands are
  drawn per grid cell.
- **Legend `"best"`** counts the vertices, segments, boxes and scatter points
  under each of the nine candidate positions and takes the first with none, else
  the least; matplotlib's badness also weights offsets and is computed in
  display space with a pad. Usually the same place; not always.
- **Annotate** draws `arrowstyle="->"` only, shrunk 2 points at both ends.
- **SVG** carries the same geometry but not matplotlib's document structure,
  ids or metadata. Paint is in presentation attributes, not `style=`, so the
  document can be inlined under a Content-Security-Policy that forbids inline
  styles. Clip ids are a hash of their rectangle, not a counter, so that two
  figures inlined in one page cannot clip each other.
- **Pie** is `ax.pie` with its defaults, checked against matplotlib 3.11.2 by
  `tools/gen_pie.py` → `testdata/pie.json`: wedge paths vertex for vertex
  (Path.arc's Bézier construction), label and percentage positions,
  alignment and rotation, the Axes box and limits, and the refusals. The
  pixel judge needs ≥ 99% of the pixels in the same colour class as matplotlib's
  PNG. A different pie is the negative control. Differences:
  - The data errors are returned, not raised, and a sum that overflows is taken
    after scaling, where matplotlib draws every wedge empty.
  - `AutoPct` takes Go format verbs.
  - There is no `shadow`, `hatch` or `frame=True`.
  - `labeldistance=None` is `LabelDistance(NaN)`.
  - Empty labels make no text object.
  - `PlainText`, `SpreadLabels` and `Donut` are go-plot's own.
- **Text in `\$`**: an escaped dollar outside math is a dollar. v0.1 drew the
  backslash too.

## Not in v0

module-level `pyplot`-style functions (by design, see above); suptitle and
`TightLayout` with explicit pads or `rect`; symlog, logit and date axes; minor ticks on linear axes (off by default in
matplotlib too) and `AutoMinorLocator`; `constrained_layout`;
`subplots` sharing; hatching; boxplot, violin, quiver, stem, step,
`pcolormesh`, 3-D; colorbar `extend` and discrete norms; image interpolation
other than nearest; `Text` bounding boxes; legend columns, titles and
`bbox_to_anchor`; native and browser windows (go-widgets/window); PDF output.
