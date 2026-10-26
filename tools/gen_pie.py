# Generates testdata/pie.json and testdata/pie_*.png: what matplotlib's
# Axes.pie builds for a set of pies — the Axes position and limits, every
# wedge's path (vertices and codes, in data coordinates) and face colour,
# every text's position, string, alignment, rotation and window extent — and
# which inputs it refuses. pie_test.go builds the same pies and compares.
#
#   uv run --with matplotlib --with numpy python tools/gen_pie.py
#
# Run from the repository root. matplotlib is the program under test here:
# this file only records what it does.
import json
import math

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
import numpy as np  # noqa: E402


def basic(ax):
    ax.pie([15, 30, 45, 10], labels=["Rent", "Food", "Travel", "Other"],
           autopct="%1.1f%%")


def clockwise(ax):
    ax.pie([15, 30, 45, 10], labels=["a", "b", "c", "d"], startangle=90,
           counterclock=False, explode=[0, 0.1, 0, 0],
           colors=["#4e79a7", "#f28e2b", "#e15759"])


def donut(ax):
    ax.pie([3, 2, 5], labels=["x", "y", "z"], autopct="%.0f%%",
           pctdistance=0.85, wedgeprops=dict(width=0.3, edgecolor="w"))


def single(ax):
    ax.pie([5], labels=["only"], autopct="%1.1f%%")


def zeros(ax):
    ax.pie([0, 3, 0, 1], labels=["z1", "three", "z2", "one"])


def tiny(ax):
    ax.pie([1000, 1, 1, 1, 500],
           labels=["Large", "tiny one", "tiny two", "tiny three", "Medium"])


def rotated(ax):
    ax.pie([1, 2, 3], labels=["p", "q", "r"], rotatelabels=True,
           labeldistance=0.7, center=(2, 3), radius=0.8, startangle=30)


def partial(ax):
    ax.pie([0.2, 0.3], labels=["a", "b"], normalize=False, autopct="%.0f%%")


def legend(ax):
    ax.pie([4, 3, 2, 1], labels=["one", "two", "three", "four"],
           labeldistance=None)
    ax.legend(loc="center left")


def full_donut(ax):
    ax.pie([1], wedgeprops=dict(width=0.4))


CASES = dict(basic=basic, clockwise=clockwise, donut=donut, single=single,
             zeros=zeros, tiny=tiny, rotated=rotated, partial=partial,
             legend=legend, full_donut=full_donut)

# Drawn to PNG as well, for the pixel judge.
PNG = ["basic", "donut", "clockwise"]

REFUSED = dict(
    negative=[1, -1, 2],
    nan=[1, float("nan")],
    inf=[1, float("inf")],
    all_zero=[0, 0],
    empty=[],
)


def colour(c):
    return [float(v) for v in matplotlib.colors.to_rgba(c)]


def main():
    out = {"version": matplotlib.__version__, "cases": {}, "refused": {}}
    for name, mk in CASES.items():
        fig = plt.figure(figsize=(4, 4), dpi=50)
        ax = fig.subplots()
        mk(ax)
        fig.canvas.draw()
        r = fig.canvas.get_renderer()
        H = fig.bbox.height
        pos = ax.get_position()
        wedges = []
        for p in ax.patches:
            path = p.get_path()
            wedges.append({
                "vertices": [[float(x), float(y)] for x, y in path.vertices],
                "codes": [int(c) for c in path.codes],
                "face": colour(p.get_facecolor()),
                "label": p.get_label(),
            })
        texts = []
        for t in ax.texts:
            if t.get_text() == "":
                continue
            e = t.get_window_extent(r)
            x, y = t.get_position()
            texts.append({
                "s": t.get_text(), "x": float(x), "y": float(y),
                "ha": t.get_ha(), "va": t.get_va(),
                "rotation": float(t.get_rotation()),
                "size": float(t.get_fontsize()),
                # pixels, y down from the top of the figure
                "extent": [float(e.x0), float(H - e.y1), float(e.x1), float(H - e.y0)],
            })
        out["cases"][name] = {
            "position": [float(pos.x0), float(pos.y0), float(pos.width), float(pos.height)],
            "xlim": [float(v) for v in ax.get_xlim()],
            "ylim": [float(v) for v in ax.get_ylim()],
            "frame": bool(ax.get_frame_on()),
            "wedges": wedges,
            "texts": texts,
        }
        if name in PNG:
            fig.savefig(f"testdata/pie_{name}.png", dpi=50)
        plt.close(fig)
    for name, x in REFUSED.items():
        fig, ax = plt.subplots()
        try:
            ax.pie(np.asarray(x, dtype=float))
            msg = ""
        except ValueError as e:
            msg = str(e)
        out["refused"][name] = {"x": [v if math.isfinite(v) else str(v) for v in x], "error": msg}
        plt.close(fig)
    with open("testdata/pie.json", "w") as f:
        json.dump(out, f, indent=0)


if __name__ == "__main__":
    main()
