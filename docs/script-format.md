# Script format

A script file describes what an interactive program prints and what should be
typed back at it. `prescript record` writes one; `prescript play` replays it.

It may be written in JSON or in YAML — see [File formats](#file-formats).

## Versioning

Every script declares a `version`, of the form `MAJOR.MINOR`:

```json
{
  "version": "0.1",
  "runs": [ ... ]
}
```

### What each part means

**MINOR** increases for an additive change — typically a new optional field.
Scripts written against an earlier minor of the same major keep working with no
edits, so there is nothing to migrate.

**MAJOR** increases when existing scripts have to be rewritten to keep working.
When that happens, the older major is dropped from the list of readable
versions and the error names the last release that could read it.

### What `play` does with it

`prescript` reads a fixed list of versions (`knownVersions` in
`internal/script/version.go`) and **rejects anything else**, including a version
newer than its own. It does not warn and continue.

That is the important half of the rule, and it is deliberate. The fields a minor
version adds are precisely the ones that change what a run *means*: an
environment variable, a redaction rule, a second run to compare against. A
reader that skipped what it did not recognise would not fail — it would quietly
do something else and report success. Confidently reporting a wrong answer is
the failure this tool exists to catch in other people's programs, so it is not
one prescript may commit itself.

A newer version says what to do about it:

```
Script validation errors:
version: script is written for format 9.9, but this build of prescript reads up to 0.2; upgrade prescript
```

which is a considerably more useful thing to read than the list of unrecognised
field names the same script would otherwise produce.

### Adding a version

In the same change that alters the schema:

1. Append the new version to `knownVersions`. `CurrentVersion` follows the last
   entry, and `record` writes it.
2. Add the field to `script_schema.json` as optional, and to the Go type.
3. Leave older versions in the list. They are still readable, and removing one
   is a major bump.

Do not add a version ahead of the change that needs it: a version that exists
but means nothing is worse than no version at all.

### Version history

| Version | Change |
| --- | --- |
| `0.1` | Initial format. |
| `0.2` | Added `env` and `inheritEnv` to a run. |
| `0.3` | A script may hold more than one run. |

### Known limitation

The schema validates the union of all fields across known minor versions, so a
script declaring `0.1` that uses a field introduced in `0.2` is accepted rather
than rejected. Version-specific schemas would catch it, at a cost in machinery
that is not yet worth paying. The version declares intent; the schema checks
shape.

## Environment

A run may declare the environment its program is started with:

```json
{
  "version": "0.2",
  "runs": [{
    "executable": "vintbas",
    "arguments": ["dice.bas"],
    "env": { "VINTBAS_SEED": "0", "TZ": "UTC" },
    "exitCode": 0,
    "steps": [ ... ]
  }]
}
```

The rule is that **declaring an environment declares all of it**:

| `env` | child process gets |
| --- | --- |
| omitted | prescript's own environment, unchanged |
| `{ "A": "1" }` | exactly `A=1`, and nothing else |
| `{ "A": "1" }` with `"inheritEnv": true` | prescript's environment, with `A=1` set over it |
| `{}` | an empty environment |

Omitting `env` is what every script written before `0.2` does, so those keep
behaving exactly as they did. Anything that declares one gets the reproducible
reading: a run that names `TZ` and `VINTBAS_SEED` is not also quietly depending
on the `LANG` that happened to be exported in the shell that started it.

`inheritEnv` is the escape hatch for a program that needs the caller's
environment — a compiler toolchain, say — plus a couple of variables of its
own. Declared values are applied over inherited ones.

### `PATH` in a replaced environment

A run that replaces its environment does not necessarily leave the child with no
`PATH`: `execvp` and most shells fall back to a compiled-in default when the
variable is unset, so `#!/usr/bin/env bash` generally still resolves and so do
the common tools.

That fallback is not something to rely on, because its contents differ by
platform — on Linux it is typically just `/bin:/usr/bin`, which excludes
`/usr/local/bin`, where a tool the program shells out to may well live. If the
program needs to find anything, set `PATH` in `env` explicitly. Depending on a
per-platform default is the machine-dependence this feature exists to remove.

Resolving the executable named by the script is unaffected either way: prescript
looks it up on *its own* `PATH` before starting the child, so a script naming
`vintbas` finds the same binary whether or not the run declares an environment.

## Multiple runs

`runs` is an array, and a script may hold more than one. This is how two
implementations of the same program are compared: each run names its own
executable, and they share the steps they are expected to produce.

```json
{
  "version": "0.3",
  "runs": [
    { "name": "reference", "executable": "vintbas", "arguments": ["dice.bas"], ... },
    { "name": "port", "executable": "python", "arguments": ["dice.py"], ... }
  ]
}
```

Every run is played, even after one has failed. Stopping at the first failure
would withhold the comparison the script was written to make: knowing the first
of three ports is wrong says nothing about the other two. `prescript` exits with
the first non-zero code and, when there was more than one run, ends with a line
naming which of them failed.

### Names

Each run has a `name`, which is what identifies it in the output. A run without
one is called `run N` after its position. Two runs may not end up with the same
name, and generated names count: a run explicitly called `run 1` collides with
the unnamed run at index 1.

### Overriding a multi-run script from the command line

`--exec` and the arguments after `--` apply to *every* run. That is meaningful
when the runs differ in their steps and a mistake when they differ in their
executable, and prescript cannot tell which was meant, so it warns rather than
refusing.

## File formats

`prescript play` reads a script as YAML when the file ends in `.yaml` or `.yml`,
and as JSON otherwise. `prescript record` writes JSON.

YAML is converted to JSON and then validated by the same schema against the same
Go types. There is one description of the format and one set of error messages,
so the two notations cannot drift apart, and a YAML script is rejected in the
same terms — including the same `runs.0.exitCode` style paths — as the JSON it
converts to.

YAML is the better authoring format, and the fixtures are meant to be written in
it. It has comments, and the expected output is not buried in quoting.

### Quote every expected line

```yaml
steps:
  - line: "HOW MANY ROLLS? "
    input: "5000"
```

YAML strips trailing whitespace from an unquoted scalar. A prompt almost always
ends in a space, and a step that differs from the program's real output only in
that space is the most annoying possible failure to read: the two look identical
everywhere except in the report, which quotes them precisely because of this.

Quote `input` too. An unquoted `5000` is a number, an unquoted `N` is a string,
and an unquoted `y`, `no` or `on` is a boolean in YAML 1.1 readers — none of
which is what a program reading from a terminal receives.

### Keys have to be strings

YAML allows a mapping key of any type; JSON does not. A script that uses one is
rejected, rather than reaching the schema as something it cannot describe.
