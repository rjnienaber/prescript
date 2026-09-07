# prescript
[![Build](https://img.shields.io/github/workflow/status/rjnienaber/prescript/ci)]()
[![Dependencies](https://img.shields.io/librariesio/github/rjnienaber/prescript)]()
[![ReportCard](https://goreportcard.com/badge/github.com/rjnienaber/prescript)]()
[![License](https://img.shields.io/github/license/rjnienaber/prescript)]()

### About
`prescript` is an automation tool to run other interactive clis and respond to their output. Using 
a script file, it will watch the output of a cli application, responding with its own input. 

### Current Status: <span style="color: red; font-weight: bold">Early alpha</span>

### Installation

```
go get github.com/rjnienaber/prescript/cmd/prescript
```

### Example

```json
{
  "version": "0.1",
  "runs": [{
    "executable": "vintbas",
    "arguments": ["examples/dice/dice.bas"],
    "exitCode": 0,
    "steps": [{
      "line": "HOW MANY ROLLS? ",
      "input": "5000"
    }, {
      "line": "TRY AGAIN? ",
      "input": "N"
    }]
  }]
}
```

We'll execute a [BASIC computer program](https://github.com/coding-horror/basic-computer-games/tree/main/33%20Dice) 
called `dice.bas` by running it against the [Vintage BASIC interpreter](http://www.vintage-basic.net/download.html).
When executed by itself, it prompts the user twice: Once for the number of rolls and then again
to find out if the user wants to roll again. Both steps are automated in the above script and
the program exits successfully:

```
$ time ./prescript play examples/dice/dice.json 
                                  DICE
               CREATIVE COMPUTING  MORRISTOWN, NEW JERSEY



THIS PROGRAM SIMULATES THE ROLLING OF A
PAIR OF DICE.
YOU ENTER THE NUMBER OF TIMES YOU WANT THE COMPUTER TO
'ROLL' THE DICE.  WATCH OUT, VERY LARGE NUMBERS TAKE
A LONG TIME.  IN PARTICULAR, NUMBERS OVER 5000.

HOW MANY ROLLS? 5000

TOTAL SPOTS   NUMBER OF TIMES
 2             147 
 3             302 
 4             426 
 5             565 
 6             668 
 7             811 
 8             683 
 9             558 
 10            444 
 11            243 
 12            153 


TRY AGAIN? N

real	0m0.026s
user	0m0.028s
sys	0m0.000s
```

The script acts effectively like a state machine, with each step the next state that is being 
waited on. Since `prescript` doesn't use a timing mechanism to know when to input data, the 
automation script should finish as quickly as the cli can execute its work.

### Commands & Options
#### `play`

Runs prescripted responses against an interactive cli

```bash
prescript play [script file] [flags] -- [executable arguments]
```

| Option | Description                                                               | Type   | Default | Required? |
| ------ | ------------------------------------------------------------------------- | ------ | ------- | --------- |
| `[script file]`         | the script to use that contains the automated steps      | `bool` |         | Yes       |
| `-e`                    | override the executable named in the script file         | `string` |       | No        |
| `-r`                    | runner file describing how to launch an implementation   | `string` |     | No        |
| `-d`                    | dont fail on external command failures                   | `bool` | `false` | No        |
| `-l`                    | log level to use with logs (`none`, `error`, `info` or `debug`) | `enum` | `none`  | No        |
| `-q`                    | no output                                                | `bool` | `false` | No        |
| `-t`                    | timeout waiting for output from external command         | `bool` | `30s  ` | No        |
| `--terminal`            | what the executable is given for its standard streams (`pty` or `pipes`) | `enum` | `pty` | No        |
| `--tape`                | file of recorded random values to replay to the executable | `string` |   | No        |
| `-- [args]`             | arguments for the executable, overriding those in the script | `list` |     | No        |

#### `record`

Runs an interactive cli, records responses and generates a script for use with `play`

```bash
prescript record [script file] [executable] [flags] -- [args]
```

| Option | Description                                                               | Type   | Default | Required? |
| ------ | ------------------------------------------------------------------------- | ------ | ------- | --------- |
| `[script file]`         | the script to use that contains the automated steps      | `bool` |         | Yes       |
| `[executable]` | an executable to run the script against                  | `bool` |         | Yes        |
| `-d`                    | don't compress lines to match on                         | `bool` | `false` | No        |
| `--terminal`            | what the executable is given for its standard streams (`pty` or `pipes`) | `enum` | `pty` | No        |

**N.B.** The `--` convention is used to stop processing arguments for `prescript`. Any arguments after
this point are passed to the executable.

### Script files

Script files are JSON or YAML — `.yaml` and `.yml` are read as YAML, anything
else as JSON — validated against one schema, and declare a format
version. See [docs/script-format.md](docs/script-format.md) for what that
version means and how it changes.

A run can declare the environment its program starts with:

```json
"env": { "VINTBAS_SEED": "0", "TZ": "UTC" }
```

Declaring an environment declares all of it — the child gets those variables
and nothing else, so the run does not quietly depend on whatever the calling
shell happened to export. Add `"inheritEnv": true` to put prescript's own
environment underneath instead. Omit `env` and the child inherits, which is what
scripts written before format `0.2` do.

A script may hold more than one run, each naming its own executable, which is
how two implementations of the same program are compared. Every run is played
even after one fails, and `prescript` exits with the first non-zero code.

Write fixtures in YAML: it has comments, and the expected output is not buried
in quoting. Quote every `line` and `input` — YAML strips trailing whitespace
from an unquoted scalar, and prompts end in a space.

A line that varies between runs is matched with a named redaction, so the
expected line still reads as the output it stands for:

```yaml
redactions:
  elapsed: '[0-9]+'
steps:
  - line: "finished in {{elapsed}}ms"
```

### Terminals

The executable is run under a pseudo-terminal, because that is what it is
checking for when it decides how to behave. Under a pipe, C and Python switch
from line buffering to a 4KB block buffer, and a prompt with no trailing
newline can sit in that buffer while the program waits for the answer it has
not yet delivered — a timeout against a program that works fine. Colour and
progress bars are commonly switched off the same way.

prescript clears echo and newline translation on the pty, so what it reads is
the same stream a pipe would have given it. Use `terminal: pipes` in the script,
or `--terminal pipes`, when the pipe is the thing being tested. See
[docs/script-format.md](docs/script-format.md#terminal).

### Runners

A runner file says *how* to start an implementation; a script says *which*
program and what it should print:

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml -- 33_Dice/ruby/dice.rb
```

One runner per language and one script per program means a hundred programs in
a dozen languages is a hundred-odd files rather than twelve hundred. See
[docs/script-format.md](docs/script-format.md#runners).

### Determinism

Most of the programs worth testing this way are games, and games roll dice. A
program that draws random numbers prints something different on every run, so a
script saying what it should print is wrong the moment it is written.

`runners/` holds a runner per language that seeds the language's generator
before the program starts, so the port itself does not have to be edited to be
testable:

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml -- 33_Dice/ruby/dice.rb
```

Ruby, Python, Node and Go ship and are covered by tests.

A seed makes one port repeatable, but the same seed gives each language a
different sequence, so one script still cannot serve them all. `--tape` closes
that gap: it replays random values recorded from the reference BASIC
interpreter, so every port draws the same numbers and one expected transcript
fits all of them.

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml \
  --tape tapes/vintbas-seed0.tape -- 33_Dice/ruby/dice.rb
```

A port that runs off the end of the tape fails rather than wrapping, because
drawing more randomness than the reference did is itself the finding. What each
runner reaches, what it does not, and where the recorded numbers come from, is
in [docs/determinism.md](docs/determinism.md).

### Timeouts

`--timeout` bounds how long the program may go on saying nothing, and defaults
to 30 seconds. A step that is known to be slow names its own instead, so one
slow moment does not set the limit for the whole script:

```yaml
- line: "Done."
  timeout: "5m"
```

### When a run fails

`prescript` exits non-zero and writes a report to stderr. It leads with which
of the ways it went wrong — `no-match`, `exited-early`, `hung`,
`wrong-exit-code` or `read-failed` — and then says which step went wrong and
what arrived instead:

```
no-match: step 1 of 2 did not match (nothing matched it within 5s)

  expected  "HOW MANY ROLES? "
  received  "HOW MANY ROLLS? "
                         ^ first difference

  output since the last matched step:
    THIS PROGRAM SIMULATES THE ROLLING OF A
    PAIR OF DICE.

1 later step was never reached
```

Lines are quoted because trailing whitespace is load-bearing: prompts usually
end in a space, and an unquoted report makes a step that differs only in that
respect look identical to the one it failed to match.

The report goes to stderr, so `--quiet` still silences the program's own output
without hiding the diagnosis, and it does not depend on `--log-level`. Nothing
in it is measured, so the same divergence reports the same bytes every time;
see [docs/script-format.md](docs/script-format.md#failure-modes).

A run that times out kills the program before writing its report, and under a
pty takes anything the program started with it.

### Development

```
make dependencies   # build tooling, plus the vintbas interpreter the examples run against
make prepush        # format, lint, test, build, examples

make pull_image     # the pinned environment, as published
make image          # or build it locally
make docker_test    # the suite, inside it
```

`make examples` runs a BASIC program under Vintage BASIC, which seeds its random
number generator from the clock and so produces different output on every run.
`make vintbas` therefore installs a patched build, produced by
`scripts/build-vintbas.sh` from the Hackage release plus the patches in
`patches/`, that seeds from `$VINTBAS_SEED` (default `0`) instead. Set that
variable to walk a program through a different sequence.

Note that the sequence is also tied to the version of Haskell's `random`
package pinned in the build script: `random` 1.2 replaced the generator behind
`StdGen`, so the same seed yields different numbers either side of it. Changing
the pin invalidates any recorded output.

Randomness is not the only thing that moves. Interpreter versions, locale,
timezone, terminal size, hash seeds and address layout change what a program
prints without changing what it does, and a difference between two ports means
nothing if the two ran in different worlds. `docker/` holds a pinned
environment that fixes all of them, published to
`ghcr.io/rjnienaber/prescript-env` so CI and a developer run the same bytes
rather than the same recipe; see [docs/container.md](docs/container.md).

### Get in touch
* Slack: Find me as @rjnienaber on https://gophers.slack.com/ 
* Twitter: [@rjnienaber](https://twitter.com/rjnienaber)
