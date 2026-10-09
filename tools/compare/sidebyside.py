# Puts matplotlib's rendering (left) and go-plot's (right) side by side, for
# looking at: DIR/side_<name>.png.
#
#   uv run --with matplotlib --with numpy python sidebyside.py DIR name...
import sys

import matplotlib.image as mpimg
import numpy as np

out = sys.argv[1]
for name in sys.argv[2:]:
    a = mpimg.imread(f"{out}/mpl_{name}.png")[..., :3]
    b = mpimg.imread(f"{out}/go_{name}.png")[..., :3]
    h = max(a.shape[0], b.shape[0])
    pad = lambda m: np.pad(m, ((0, h - m.shape[0]), (0, 0), (0, 0)), constant_values=1)
    sep = np.zeros((h, 4, 3))
    sep[..., 0] = 1
    mpimg.imsave(f"{out}/side_{name}.png", np.concatenate([pad(a), sep, pad(b)], axis=1))
