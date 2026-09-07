"""Makes a Python port's randomness repeatable, in one of two ways.

With PRESCRIPT_TAPE set, random.random returns values recorded from the
reference BASIC interpreter, in order, so this port and every other port of the
same program draw the same numbers. Without it, Python's own generator is
seeded from PRESCRIPT_SEED, which makes this port repeatable but leaves it
drawing a different sequence from every other language. See
docs/determinism.md.

Activated by putting this directory on PYTHONPATH. CPython imports
sitecustomize during startup, before the program runs, so the program does not
have to know it exists and does not have to be edited.

Two things to know about that mechanism. It does nothing under python -S, which
skips site entirely. And a sitecustomize already installed elsewhere is
shadowed rather than run alongside this one, because the first match on the
path wins.

What this does not reach: random.Random(...) instances, secrets, and numpy,
all of which have generators of their own.
"""

import atexit
import os
import random


def _read_tape(path):
    values = []
    with open(path, encoding="utf-8") as tape:
        for line in tape:
            line = line.strip()
            if line and not line.startswith("#"):
                values.append(float(line))
    return values


def _install_tape(path):
    values = _read_tape(path)
    drawn = 0

    def next_value():
        # Running off the end is an error rather than a wrap. A port that draws
        # more than the reference did has restructured how it consumes
        # randomness, which is a finding; silently starting the tape again
        # would turn that finding into a transcript that merely looks wrong
        # somewhere further on.
        nonlocal drawn
        if drawn >= len(values):
            raise RuntimeError(
                "prescript: the tape ran out after %d values; this port draws "
                "more randomness than the reference did" % len(values)
            )

        value = values[drawn]
        drawn += 1
        return value

    instance = random._inst
    instance.random = next_value

    # randrange, randint and choice reach for getrandbits, which is the Mersenne
    # Twister's own and knows nothing about a tape. Routing them through the
    # variant built on random() instead is what makes the whole module draw
    # from one place.
    instance._randbelow = instance._randbelow_without_getrandbits

    # The module-level names were bound to the instance's methods at import,
    # so the one that was captured has to be replaced by name as well.
    random.random = next_value

    # How much of the tape a port used is the number that separates a real
    # logic bug from a port that restructured its draws, so it is written down
    # rather than left to be inferred from the transcript.
    usage = os.environ.get("PRESCRIPT_TAPE_USAGE")
    if usage:
        atexit.register(lambda: open(usage, "w", encoding="utf-8").write("%d\n" % drawn))


_tape = os.environ.get("PRESCRIPT_TAPE")
if _tape:
    _install_tape(_tape)
else:
    random.seed(int(os.environ.get("PRESCRIPT_SEED", "0")))
