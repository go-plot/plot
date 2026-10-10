# Performance: go-plot v0 against matplotlib

Two figures, each saved as PNG and as SVG, at matplotlib's default size
(6.4 x 4.8 in, 100 dpi):

- **line**: `Plot` of a 1,000,000-point random walk;
- **imshow**: `Imshow` of a 1000 x 1000 array (`sin(i/37)·cos(j/23)`), viridis.

## Method

`tools/compare/perf.py` (driving `tools/compare/perf/main.go`):

- both sides build the same data, with the same generator, **before** the clock
  starts; the clock covers creating the figure, plotting, and writing the file
  to memory;
- every measurement is a **fresh process** on both sides (so neither keeps a warm
  glyph or path cache the other lacks), and the two are **interleaved** case by
  case, run by run, so that load changes on the host hit both alike;
- 9 runs per case; the table gives **medians**, with the spread below.

What is *not* in the numbers: Python's interpreter start and
`import matplotlib.pyplot` (about half a second on this machine), and the Go
process start (a few milliseconds). A notebook keeps both warm, so the in-process
time is the comparable one.

Host: Apple M4 Max, 16 cores, macOS; Go 1.27.2; Python 3.12.12, matplotlib
3.11.2, NumPy 2.5.3, Agg backend. **The host was shared and loaded**: load
average 22.6 before the run and 21.7 after it, on 16 cores. Absolute times are
therefore pessimistic for both; the interleaving is what keeps the ratio fair.
Measured 2026-10-10.

## Results

| case | go-plot median (s) | matplotlib median (s) | matplotlib / go-plot | go-plot bytes | matplotlib bytes |
|---|---|---|---|---|---|
| line, 1e6 points, PNG | 0.055 | 0.065 | 1.2x | 40,600 | 28,055 |
| line, 1e6 points, SVG | 0.019 | 0.053 | 2.8x | 218,801 | 224,082 |
| imshow 1000², PNG | 0.025 | 0.042 | 1.7x | 154,860 | 157,539 |
| imshow 1000², SVG | 0.020 | 0.042 | 2.1x | 207,581 | 198,367 |

Spread (min / max over the 9 runs):

| case | go-plot | matplotlib |
|---|---|---|
| line PNG | 0.052 / 0.085 | 0.061 / 0.107 |
| line SVG | 0.018 / 0.024 | 0.050 / 0.063 |
| imshow PNG | 0.023 / 0.034 | 0.040 / 0.058 |
| imshow SVG | 0.019 / 0.034 | 0.041 / 0.049 |

go-plot's PNGs are larger because it encodes with `png.BestSpeed`; matplotlib
uses zlib's default level.

## Where the time goes, and what changed to get here

The first measurement of the line PNG was **0.124 s, half matplotlib's speed**.
Profiling (`go test -bench -cpuprofile`) showed three costs, each now fixed:

1. the data limits of the million points were recomputed per axis per draw
   (0.67 s of 2.2 s sampled): they are now taken once, when the artist is added;
2. the coverage sweep of go-gfx's rasteriser sorts each scanline's crossings by
   insertion, quadratic in the crossings, and a dense polyline has thousands per
   scanline: strokes of more than 512 segments are now rasterised in 512-segment
   pieces (matplotlib has the same knob, `agg.path.chunksize`, off by default),
   whose seams are round joins, so the only visible change is that the line's
   own two ends are round rather than projecting;
3. STIX Two Math (for TeX labels) was parsed on every figure: it is now parsed
   only when a label contains TeX.

Like matplotlib, go-plot simplifies dense lines before drawing them
(matplotlib's `PathSimplifier`, ported): the 1e6-point walk is drawn as 11,738
vertices. That is why the SVG of a million points is about 220 kB on both sides.

## Reproduce

```sh
cd tools/compare
uv run --with matplotlib --with numpy python perf.py 9
```
