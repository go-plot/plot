# Crops the same region of matplotlib's and go-plot's rendering, enlarged,
# side by side: DIR/crop_<name>.png.
#
#   uv run --with matplotlib --with numpy python crop.py DIR name x0 y0 x1 y1 [scale]
import sys

import matplotlib.image as mpimg
import numpy as np

out, name = sys.argv[1], sys.argv[2]
x0, y0, x1, y1 = map(int, sys.argv[3:7])
k = int(sys.argv[7]) if len(sys.argv) > 7 else 6
a = mpimg.imread(f"{out}/mpl_{name}.png")[y0:y1, x0:x1, :3]
b = mpimg.imread(f"{out}/go_{name}.png")[y0:y1, x0:x1, :3]
up = lambda m: np.kron(m, np.ones((k, k, 1)))
sep = np.zeros(((y1 - y0) * k, 6, 3))
sep[..., 0] = 1
mpimg.imsave(f"{out}/crop_{name}.png", np.concatenate([up(a), sep, up(b)], axis=1))
