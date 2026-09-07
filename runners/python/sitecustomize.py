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

    # How much of the tape a port used is the number that separates a real
    # logic bug from a port that restructured its draws, so it is written down
    # rather than left to be inferred from the transcript.
    #
    # It is rewritten after every draw rather than once at exit, because the
    # number that matters is how much had been drawn when the run stopped
    # agreeing with the transcript -- and a run that diverged by hanging is
    # killed, which runs no atexit handler at all. Rewriting in place is safe
    # without truncating: the count only ever grows, so its decimal form only
    # ever gets longer and each write covers the last.
    usage = os.environ.get("PRESCRIPT_TAPE_USAGE")
    usage_file = open(usage, "w", encoding="utf-8") if usage else None

    def write_usage():
        if usage_file is None:
            return
        usage_file.seek(0)
        usage_file.write("%d\n" % drawn)
        usage_file.flush()

    # Written from the start, so that a port which drew nothing at all says so
    # rather than saying nothing.
    write_usage()

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
        write_usage()
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


_tape = os.environ.get("PRESCRIPT_TAPE")
if _tape:
    _install_tape(_tape)
else:
    random.seed(int(os.environ.get("PRESCRIPT_SEED", "0")))
