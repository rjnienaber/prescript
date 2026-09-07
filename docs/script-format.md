# Script format

A script file describes what an interactive program prints and what should be
typed back at it. `prescript record` writes one; `prescript play` replays it.

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
version: script is written for format 0.2, but this build of prescript reads up to 0.1; upgrade prescript
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

### Known limitation

The schema validates the union of all fields across known minor versions, so a
script declaring `0.1` that uses a field introduced in `0.2` is accepted rather
than rejected. Version-specific schemas would catch it, at a cost in machinery
that is not yet worth paying. The version declares intent; the schema checks
shape.
