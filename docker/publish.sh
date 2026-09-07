#!/usr/bin/env bash
#
# Publishes the pinned environment so everyone runs the same bytes rather than
# their own build of the same Dockerfile.
#
# Everything in the Dockerfile is pinned, but a rebuild still produces its own
# layers: apt indexes move, timestamps differ, and the two images agree on
# every version without being the same image. That is enough for the toolchains
# to match and not enough to name in a bug report, which is what a digest is
# for.
#
#   docker/publish.sh [local image] [remote repository]
#
# Set GHCR_TOKEN to log in first; without it the daemon's existing credentials
# are used, which is what a developer pushing by hand already has.
set -euo pipefail

LOCAL_IMAGE="${1:-prescript-env:dev}"
REMOTE_IMAGE="${2:-ghcr.io/rjnienaber/prescript-env}"

if ! docker image inspect "$LOCAL_IMAGE" > /dev/null 2>&1; then
  echo "$LOCAL_IMAGE has not been built; run 'make image' first" >&2
  exit 1
fi

if [ -n "${GHCR_TOKEN:-}" ]; then
  echo "$GHCR_TOKEN" |
    docker login ghcr.io --username "${GHCR_USER:-${GITHUB_ACTOR:-x}}" --password-stdin
fi

# A commit tag as well as latest, because latest answers "what is current" and
# nothing else: a run that has to be reproduced needs a name that still means
# the same image next month.
COMMIT="${GITHUB_SHA:-$(git rev-parse HEAD)}"
COMMIT_TAG="git-${COMMIT:0:12}"

for TAG in latest "$COMMIT_TAG"; do
  echo "==> pushing $REMOTE_IMAGE:$TAG"
  docker tag "$LOCAL_IMAGE" "$REMOTE_IMAGE:$TAG"
  docker push "$REMOTE_IMAGE:$TAG" > /dev/null
done

# The digest only exists once the registry has the image, so it is read back
# rather than computed. A push that succeeded is not worth failing over if the
# daemon declines to report one, so the tag stands in.
DIGEST="$(docker image inspect \
  --format '{{range .RepoDigests}}{{println .}}{{end}}' "$REMOTE_IMAGE:$COMMIT_TAG" |
  grep "^$REMOTE_IMAGE@" | head -n 1 || true)"
if [ -z "$DIGEST" ]; then
  DIGEST="$REMOTE_IMAGE:$COMMIT_TAG"
fi
echo "==> published $DIGEST"

# The digest is the only part of this worth keeping, so it goes somewhere the
# run that produced it can be read from afterwards rather than only into logs
# that age out.
if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  {
    echo "### Pinned environment published"
    echo
    echo '```'
    echo "$DIGEST"
    echo '```'
    echo
    echo "Pull it with \`make pull_image PUBLISHED_REF=$DIGEST\`."
  } >> "$GITHUB_STEP_SUMMARY"
fi
