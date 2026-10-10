# Generates testdata/layout.json: for a set of small figures, what matplotlib
# decides about each Axes once drawn — its position (after aspect and
# colorbars), its view limits (after autoscaling, margins and sticky edges),
# its tick labels and offset text. layout_test.go builds the same figures.
#
#   uv run --with matplotlib --with numpy python tools/gen_layout.py
#
# Run from the repository root.
import json

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
import numpy as np  # noqa: E402

X = [0.0, 0.5, 1.3, 2.0, 3.7, 5.0]
Y = [1.0, -2.0, 0.5, 4.0, 3.0, 2.5]
Z = [[float((i * 7 + j * 3) % 11) for j in range(6)] for i in range(4)]


def line(fig):
    ax = fig.subplots()
    ax.plot(X, Y)


def bar(fig):
    ax = fig.subplots()
    ax.bar([1, 2, 3], [4, 9, 2])


def barh(fig):
    ax = fig.subplots()
    ax.barh([1, 2, 3], [4, 9, 2])


def hist(fig):
    ax = fig.subplots()
    ax.hist(X + Y, bins=5)


def imshow(fig):
    ax = fig.subplots()
    ax.imshow(Z)


def imshow_cbar(fig):
    ax = fig.subplots()
    m = ax.imshow(Z)
    fig.colorbar(m, ax=ax)


def scatter(fig):
    ax = fig.subplots()
    ax.scatter(X, Y)


def loglog(fig):
    ax = fig.subplots()
    ax.plot([0.5, 3, 40, 800], [2e-3, 1, 5, 7e4])
    ax.set_xscale("log")
    ax.set_yscale("log")


def fill(fig):
    ax = fig.subplots()
    ax.fill_between(X, Y, 0)
    ax.errorbar(X, Y, yerr=0.5)


def contourf(fig):
    ax = fig.subplots()
    ax.contourf([0, 1, 2, 3, 4, 5], [0, 10, 20, 30], Z)


def grid22(fig):
    axs = fig.subplots(2, 2)
    axs[0][0].plot(X, Y)
    axs[1][1].plot(Y, X)


def twin(fig):
    ax = fig.subplots()
    ax.plot(X, Y)
    t = ax.twinx()
    t.plot(X, [v * 1000 for v in Y])


def offset(fig):
    ax = fig.subplots()
    ax.plot([1e5 + v for v in X], [1e-7 * v for v in Y])


def axlines(fig):
    ax = fig.subplots()
    ax.plot(X, Y)
    ax.axhline(7)
    ax.axvline(-1)


def setlim(fig):
    ax = fig.subplots()
    ax.plot(X, Y)
    ax.set_xlim(5, -1)
    ax.set_ylim(2, 2)


CASES = dict(line=line, bar=bar, barh=barh, hist=hist, imshow=imshow,
             imshow_cbar=imshow_cbar, scatter=scatter, loglog=loglog, fill=fill,
             contourf=contourf, grid22=grid22, twin=twin, offset=offset,
             axlines=axlines, setlim=setlim)


def labelled(fig):
    ax = fig.subplots()
    ax.plot(X, Y)
    ax.set_title("A title")
    ax.set_xlabel("x label")
    ax.set_ylabel("y label")


def grid22_labelled(fig):
    axs = fig.subplots(2, 2)
    for i, a in enumerate(axs.flat):
        a.plot(X, [v * 10 ** i for v in Y])
        a.set_title(f"panel {i}")
        a.set_xlabel("time")
        a.set_ylabel("value")


def twin_labelled(fig):
    ax = fig.subplots()
    ax.plot(X, Y)
    ax.set_ylabel("left")
    t = ax.twinx()
    t.plot(X, [v * 1000 for v in Y])
    t.set_ylabel("right")


# Figures laid out again by fig.tight_layout(), the operation 27% of the
# notebooks in the census call.
TIGHT = dict(line=line, labelled=labelled, grid22_labelled=grid22_labelled,
             twin_labelled=twin_labelled, offset=offset, loglog=loglog)


def main():
    out = {"version": matplotlib.__version__, "cases": {}, "tight": {}}
    for size in [(6.4, 4.8), (5.0, 3.0)]:
        for name, mk in TIGHT.items():
            fig = plt.figure(figsize=size, dpi=100)
            mk(fig)
            fig.tight_layout()
            fig.canvas.draw()
            out["tight"][f"{name}@{size[0]}x{size[1]}"] = [
                [float(v) for v in (p.x0, p.y0, p.width, p.height)]
                for p in (ax.get_position() for ax in fig.axes)]
            plt.close(fig)
    for size in [(6.4, 4.8), (5.0, 3.0)]:
        for name, mk in CASES.items():
            fig = plt.figure(figsize=size, dpi=100)
            mk(fig)
            fig.canvas.draw()
            axes = []
            for ax in fig.axes:
                pos = ax.get_position()
                axes.append({
                    "position": [float(pos.x0), float(pos.y0), float(pos.width), float(pos.height)],
                    "xlim": [float(v) for v in ax.get_xlim()],
                    "ylim": [float(v) for v in ax.get_ylim()],
                    "xticks": [t.get_text() for t in ax.get_xticklabels()],
                    "yticks": [t.get_text() for t in ax.get_yticklabels()],
                    "xoffset": ax.xaxis.get_offset_text().get_text(),
                    "yoffset": ax.yaxis.get_offset_text().get_text(),
                })
            out["cases"][f"{name}@{size[0]}x{size[1]}"] = axes
            plt.close(fig)
    with open("testdata/layout.json", "w") as f:
        json.dump(out, f, indent=0)


if __name__ == "__main__":
    main()
