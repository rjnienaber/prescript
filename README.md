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
| `-d`                    | dont fail on external command failures                   | `bool` | `false` | No        |
| `-l`                    | log level to use with logs (`none`, `error`, `info` or `debug`) | `enum` | `none`  | No        |
| `-q`                    | no output                                                | `bool` | `false` | No        |
| `-t`                    | timeout waiting for output from external command         | `bool` | `30s  ` | No        |
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

**N.B.** The `--` convention is used to stop processing arguments for `prescript`. Any arguments after
this point are passed to the executable.

### When a run fails

`prescript` exits non-zero and writes a report to stderr saying which step went
wrong and what arrived instead:

```
step 1 of 2 did not match (timed out after 5s)

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
without hiding the diagnosis, and it does not depend on `--log-level`.

### Development

```
make dependencies   # build tooling, plus the vintbas interpreter the examples run against
make prepush        # format, lint, test, build, examples
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

### Get in touch
* Slack: Find me as @rjnienaber on https://gophers.slack.com/ 
* Twitter: [@rjnienaber](https://twitter.com/rjnienaber)
