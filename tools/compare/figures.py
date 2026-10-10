# Renders the reference figures with matplotlib: the same figures, data,
# size and dpi as main.go renders with go-plot.
#
#   uv run --with matplotlib --with numpy python figures.py DIR
import math
import sys

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
import numpy as np  # noqa: E402


class LCG:
    """The generator main.go also implements."""

    def __init__(self, s):
        self.s = s

    def uniform(self):
        self.s = (1664525 * self.s + 1013904223) % 2**32
        return self.s / 4294967296.0

    def normal(self):
        u1, u2 = self.uniform(), self.uniform()
        return math.sqrt(-2 * math.log(1 - u1)) * math.cos(2 * math.pi * u2)


def field(nx, ny):
    xs, ys = np.linspace(-3, 3, nx), np.linspace(-2, 2, ny)
    X, Y = np.meshgrid(xs, ys)
    z = (1 - X / 2 + X**5 + Y**3) * np.exp(-X**2 - Y**2)
    return xs, ys, z


def line():
    fig, ax = plt.subplots()
    x = np.linspace(0, 10, 200)
    ax.plot(x, np.sin(x), label="sin")
    ax.plot(x, np.cos(x), label="cos", linestyle="--")
    ax.set_title("Line plot")
    ax.set_xlabel("time (s)")
    ax.set_ylabel("amplitude")
    ax.legend()
    return fig


def scatter():
    fig, ax = plt.subplots()
    g = LCG(1)
    x, y, c, s = [], [], [], []
    for _ in range(150):
        xi, yi = g.normal(), g.normal()
        x.append(xi)
        y.append(yi)
        c.append(xi + yi)
        s.append(10 + 90 * g.uniform())
    m = ax.scatter(x, y, c=c, s=s, cmap="plasma")
    fig.colorbar(m, ax=ax)
    ax.set_title("Scatter")
    return fig


def bars():
    fig, axs = plt.subplots(1, 2, figsize=(8, 4))
    axs[0].bar([1, 2, 3, 4, 5], [3, 7, 2, 5, 4], label="a")
    axs[0].bar([1, 2, 3, 4, 5], [1, 2, 1, 2, 1], bottom=0, width=0.4, label="b")
    axs[0].legend()
    axs[1].barh([0, 1, 2], [10, 25, 17])
    axs[1].set_title("barh")
    return fig


def hist():
    fig, ax = plt.subplots()
    g = LCG(7)
    v = [5 + 2 * g.normal() for _ in range(2000)]
    ax.hist(v, bins=30)
    ax.set_xlabel("value")
    ax.set_ylabel("count")
    return fig


def imshow():
    fig, ax = plt.subplots()
    _, _, z = field(40, 30)
    m = ax.imshow(z)
    fig.colorbar(m, ax=ax)
    return fig


def contour():
    fig, ax = plt.subplots()
    xs, ys, z = field(80, 60)
    m = ax.contourf(xs, ys, z, cmap="coolwarm")
    ax.contour(xs, ys, z, colors="k", linewidths=0.8)
    fig.colorbar(m, ax=ax)
    return fig


def log():
    fig, ax = plt.subplots()
    x = np.linspace(0.1, 100, 300)
    ax.plot(x, x * x)
    ax.plot(x, np.exp(x / 10))
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.grid(True)
    return fig


def misc():
    fig, ax = plt.subplots()
    x = np.linspace(0, 4, 9)
    y = x * x / 4 + 1
    ax.errorbar(x, y, yerr=0.3, capsize=3, label="data")
    ax.fill_between(x, y - 0.5, y + 0.5, alpha=0.3)
    ax.axhline(2, color="gray", linestyle=":")
    ax.axvline(1.5, color="r")
    ax.text(0.2, 4.5, "a note")
    ax.annotate("peak", xy=(4, 5), xytext=(2.5, 5.5), arrowprops=dict(arrowstyle="->"))
    ax.set_title("Errors, fills and lines")
    return fig


def twinx():
    fig, ax = plt.subplots()
    x = np.linspace(0, 10, 50)
    ax.plot(x, np.exp(x), color="C0")
    ax.set_ylabel("exp")
    t = ax.twinx()
    t.plot(x, np.sin(x), color="C1")
    t.set_ylabel("sin")
    ax.set_xlabel("x")
    return fig


def grid():
    fig, axs = plt.subplots(2, 2)
    x = np.linspace(-1, 1, 101)
    axs[0][0].plot(x, x**3)
    axs[0][1].plot(x, 1e-6 * x)
    axs[1][0].plot(x, 1000 + x)
    axs[1][1].scatter(x, np.abs(x), marker="s")
    for a in axs.flat:
        a.grid(True)
    return fig


def tight():
    fig, axs = plt.subplots(2, 2)
    x = np.linspace(0, 10, 100)
    for i in range(2):
        for j in range(2):
            k = 1 + 2 * i + j
            a = axs[i][j]
            a.plot(x, 10.0 ** k * np.sin(x / k))
            a.set_title(f"panel {k}")
            a.set_xlabel("time (s)")
            a.set_ylabel("signal")
    axs[0][0].axhline(0, color="k", linewidth=0.8)
    fig.tight_layout()
    return fig


def ndarray():
    fig, ax = plt.subplots()
    ax.plot(np.array([3, 1, 4, 1, 5, 9, 2, 6, 5, 3], dtype=np.int32), marker="o")
    return fig


FIGURES = dict(line=line, scatter=scatter, bars=bars, hist=hist, imshow=imshow,
               contour=contour, log=log, misc=misc, twinx=twinx, grid=grid, tight=tight,
               ndarray=ndarray)

if __name__ == "__main__":
    out = sys.argv[1]
    for name, mk in FIGURES.items():
        fig = mk()
        fig.savefig(f"{out}/mpl_{name}.png")
        fig.savefig(f"{out}/mpl_{name}.svg")
        plt.close(fig)
        print("wrote", name)
