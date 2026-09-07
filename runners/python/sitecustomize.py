"""Seeds the random module, so a port that calls random.random produces the
same sequence on every run and on every machine.

Activated by putting this directory on PYTHONPATH. CPython imports
sitecustomize during startup, before the program runs, so the program does not
have to know it exists and does not have to be edited.

Two things to know about that mechanism. It does nothing under python -S, which
skips site entirely. And a sitecustomize already installed elsewhere is
shadowed rather than run alongside this one, because the first match on the
path wins.

What this does not reach: random.Random(...) instances, secrets, and numpy,
all of which seed themselves. A port using any of them needs its own answer;
see docs/determinism.md.
"""

import os
import random

random.seed(int(os.environ.get("PRESCRIPT_SEED", "0")))
