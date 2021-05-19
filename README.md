# prescript
[![License](https://img.shields.io/github/license/rjnienaber/prescript)]()

### About
`prescript` is an automation tool to run other interactive clis and respond to their output. Using 
a script file, it will watch the output of a cli application, responding with its own input. 

### Current Status: <span style="color: red; font-weight: bold">Early alpha</span>

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
automation script should finish as quickly as the cli can execute it's work.

### Commands & Options
#### `play`

Runs prescripted responses against an interactive cli

```bash
prescript play [script file] [optional executable] [flags]
```

| Option | Description                                                               | Type   | Default | Required? |
| ------ | ------------------------------------------------------------------------- | ------ | ------- | --------- |
| `[script file]`         | the script to use that contains the automated steps      | `bool` |         | Yes       |
| `[optional executable]` | an executable to run the script against                  | `bool` |         | No        |
| `-d`                    | dont fail on external command failures                   | `bool` | `false` | No        |
| `-l`                    | log level to use with logs (e.g. none, debug, info)      | `enum` | `none`  | No        |
| `-q`                    | no output                                                | `bool` | `false` | No        |
| `-t`                    | timeout waiting for output from external command         | `bool` | `30s  ` | No        |
