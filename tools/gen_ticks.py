# Generates ticker/testdata/ticks.json: what matplotlib's AutoLocator +
# ScalarFormatter (linear axes) and LogLocator + LogFormatterSciNotation (log
# axes) produce for a set of view intervals, on real axes of real figures, so
# the tick space (nbins) comes from the axis length exactly as in a plot.
#
#   uv run --with matplotlib --with numpy python tools/gen_ticks.py
#
# Run from the repository root.
import json

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402

LINEAR = [
    (0, 1), (0, 10), (-1, 1), (0, 100), (0.5, 7.3), (-3.7, 12.9), (1e-9, 3e-9),
    (0, 1e-6), (0, 1e6), (0, 2.5e7), (1000, 1001), (100000, 100010),
    (123456.7, 123457.2), (-1001, -1000), (-0.05, 1.05), (0.0001, 0.0005),
    (1e5, 1e5 * 1.0001), (-5e8, 5e8), (3.14159, 3.14160), (1e-3, 1.0),
    (0, 0.3), (-2.2, -0.1), (99.5, 100.5), (0.999, 1.002), (-1e-12, 1e-12),
    (5, 5), (0, 0), (7.5e-5, 1.2e-4), (-200, 1800), (1e10, 1e10 + 3e5),
    (-0.3, 0.3), (0.12, 0.13), (2020, 2030), (1.5, 2.25), (-7, 3),
    (-1.0471975511965976, 6.283185307179586), (0.49999, 0.50001),
]
LOG = [
    (1, 10), (1, 1000), (0.1, 100), (1e-3, 1e3), (1, 1e10), (1e-30, 1e30),
    (2, 7), (3, 30), (0.5, 50000), (1.5, 1.5e6), (1e-5, 1e-4), (20, 80),
    (1e2, 1e9), (0.9, 1.1e1),
]
SIZES = [(6.4, 4.8), (3.0, 2.0), (12.0, 2.5), (2.0, 6.0)]


def record(ax, axis_name, scale):
    fig = ax.figure
    fig.canvas.draw()
    axis = ax.xaxis if axis_name == "x" else ax.yaxis
    vmin, vmax = (float(v) for v in axis.get_view_interval())
    ticks = axis._update_ticks()
    major = [t for t in ticks if t in axis.get_major_ticks()]
    minor = [t for t in ticks if t not in major]
    return {
        "axis": axis_name,
        "scale": scale,
        "tick_space": int(axis.get_tick_space()),
        "length_pt": float(
            (ax.bbox.width if axis_name == "x" else ax.bbox.height) * 72 / fig.dpi),
        "vmin": vmin,
        "vmax": vmax,
        "locs": [float(v) for v in axis.get_majorticklocs()],
        "visible": [float(t.get_loc()) for t in major],
        "labels": [t.label1.get_text() for t in major],
        "offset": axis.get_offset_text().get_text(),
        "minor_visible": [float(t.get_loc()) for t in minor],
        "minor_labels": [t.label1.get_text() for t in minor],
    }


def main():
    cases = []
    for size in SIZES:
        for lo, hi in LINEAR:
            for axis_name in ("x", "y"):
                fig, ax = plt.subplots(figsize=size, dpi=100)
                (ax.set_xlim if axis_name == "x" else ax.set_ylim)(lo, hi)
                c = record(ax, axis_name, "linear")
                c["figsize"] = list(size)
                cases.append(c)
                plt.close(fig)
        for lo, hi in LOG:
            for axis_name in ("x", "y"):
                fig, ax = plt.subplots(figsize=size, dpi=100)
                (ax.set_xscale if axis_name == "x" else ax.set_yscale)("log")
                (ax.set_xlim if axis_name == "x" else ax.set_ylim)(lo, hi)
                c = record(ax, axis_name, "log")
                c["figsize"] = list(size)
                cases.append(c)
                plt.close(fig)
    with open("ticker/testdata/ticks.json", "w") as f:
        json.dump({"version": matplotlib.__version__, "cases": cases}, f, indent=0)


if __name__ == "__main__":
    main()
