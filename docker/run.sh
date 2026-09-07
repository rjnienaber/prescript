#!/usr/bin/env bash
#
# Runs a command inside the pinned environment, with the repository mounted.
#
#   docker/run.sh make test
#   docker/run.sh bash
#
# Everything the container needs to be reproducible is set here rather than
# left to whoever types the docker command, because a run that forgot one of
# these flags looks exactly like a run that did not.
#
# The repository is mounted so a change can be tried without a rebuild, but
# tmp/ is not: `make build_dev` in here produces a linux binary, and dropping
# that on top of the one the host just built turns the next `make examples`
# into "cannot execute binary file".
set -euo pipefail

IMAGE="${PRESCRIPT_IMAGE_NAME:-prescript-env:dev}"
REPOSITORY="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! docker image inspect "$IMAGE" > /dev/null 2>&1; then
  echo "$IMAGE has not been built; run 'make image' first" >&2
  exit 1
fi

# What a bug report has to name alongside the versions inside the image: which
# image. A published image is identified by its registry digest and a locally
# built one by its config digest, and either is enough for someone else to get
# the same bytes -- but only if the name in front of the digest is one they can
# reach. `make pull_image` gives the pulled image a local tag, and the daemon
# then reports a digest under that short name as well as under the registry
# one, so the registry-qualified entry is picked out by its hostname rather
# than by being listed first.
DIGESTS="$(docker image inspect --format '{{range .RepoDigests}}{{println .}}{{end}}' "$IMAGE")"
DIGEST="$(echo "$DIGESTS" | grep -E '^([^/]+\.[^/]*|localhost(:[0-9]+)?)/' | head -n 1 || true)"
if [ -z "$DIGEST" ]; then
  DIGEST="$(docker image inspect --format '{{.Id}}' "$IMAGE")"
fi

# Docker's default seccomp profile permits personality() with a handful of
# arguments and ADDR_NO_RANDOMIZE is not among them, so fixing the address
# layout means dropping the profile entirely. That is off by default and asked
# for explicitly, because the code being compared is other people's and the
# repository is mounted writable: the profile is worth keeping until a run
# actually needs the address layout held still.
SECCOMP=()
if [ "${PRESCRIPT_FIX_ADDRESS_LAYOUT:-0}" = "1" ]; then
  SECCOMP=(--security-opt seccomp=unconfined)
fi

# A tty only when there is one to pass on, so this works the same from a
# terminal and from CI. Written the long way because macOS still ships bash
# 3.2, where an empty array under `set -u` is an error rather than nothing.
INTERACTIVE=()
if [ -t 0 ]; then
  INTERACTIVE=(--interactive --tty)
fi

exec docker run --rm ${INTERACTIVE[@]+"${INTERACTIVE[@]}"} ${SECCOMP[@]+"${SECCOMP[@]}"} \
  --platform linux/amd64 \
  --volume "$REPOSITORY:/workspace" \
  --mount type=volume,source=prescript-tmp,target=/workspace/tmp \
  --mount type=volume,source=prescript-go-build,target=/root/.cache/go-build \
  --mount type=volume,source=prescript-go-mod,target=/root/go/pkg/mod \
  --env PRESCRIPT_IMAGE="$DIGEST" \
  --env PRESCRIPT_REQUIRE_RUNNERS=1 \
  "$IMAGE" "$@"
