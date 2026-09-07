# The pinned environment

A difference between two ports only means something when both ran in the same
world. [Determinism](determinism.md) covers the randomness half of that — a
seed makes one port repeatable, a tape makes every port draw the same numbers.
This is the other half, and in practice the larger one: interpreter versions,
locale, timezone, terminal size, hash seeds and address layout all change what
a program prints without changing what it does.

```
make pull_image   # get the published one
make image        # or build it yourself
make docker_test  # run the suite inside it
make docker_shell # look around
```

## What is pinned

| | How |
| --- | --- |
| Base | `debian:bookworm-slim` by digest |
| Ruby, Python | Debian stable, a fixed series taking security fixes only |
| Go, Node | upstream tarballs, checked against a published SHA-256 |
| vintbas | the same release asset the Makefile installs, checked the same way |
| Locale | `LANG=LC_ALL=C.UTF-8` |
| Timezone | `TZ=UTC` |
| Terminal | 80×24, set on the pty prescript gives the child |
| Hash seed | `PYTHONHASHSEED=0` |
| Address layout | `setarch --addr-no-randomize`, opt-in; see below |

`GOTOOLCHAIN=local` is in there for the same reason as everything else: without
it the `go` command will fetch a newer toolchain when a module asks for one,
which would quietly unpin the Go that was so carefully pinned above.

Rebuilding produces the same toolchains next year as today. That is the whole
point — an image that drifts turns a fixed bug back into an open one.

## amd64 only, deliberately

The reference interpreter is published for linux x86_64. Comparing a port
against a reference built for a different architecture folds floating-point and
standard-library differences into every comparison, which is exactly the noise
this image exists to remove. Building on an arm64 machine works through
emulation and is slow; running the tests that way is fine, recording a tape
that way is not.

## Terminal size is prescript's job, not the image's

`COLUMNS` and `LINES` are set here, but the number that actually matters is the
one the kernel reports for the pty, and that is prescript's to set. It gives
every child an 80×24 pty whatever terminal prescript itself was started from —
a pty opened without a size reports zero rows and columns, and a program asking
how wide its terminal is would otherwise get an answer that is neither what a
real terminal gives nor what the next machine gives.

## What a bug report needs

Two things, and both are readable from inside a run:

- `$PRESCRIPT_ENVIRONMENT` names a file listing the base digest and every
  toolchain version. It is written at build time from the binaries that were
  actually installed, not from the arguments that asked for them, so it cannot
  claim a version the image does not have.
- `$PRESCRIPT_IMAGE` is the image itself — its registry digest when it was
  pulled, its config digest when it was built locally. `docker/run.sh` sets it,
  because an image can not know its own digest at build time.

```
$ make docker_shell
# cat $PRESCRIPT_ENVIRONMENT
base=debian:bookworm-slim@sha256:8820...
arch=x86_64
go=go1.24.13
node=v24.20.0
python=3.11.2
ruby=3.1.2p20
vintbas=vintbas-1.0.3-1
locale=C.UTF-8
timezone=UTC
```

Issue #18 folds both into the repro block of a generated bug report.

## Where it lives

`ghcr.io/rjnienaber/prescript-env`, pushed by CI from `main` after the suite
has passed inside it. Building from the same pinned Dockerfile makes two
images agree on every version without making them the same image — apt
indexes move, timestamps differ, and the layers come out different. Pulling
removes that last gap, and gives the run a digest to be named by.

```
make pull_image                                                  # :latest
make pull_image PUBLISHED_REF=ghcr.io/rjnienaber/prescript-env@sha256:...
```

Both give the image the local name the rest of the tooling uses, so
`make docker_test` and `docker/run.sh` do not need to know which one was used.
A tag says which build; a digest says which bytes, and a digest is what to use
when reproducing someone else's run — the publishing step prints the one it
pushed, and `$PRESCRIPT_IMAGE` reports the one a run is actually inside.

The package is public. Public packages on GitHub Packages carry no storage or
bandwidth charge; a private one would bill against the account's quota, and
this image is around a gigabyte. If a first publish ever creates it private,
the fix is one setting on the package, not a change here.

## What it does not do

Address-space randomisation is left on by default. Docker's default seccomp
profile permits `personality()` with a handful of arguments and
`ADDR_NO_RANDOMIZE` is not among them, so holding the layout still means
dropping the profile entirely — and the code being compared is other people's,
run against a writable mount of the repository, so the profile is worth keeping
until a run actually needs the addresses held still:

```
PRESCRIPT_FIX_ADDRESS_LAYOUT=1 make docker_test
```

Without it the entrypoint says on stderr that addresses will move, and carries
on. A container that will not start is worse than one that is slightly less
repeatable, and this matters only for a runtime that mixes a pointer into a
hash or a seed.
