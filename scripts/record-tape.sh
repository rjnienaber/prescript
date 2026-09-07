#!/usr/bin/env bash
#
# Records the sequence of random values the reference BASIC interpreter
# produces, as a tape that can be replayed to a port in any language.
#
# Randomness is treated as an input channel rather than as noise to be
# suppressed. Seeding makes each implementation repeatable but leaves every
# implementation drawing a different sequence, so one expected transcript still
# cannot serve them all. A tape gives every port the same numbers, which is
# what makes one transcript enough -- and when a port drains the tape at a
# different rate, that divergence is itself the finding.
#
# The tape does not require instrumenting vintbas. Its generator is a pure
# `random` call over a StdGen threaded through interpreter state, so a BASIC
# program that does nothing but print RND(1) reads the sequence straight out.
#
# Usage: scripts/record-tape.sh [count] [seed]
set -euo pipefail

COUNT="${1:-10000}"
SEED="${2:-0}"

# Must match the Makefile: the release identifies the build, and the build is
# what fixes the sequence.
VINTBAS_RELEASE="vintbas-1.0.3-1"
VINTAGE_BASIC_VERSION="1.0.3"

# `random` 1.2 replaced the LCG behind StdGen with SplitMix, so mkStdGen 0
# yields a completely different sequence either side of it. A tape is only
# meaningful alongside the version that produced it, which is why the number is
# written into the file rather than left in a build script.
RANDOM_VERSION="1.2.1.2"

REPOSITORY="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="$REPOSITORY/tapes/vintbas-seed$SEED.tape"

if ! command -v vintbas > /dev/null; then
  echo "vintbas is not on PATH; run 'make vintbas' first" >&2
  exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

cat > "$WORK_DIR/tape.bas" <<BAS
10 FOR I = 1 TO $COUNT
20 PRINT RND(1)
30 NEXT I
BAS

echo "==> reading $COUNT values from $VINTBAS_RELEASE at seed $SEED"

# PRINT pads a number with a leading and a trailing space, and prints values
# below 0.01 in exponent form. Both are read back as decimals by every shim, so
# only the padding is stripped here: rewriting the numbers themselves would be
# the one thing this script must not do.
mkdir -p "$(dirname "$OUTPUT")"
{
  echo "# prescript tape 1"
  echo "# source: vintage-basic $VINTAGE_BASIC_VERSION, built as $VINTBAS_RELEASE"
  echo "# random: $RANDOM_VERSION"
  echo "# seed: $SEED"
  echo "# values: $COUNT"
  echo "#"
  echo "# Every value is the shortest decimal that reads back as the single-"
  echo "# precision float the interpreter drew, so replaying this file gives a"
  echo "# port the reference's numbers exactly rather than approximately."
  VINTBAS_SEED="$SEED" vintbas "$WORK_DIR/tape.bas" | tr -d ' \r'
} > "$OUTPUT"

RECORDED="$(grep -vc '^#' "$OUTPUT" || true)"
if [ "$RECORDED" != "$COUNT" ]; then
  echo "expected $COUNT values, recorded $RECORDED" >&2
  exit 1
fi

echo "==> wrote $RECORDED values to $OUTPUT"
