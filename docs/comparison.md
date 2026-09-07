# Comparing implementations

A script with one run answers a question about one program. A script with more
than one run is asking a different question — *where do these implementations
stop agreeing?* — and the answer to that is not a verdict on each of them
separately. Playing ten ports and printing ten failure reports leaves the
reader to do the comparison by hand, which is the work the script was written
to do for them.

So a multi-run script ends with a comparison: every port lined up against the
reference at the step where it left it.

```
=== comparison against basic ===

step 1 of 2 (exited-early)
  basic         "YOU HAVE 3 GUESSES LEFT"
  ruby, python  "You have 3 guesses left"
                "TRY AGAIN"

step 2 of 2 (exited-early)
  basic         "TRY AGAIN"
  node          ""

matched: perl
```

The comparison is written to stderr after the last run, and does not replace
the individual failure reports above it — those say *what* each run did, in
full, and this says how the runs relate to each other.

## The first run is the reference

Not a rule imposed on the format so much as a description of how such a script
comes about: it is recorded from the implementation whose behaviour is being
treated as correct, and the ports are added underneath.

The reference is played like every other run, which matters. If it no longer
matches its own transcript then the environment has moved — a different
interpreter build, a different locale, a tape that no longer fits — and every
port's "divergence" is being measured against something that is itself wrong.
When that happens nothing is compared:

```
=== comparison against basic ===

basic did not match its own transcript, so there was nothing for the
ports to be compared against. Its own failure is above.
```

## Ports that got the same thing wrong are one finding

Ports are grouped by where they diverged, how, and what they printed instead.
They are not independent observations: a dozen ports of the same BASIC program
missing the leading space BASIC prints before a positive number is one bug
repeated twelve times, and a report that says it twelve times is a report
nobody reads to the end.

Findings are ordered by step, earliest first. A later divergence is often a
consequence of an earlier one — the port that got the prompt wrong is the port
that then answered the wrong question — so the first one is the one to read,
and it should not be somewhere in the middle because of which port happened to
hit it.

Every quoted line uses the same column across the whole report, so the
differences are read down the page rather than re-found at each step.

## What is shown against the reference

The reference's expected line, and then everything the ports printed since they
last agreed with the transcript — not one chosen line.

Which of several lines was *meant* to be the step is exactly the question a
divergence raises, so the report does not answer it. When a port printed one
line, there is nothing to choose between and the caret points at the byte where
it went wrong:

```
step 1 of 2 (no-match)
  basic  "HOW MANY ROLLS? "
  node   "HOW MANY ROLLS?"
                         ^ received output ends here
```

When it printed several, they are all shown and none is pointed at.

A step matched through [redactions](script-format.md#redactions) shows the
expected line as written, marked `(with redactions applied)`, and gets no
caret: the difference is against a pattern, and a column number would name a
position in the pattern rather than in the output.

A run that matched every step and then went wrong — `hung`, or
`wrong-exit-code` — has no line left to put opposite the reference, so it says
what it did instead of finishing:

```
after all 12 steps (hung)
  ruby  the executable had not exited after 30s
```

## Classification

Not every divergence is a bug worth filing, and two things about one can be
worked out without a person reading it. Both appear as fixed tokens in the
finding's header, next to the failure mode:

```
step 5 of 12 (no-match, formatting, no-draws)
```

They are tokens rather than prose so that a few hundred runs can be counted and
sorted by them, and what is *not* known is left out rather than named — a token
that appears on every line carries nothing.

### Formatting or semantic

Collapse every run of whitespace to one space, trim the ends, and compare
again. If the two sides now agree, the divergence is `formatting`; otherwise it
is `semantic`.

This is worth separating for a reason beyond tidiness. BASIC's `PRINT` puts a
space before a non-negative number and another after it, and `,` tabs to
fourteen-column zones. Ports get this wrong constantly and identically, and
filing each one separately would mean several hundred near-identical issues
against the same repository — which would be ignored, reasonably. Formatting
divergences belong together, one report per language; a semantic one is about a
particular program and belongs on its own.

The received lines are joined before normalising, so a port that broke one
logical line across two prints is `formatting` — where a line breaks is layout
by any reading. Case is not touched: a port that shouts where the reference
whispers has changed the text, not its layout, and is `semantic`.

A step matched through a [redaction](script-format.md#redactions) gets neither
token. The expected side is a pattern rather than a line, so there is nothing
to normalise and no honest answer to give.

### What the tape says

When a run is played with `--tape`, prescript asks the shim to record how many
values it drew, and reads the count back afterwards — no variable to set, and
the number is what had been drawn when the run *stopped*, not when it exited,
so a run killed for hanging still reports one.

| Token | What it means | What to do |
| --- | --- | --- |
| `no-draws` | the ports had taken nothing from the tape | randomness is not involved; file it |
| `same-draws` | the ports and the reference drew the same amount | they asked in the same places and disagreed about the answer; file it |
| `different-draws` | known counts that disagree, with the reference or with each other | the port consumes randomness in a different order; a person has to judge it |
| *absent* | not enough counts are known | — |

Two ports that printed the same thing at the same step having drawn different
amounts is itself a finding, and says so without the reference.

**The reference interpreter does not report its draws.** Vintage BASIC has no
shim — a tape is *recorded from* it rather than replayed into it — so on a
corpus run the reference's count is unknown and `same-draws` is out of reach.
What survives is `no-draws`, which needs only the port, and disagreement
between ports. The full table applies when the reference is itself a taped
port, such as comparing Ruby against Python. Instrumenting the interpreter is
[#40](https://github.com/rjnienaber/prescript/issues/40).

## Ports that were not compared

Three things are not divergences, and saying they were would be the kind of
confident wrong answer this tool exists to catch:

| Line | What it means |
| --- | --- |
| `matched: ...` | played the reference's transcript and agreed with all of it |
| `not comparable, they play a different transcript: ...` | its steps are not the reference's steps |
| `could not be played: ...` | it never ran — a missing executable, an unreadable tape |

A run is comparable when it expects the same lines and sends the same input.
Everything else about it — which executable, which arguments, which
environment, which terminal — is what the comparison is *about*, and two runs
identical in those respects would have nothing to say to each other.

Step 7 of one transcript has nothing to do with step 7 of another, so lining up
two different transcripts would report a difference that is an artefact of the
script rather than of the implementations. Two runs that legitimately expect
different output belong in two scripts.

## Exit codes are unchanged

`prescript` still exits with the first non-zero code any run produced. The
comparison is a report, not a verdict: it says where the implementations parted
company, and what to do about that is the reader's call.

## Filing a finding

`--bug-report <file>` writes the earliest finding out as Markdown, with the
interpreter version, the container digest, the tape and everything else needed
to answer "works for me". See [bug-reports.md](bug-reports.md).
