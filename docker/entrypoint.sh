#!/bin/bash
#
# Turns off address-space randomisation for everything run in the container.
#
# ASLR is the one source of churn that no environment variable reaches. A
# runtime that folds a pointer address into a hash or an RNG seed -- and
# several do, as a defence against collision attacks -- prints something
# different on every run of an otherwise identical program, which looks exactly
# like the divergence this tool exists to find.
#
# Best effort on purpose, and usually refused: Docker's default seccomp profile
# permits personality() with a handful of arguments and ADDR_NO_RANDOMIZE is
# not among them, so holding the address layout still means dropping the
# profile. docker/run.sh does that when PRESCRIPT_FIX_ADDRESS_LAYOUT=1 asks it
# to. Refusal is not fatal here -- a container that will not start is worse
# than one that is slightly less repeatable -- and the warning goes to stderr
# so it lands in the log rather than in a transcript.
set -euo pipefail

if setarch --addr-no-randomize true 2> /dev/null; then
  exec setarch --addr-no-randomize "$@"
fi

echo "prescript: address-space randomisation is on, so a runtime that mixes pointers into a hash or a seed will vary between runs; set PRESCRIPT_FIX_ADDRESS_LAYOUT=1 to turn it off" >&2
exec "$@"
