# Measures go-plot's renderings against matplotlib's, structurally, with the
# same detector on both images (so neither side is judged by a rule the
# other is not):
#
#   frame  the axes frames: columns and rows holding a long run of dark
#          pixels (the spines), compared position by position;
#   ticks  the tick marks: dark runs just outside the bottom and left spines
#          of every frame, compared as positions;
#   line   pixels of the first cycle colour (C0) inside the frames: their
#          count, and their overlap after a 1-pixel dilation of each side;
#   pixels the mean absolute difference over every pixel and channel.
#
#   uv run --with matplotlib --with numpy python compare.py DIR [go_name=mpl_name ...]
#
# Exits non-zero when a figure is outside the tolerances printed at the end.
import json
import sys

import matplotlib.image as mpimg
import numpy as np

TOL_FRAME = 1      # pixels
TOL_TICK = 1       # pixels
TOL_COUNT = 0.10   # relative difference in C0 pixel count
TOL_IOU = 0.90     # overlap of C0 pixels, each dilated by 1 pixel


def load(path):
    im = mpimg.imread(path)[..., :3]
    return (im * 255).astype(np.int32)


def dark(im):
    return (im.max(axis=2) < 80)


def runs(mask1d):
    """(start, end) of each run of True."""
    out, start = [], None
    for i, v in enumerate(mask1d):
        if v and start is None:
            start = i
        elif not v and start is not None:
            out.append((start, i))
            start = None
    if start is not None:
        out.append((start, len(mask1d)))
    return out


def frames(im):
    """Spine columns and rows: a dark run at least 60 pixels long."""
    d = dark(im)
    cols = [x for x in range(d.shape[1]) if any(e - s >= 60 for s, e in runs(d[:, x]))]
    rows = [y for y in range(d.shape[0]) if any(e - s >= 60 for s, e in runs(d[y, :]))]
    return cols, rows


def group(v):
    """Collapse adjacent indices into their centres."""
    out = []
    for i in v:
        if out and i - out[-1][-1] <= 1:
            out[-1].append(i)
        else:
            out.append([i])
    return [sum(g) / len(g) for g in out]


def ticks(im, cols, rows):
    """Tick centres: dark for the three pixels just below each bottom spine,
    or just left of each left spine (glyphs of labels are not that close)."""
    d = dark(im)
    gc, gr = group(cols), group(rows)
    xt, yt = [], []
    # frames come in pairs: top/bottom rows, left/right columns
    # only between the frame's own spines, so that no label glyph counts
    for y in gr[1::2]:
        r = int(round(y))
        for x0, x1 in zip(gc[0::2], gc[1::2]):
            xt += group([x for x in range(int(x0) - 1, int(x1) + 2) if d[r + 1:r + 4, x].all()])
    for x in gc[0::2]:
        c = int(round(x))
        for y0, y1 in zip(gr[0::2], gr[1::2]):
            yt += group([y for y in range(int(y0) - 1, int(y1) + 2) if d[y, c - 3:c].all()])
    return xt, yt


def c0mask(im, cols, rows):
    m = (np.abs(im - np.array([31, 119, 180])).sum(axis=2) < 60)
    inside = np.zeros_like(m)
    gc, gr = group(cols), group(rows)
    for x0, x1 in zip(gc[0::2], gc[1::2]):
        for y0, y1 in zip(gr[0::2], gr[1::2]):
            inside[int(y0) + 2:int(y1) - 1, int(x0) + 2:int(x1) - 1] = True
    return m & inside


def dilate(m):
    out = m.copy()
    out[1:] |= m[:-1]
    out[:-1] |= m[1:]
    out[:, 1:] |= m[:, :-1]
    out[:, :-1] |= m[:, 1:]
    return out


def match(a, b):
    """Largest distance between paired positions, or None if the counts differ."""
    if len(a) != len(b):
        return None
    return max((abs(p - q) for p, q in zip(sorted(a), sorted(b))), default=0.0)


def compare(goimg, mplimg):
    g, m = load(goimg), load(mplimg)
    res = {}
    if g.shape != m.shape:
        return {"size": [list(g.shape), list(m.shape)], "ok": False}
    gcols, grows = frames(g)
    mcols, mrows = frames(m)
    res["frame_cols"] = [group(gcols), group(mcols)]
    res["frame_rows"] = [group(grows), group(mrows)]
    fx, fy = match(group(gcols), group(mcols)), match(group(grows), group(mrows))
    res["frame_err"] = None if fx is None or fy is None else max(fx, fy)
    gx, gy = ticks(g, gcols, grows)
    mx, my = ticks(m, mcols, mrows)
    tx, ty = match(gx, mx), match(gy, my)
    res["ticks"] = [len(gx) + len(gy), len(mx) + len(my)]
    res["tick_err"] = None if tx is None or ty is None else max(tx, ty)
    gm, mm = c0mask(g, gcols, grows), c0mask(m, mcols, mrows)
    res["c0_pixels"] = [int(gm.sum()), int(mm.sum())]
    if mm.sum() > 0:
        res["c0_count_rel"] = abs(int(gm.sum()) - int(mm.sum())) / mm.sum()
        inter = (dilate(gm) & mm).sum() + (gm & dilate(mm)).sum()
        res["c0_overlap"] = inter / (gm.sum() + mm.sum())
    res["mean_abs_diff"] = float(np.abs(g - m).mean())
    res["pixels_off_64"] = float((np.abs(g - m).max(axis=2) > 64).mean())
    ok = (res["frame_err"] is not None and res["frame_err"] <= TOL_FRAME
          and res["tick_err"] is not None and res["tick_err"] <= TOL_TICK)
    if "c0_overlap" in res:
        ok = ok and res["c0_count_rel"] <= TOL_COUNT and res["c0_overlap"] >= TOL_IOU
    res["ok"] = bool(ok)
    return res


if __name__ == "__main__":
    out = sys.argv[1]
    pairs = [a.split("=") for a in sys.argv[2:]]
    bad = 0
    report = {}
    for gname, mname in pairs:
        r = compare(f"{out}/go_{gname}.png", f"{out}/mpl_{mname}.png")
        report[gname] = r
        flag = "ok  " if r["ok"] else "FAIL"
        print(f"{flag} {gname:14s} vs {mname:8s} frame±{r.get('frame_err')} "
              f"ticks {r.get('ticks')} ±{r.get('tick_err')} "
              f"C0 {r.get('c0_pixels')} overlap {r.get('c0_overlap', float('nan')):.3f} "
              f"|Δ|={r.get('mean_abs_diff', float('nan')):.2f} off64={100*r.get('pixels_off_64', 0):.2f}%")
        bad += not r["ok"]
    with open(f"{out}/compare.json", "w") as f:
        json.dump(report, f, indent=1, default=float)
    print(f"tolerances: frame <= {TOL_FRAME}px, ticks <= {TOL_TICK}px, "
          f"C0 count within {TOL_COUNT:.0%}, C0 overlap >= {TOL_IOU}")
    sys.exit(1 if bad else 0)
