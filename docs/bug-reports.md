# Bug reports

```
prescript play dice.yaml --tape tapes/vintbas-seed0.tape --bug-report dice.md
```

`--bug-report` writes the first divergence to a file, as Markdown, with
everything somebody else needs to reproduce it.

## Why prescript writes it rather than you

The default answer to a cross-implementation bug report is "works for me", and
it is usually a fair answer. Two people running the same program in two
languages are also running two interpreter builds, two locales, and two
sequences of random numbers, and any one of those explains a difference on its
own. A report that does not pin all of them down cannot be acted on, only
argued with.

Pinning them down by hand is perhaps ten minutes of copying version strings out
of terminals. Nobody does it twice, so the second report is worse than the
first, and by the tenth the reports have stopped being filed at all. Everything
in the block below is something the run already knew about itself, so it is
written down rather than remembered.

## What it writes

````markdown
# ruby: no-match at step 5 of 12

In `dice.yaml`, against `basic`.
Classified `formatting, same-draws`.

## The divergence

```
step 5 of 12 (no-match, formatting, same-draws)
  basic  " 7             807 "
  ruby   "7 807"
          ^ first difference
```

This is the first divergence; 7 later steps were never reached.

## Reproducing it

```sh
make pull_image PUBLISHED_REF=ghcr.io/rjnienaber/prescript-env@sha256:2a1f...
prescript play dice.yaml --tape tapes/vintbas-seed0.tape --bug-report dice.md
```

## The environment

| | |
| --- | --- |
| prescript | `86fa312b39df` |
| script | `dice.yaml` |
| command | `ruby ports/ruby/dice.rb` |
| ruby | `ruby 3.3.5 (2024-09-03 revision ef084cc8f4) [x86_64-linux]` |
| image | `ghcr.io/rjnienaber/prescript-env@sha256:2a1f...` |
| tape | `tapes/vintbas-seed0.tape` |
| draws | `5` |
| terminal | `pty, 80x24` |
| timeout | `30s` |
| LC_ALL | `C.UTF-8` |
| TZ | `UTC` |

The tape says of itself:

```
# prescript tape 1
# source: vintage-basic 1.0.3, built as vintbas-1.0.3-1
# random: 1.2.1.2
# seed: 0
# values: 10000
```
````

The three sections are in the order they get read in. Somebody who already
knows the bug stops after the first; somebody about to say "works for me" is
looking for the third.

## The first divergence only

A run that went wrong at step 5 was not really running the same program from
step 6 onwards. Everything after the first difference is a consequence of it,
and a report that lists all of them is asking its reader to work out which one
was the cause — which is the work the report was supposed to do.

Which divergence that is depends on what was played:

- **A script with several runs.** The earliest finding in the
  [comparison](comparison.md): the first step at which any port left the
  reference, and every port that left it there in the same way. Findings are
  grouped, so a report about one bug in twelve ports is one report naming
  twelve ports.
- **A script whose reference failed.** The reference's own divergence. That is
  the more urgent finding rather than a reason to write nothing: the
  environment has moved under a script that used to pass, and the ports were
  never compared against anything.
- **A single run.** Its own failure, reported against `the transcript`. There is
  no implementation being treated as correct, and naming one that was never
  played would misdescribe where the expected side came from.

When every run matched there is nothing to write, and prescript says so instead
of writing an empty file.

## What is in the facts table, and why

Each row is there because it has been the answer to a "works for me" at least
once.

| Row | Why it is there |
| --- | --- |
| `prescript` | The revision that produced the report, from the binary's own build info. Reports outlive releases. |
| `script` | Which script, since one corpus has hundreds. |
| `command` | The executable and arguments the run actually used, after any runner and any `--exec`. |
| the interpreter | What the toolchain says it is. See below. |
| `image` | The pinned digest, which is what makes everything else reproducible. |
| `tape` or `seed` | Where the randomness came from. Without it the report is not reproducible even in principle. |
| `draws` | How much of the tape the port had used when it stopped. Zero means randomness is not implicated at all. |
| `terminal` | `pty, 80x24` or `pipes`. Programs lay their output out differently depending on the answer, and a size is not something a reader can guess. |
| `timeout` | What a timeout meant on this run. |
| `LC_ALL`, `LANG`, `TZ` | The three variables most likely to change a program's output between two machines that are otherwise identical. |

The locale and timezone are read from the environment the program was actually
started with, which is the run's own when it
[declared one](script-format.md) and prescript's otherwise.

### Asking the interpreter what it is

The executable is run once more with `--version` (`version` for `go`), and the
first line of its output becomes a row. Only an executable named by name is
asked. One named by a path — `./dice`, `ports/ruby/dice.rb` — is the program
under test rather than the toolchain under it, and running somebody else's
program with an argument it never expected is not a thing to do behind their
back. The probe is bounded at five seconds and omitted entirely if it fails, so
it cannot become a second way for a failed run to hang.

### The image

A run outside the [pinned container](container.md) says so, in the table, in
those words:

```
| image | `none; this did not run in the pinned environment` |
```

That is a fact worth stating rather than a gap worth leaving. It is the single
likeliest reason the reader cannot reproduce what they are being shown.

The `make pull_image` line appears in the repro block only when the digest is
one somebody else can pull. A locally built image is identified by its config
digest — a bare `sha256:` with no name in front of it — which says which bytes
ran but cannot be fetched, so it is named in the table and left out of the
commands.

## The repro command

The command line is reproduced as it was given, with two changes:

- `argv[0]` becomes `prescript`. How this machine reaches the binary is not
  part of the bug.
- Absolute paths inside the working directory become relative to it. This makes
  the command work from a fresh checkout, and it keeps a home directory out of
  a report that is about to be posted in public.

Paths outside the working directory are left alone: shortening them would
produce a command that does not work. If a report is going somewhere public,
this is the line to read before pasting it.

## What it does not do

It does not file anything. The output is a file; where it goes is a decision
with a person in it.

It does not change the exit code. Whether the runs passed was decided by the
runs. A report that could not be written is said out loud on stderr — the run
it described has already finished, and will not be repeated for free — but it
is not itself a failure of the run.
