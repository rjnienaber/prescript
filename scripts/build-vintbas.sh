#!/usr/bin/env bash
#
# Builds a deterministic `vintbas` from the Hackage release of vintage-basic.
#
# Upstream seeds its random number generator from the wall clock, which makes
# every run of a BASIC program different and so impossible to script against.
# patches/vintbas-deterministic-seed.patch replaces that with a fixed seed taken
# from $VINTBAS_SEED (default 0). See https://github.com/rjnienaber/prescript/issues/20.
#
# Usage: scripts/build-vintbas.sh [output-path]   (default: ./vintbas)
#
# Requires ghc and cabal on PATH. GHC 9.4.8 is what CI pins and what the
# patches are verified against. To build without installing a Haskell
# toolchain, run it inside the official image:
#
#   docker run --rm -v "$PWD:/src" -w /src haskell:9.4.8 scripts/build-vintbas.sh
#
# Prefer the prebuilt binaries from this repository's `vintbas-*` releases
# (see `make vintbas`); this script is how those are produced.
#
set -euo pipefail

# vintage-basic is unmaintained, so these are expected to stay put. The checksum
# is here to make the build fail loudly rather than silently pick up a
# substituted tarball.
VINTAGE_BASIC_VERSION=1.0.3
VINTAGE_BASIC_SHA256=687551dc3c7b2a1056a7451ffbcd3730357634c01a75c970176d13eec97f67f3

# `random` 1.2 replaced the LCG behind StdGen with SplitMix, so `mkStdGen 0`
# yields a completely different sequence either side of that release. Every
# expected-output fixture is tied to this number: changing it invalidates them.
RANDOM_VERSION=1.2.1.2

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PATCH_DIR="$SCRIPT_DIR/../patches"
OUTPUT="$(pwd)/${1:-vintbas}"

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "$WORK_DIR"' EXIT

TARBALL="vintage-basic-$VINTAGE_BASIC_VERSION.tar.gz"
echo "==> downloading $TARBALL"
curl -fsSL -o "$WORK_DIR/$TARBALL" \
  "https://hackage.haskell.org/package/vintage-basic-$VINTAGE_BASIC_VERSION/$TARBALL"

echo "==> verifying checksum"
echo "$VINTAGE_BASIC_SHA256  $WORK_DIR/$TARBALL" | shasum -a 256 -c -

echo "==> applying patches"
tar xzf "$WORK_DIR/$TARBALL" -C "$WORK_DIR"
cd "$WORK_DIR/vintage-basic-$VINTAGE_BASIC_VERSION"
for patch_file in "$PATCH_DIR"/vintbas-*.patch; do
  echo "    $(basename "$patch_file")"
  patch -p1 --batch < "$patch_file"
done

echo "==> building with random==$RANDOM_VERSION"
cabal update
cabal build --constraint "random ==$RANDOM_VERSION"

cp "$(cabal list-bin vintbas)" "$OUTPUT"
chmod +x "$OUTPUT"
echo "==> wrote $OUTPUT"
