# Times matplotlib and go-plot on the same two figures, interleaved: a line
# plot of a million points and a 1000 x 1000 imshow, each to PNG and SVG.
# Each run is a fresh process for both (so neither keeps a warm cache the
# other lacks), the data are built before the clock starts, and the clock
# covers building the figure and writing the file to memory. Reports medians.
#
#   uv run --with matplotlib --with numpy python perf.py [runs]
import io
import json
import statistics
import subprocess
import sys

RUNS = int(sys.argv[1]) if len(sys.argv) > 1 else 7
CASES = ["line_png", "line_svg", "imshow_png", "imshow_svg"]

MPL = r'''
import io, json, sys, time
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np
case = sys.argv[1]
n = 1_000_000
s = 3
y = np.empty(n)
# the same LCG walk as perf/main.go, vectorised would differ in rounding
# order, so it is built the same way, element by element
acc = 0.0
for i in range(n):
    s = (1664525 * s + 1013904223) % 4294967296
    acc += s / 4294967296.0 - 0.5
    y[i] = acc
x = np.arange(n, dtype=float)
i, j = np.mgrid[0:1000, 0:1000]
img = np.sin(i / 37) * np.cos(j / 23)
buf = io.BytesIO()
t = time.perf_counter()
fig, ax = plt.subplots()
if case.startswith("line"):
    ax.plot(x, y)
else:
    ax.imshow(img)
fig.savefig(buf, format=case.split("_")[1])
el = time.perf_counter() - t
if len(sys.argv) > 2:
    open(sys.argv[2], "wb").write(buf.getvalue())
print(json.dumps({"case": case, "seconds": el, "bytes": len(buf.getvalue())}))
'''


def run(cmd):
    out = subprocess.run(cmd, capture_output=True, text=True, check=True).stdout
    return json.loads(out.strip().splitlines()[-1])


def main():
    subprocess.run(["go", "build", "-o", "/tmp/goplot-perf", "./perf"], check=True,
                   env={**__import__("os").environ, "GOWORK": "off"})
    res = {c: {"go": [], "mpl": []} for c in CASES}
    size = {}
    for r in range(RUNS):
        for c in CASES:
            g = run(["/tmp/goplot-perf", "-case", c])
            m = run([sys.executable, "-c", MPL, c])
            res[c]["go"].append(g["seconds"])
            res[c]["mpl"].append(m["seconds"])
            size[c] = (g["bytes"], m["bytes"])
        print(f"run {r + 1}/{RUNS} done", file=sys.stderr)
    print("| case | go-plot median (s) | matplotlib median (s) | speed-up | go-plot bytes | matplotlib bytes |")
    print("|---|---|---|---|---|---|")
    for c in CASES:
        g, m = statistics.median(res[c]["go"]), statistics.median(res[c]["mpl"])
        print(f"| {c} | {g:.3f} | {m:.3f} | {m / g:.1f}x | {size[c][0]:,} | {size[c][1]:,} |")
    print(json.dumps(res), file=sys.stderr)


if __name__ == "__main__":
    main()
