# Generates ticker/testdata/units.json: matplotlib's locators and formatters
# called directly, with the parameters a default figure never uses (nbins,
# steps, integer, symmetric, prune, base, subs, math text, power limits...),
# so that every branch of the port is checked against matplotlib itself.
#
#   uv run --with matplotlib --with numpy python tools/gen_ticker_units.py
#
# Run from the repository root.
import json
import math

import matplotlib
import matplotlib.ticker as mt
import matplotlib.transforms as mtr
import numpy as np


class Axis:
    """The little of an Axis a formatter asks for."""

    def __init__(self, vmin, vmax):
        self.v = (vmin, vmax)

    def get_view_interval(self):
        return self.v

    def get_minpos(self):
        return 1e-300


def floats(a):
    return [float(v) for v in a]


def main():
    out = {"version": matplotlib.__version__, "maxn": [], "log": [], "scalar": [], "logfmt": [], "nonsingular": [], "fmtg": []}
    for params in [
        dict(nbins=5), dict(nbins=3, steps=[1, 3, 10]), dict(nbins=4, steps=[2, 5]),
        dict(nbins=6, integer=True), dict(nbins=5, symmetric=True), dict(nbins=5, prune="lower"),
        dict(nbins=5, prune="upper"), dict(nbins=5, prune="both"), dict(nbins=9, integer=True, min_n_ticks=4),
    ]:
        for lo, hi in [(0, 1), (-3.3, 7.7), (0.2, 2.9), (1e6, 1e6 + 7), (-2, -1)]:
            loc = mt.MaxNLocator(**params)
            out["maxn"].append({"params": params, "vmin": lo, "vmax": hi, "locs": floats(loc.tick_values(lo, hi))})
    for params in [dict(base=2), dict(base=math.e), dict(base=10, subs=[2.0, 5.0]), dict(base=10, numticks=3),
                   dict(base=10, subs="auto"), dict(base=10, subs="all"), dict(base=2, subs="auto")]:
        for lo, hi in [(1, 1000), (0.3, 7), (5, 2), (1e-8, 1e8), (3, 4), (1.5, 2.5)]:
            loc = mt.LogLocator(**params)
            p = dict(params)
            if p.get("base") == math.e:
                p["base"] = "e"
            out["log"].append({"params": p, "vmin": lo, "vmax": hi, "locs": floats(loc.tick_values(lo, hi))})
    for params in [dict(), dict(useMathText=True), dict(useOffset=False), dict(powerlimits=(3, 3)),
                   dict(scientific=False), dict(unicode=False)]:
        for locs, view in [([0.0, 0.5, 1.0], (0, 1)), ([1e5 + 1, 1e5 + 2, 1e5 + 3], (1e5, 1e5 + 4)),
                           ([2e-7, 4e-7, 6e-7], (0, 7e-7)), ([5.0], (4.0, 6.0)), ([0.0], (-1e-9, 1e-9)),
                           ([3.0, 3.0], (2, 4)), ([], (0, 1)), ([-1e7, 0.0, 1e7], (-1e7, 1e7)),
                           ([0.0, 0.0], (0, 0)), ([10.0], (20, 30)), ([123456.8, 123457.0, 123457.2], (123456.7, 123457.2)), ([1e10 + 1, 1e10 + 2], (1e10, 1e10 + 3))]:
            f = mt.ScalarFormatter(useMathText=params.get("useMathText", False),
                                   useOffset=params.get("useOffset", True))
            if "powerlimits" in params:
                f.set_powerlimits(params["powerlimits"])
            if "scientific" in params:
                f.set_scientific(False)
            f.axis = Axis(*view)
            with matplotlib.rc_context({"axes.unicode_minus": params.get("unicode", True)}):
                f.set_locs(np.array(locs))
                labels = [f(v) for v in locs]
                off = f.get_offset()
            p = {k: (list(v) if isinstance(v, tuple) else v) for k, v in params.items()}
            out["scalar"].append({"params": p, "locs": locs, "view": list(view), "labels": labels, "offset": off})
    for params in [dict(), dict(labelOnlyBase=True), dict(base=2), dict(minor_thresholds=(math.inf, math.inf)),
                   dict(minor_thresholds=(10, 10))]:
        for view, vals in [((1, 1000), [1, 10, 100, 2, 50, 0.0, -100, 3e-5]), ((2, 3), [2, 2.5, 3]),
                           ((-1, 10), [1, 2, 5]), ((10, 1), [1, 2, 3, 4, 6, 7]), ((1, 4), [1, 2, 3, 4])]:
            f = mt.LogFormatterSciNotation(base=params.get("base", 10), labelOnlyBase=params.get("labelOnlyBase", False),
                                           minor_thresholds=params.get("minor_thresholds"))
            f.axis = Axis(*view)
            f.set_locs(vals)
            p = {k: ([("inf" if math.isinf(x) else x) for x in v] if isinstance(v, tuple) else v) for k, v in params.items()}
            out["logfmt"].append({"params": p, "view": list(view), "vals": vals, "labels": [f(v) for v in vals]})
    for args in [(0, 0, 0.1, 1e-15, True), (1, 1, 0.05, 1e-15, True), (5, 1, 0.05, 1e-15, False), (5, 1, 0.05, 1e-15, True),
                 (math.inf, 1, 0.1, 1e-15, True), (1e-320, 2e-320, 0.1, 1e-15, True), (1e20, 1e20 + 1, 0.05, 1e-15, True),
                 (-3, 2, 0.05, 1e-15, True)]:
        a, b = mtr.nonsingular(*args[:2], expander=args[2], tiny=args[3], increasing=args[4])
        out["nonsingular"].append({"args": [("inf" if isinstance(v, float) and math.isinf(v) else v) for v in args], "out": [float(a), float(b)]})
    for v, prec in [(0.0, 6), (-0.0, 6), (1.5e-5, 6), (123456789.0, 6), (2.5, 10), (1e100, 10), (-3.25e-12, 3), (100000.0, 6), (0.0001, 6), (5e-5, 6)]:
        out["fmtg"].append({"v": v, "prec": prec, "s": ("%%1.%dg" % prec) % v})
    with open("ticker/testdata/units.json", "w") as f:
        json.dump(out, f, indent=0)


if __name__ == "__main__":
    main()
