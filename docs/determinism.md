# Determinism

A program that draws random numbers prints something different every time it
runs, and a script that says what it should print is therefore wrong the moment
it is written. Most of the programs this tool exists to test are games, and
most games roll dice, so this is not an edge case — it is the majority of the
corpus.

The fix is to control the language, not the program. A port should not have to
be edited to be testable: the moment a test requires changing the thing under
test, it is no longer testing the thing that shipped.

There are two levels of that, and they answer different questions. A **seed**
makes one port repeatable, so a script written against it keeps passing. A
**tape** makes every port draw the *same* numbers, so one script can be written
once and played against all of them.

## What a runner does

Every language has some hook that runs code before the program does. A runner
sets the environment variable that activates it and points it at a small shim
file kept beside the runner:

```yaml
# runners/ruby.yaml
version: "0.8"
executable: ruby
env:
  RUBYOPT: "-r${runnerDir}/ruby/random.rb"
  PRESCRIPT_SEED: "0"
inheritEnv: true
```

```ruby
# runners/ruby/random.rb, in outline
if ENV["PRESCRIPT_TAPE"]
  # replay recorded values, in order
else
  srand(Integer(ENV.fetch("PRESCRIPT_SEED", "0")))
end
```

`${runnerDir}` is what makes that path work from any directory; see
[script-format.md](script-format.md#runnerdir). Every shim reads the same
`PRESCRIPT_SEED` and the same `PRESCRIPT_TAPE`, so one variable steers a whole
corpus run.

Play a script through one and the program is untouched:

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml -- 33_Dice/ruby/dice.rb
```

## What ships

| Language | Hook | Reaches | Takes a tape |
| --- | --- | --- | --- |
| Ruby | `RUBYOPT=-r<shim>` | `rand`, `Kernel#rand` | yes |
| Python | `sitecustomize.py` on `PYTHONPATH` | the `random` module's functions | yes |
| Node | `NODE_OPTIONS=--require <shim>` | `Math.random` | yes |
| Java | `-javaagent:<jar>` rewriting call sites | `Math.random`, `Random`, `ThreadLocalRandom` | yes |
| .NET | a `System.Random` compiled in ahead of the real one | everything on `Random`, including `Shared` | yes |
| Go | `GODEBUG=randautoseed=0` | `math/rand`'s global functions | no |

Java and .NET are the two that reach the program by getting in front of the
compiler rather than by setting a variable the runtime reads, because neither
runtime has a variable that would do.

The Java agent rewrites the port's own bytecode as each class loads: a call to
`Random.nextInt()` becomes a call to the shim, which returns the next tape value
and ignores the generator it was called on. It rewrites the port and leaves the
JDK alone, so no platform class is patched and no retransformation happens. The
agent is a jar, and the only shim that has to be built before it can be used:

```
make shims
```

The .NET shim is a source file declaring `namespace System { public class Random }`.
The C# compiler prefers a type it is compiling over the same type from a
reference, so the port's `new Random()` binds to the shim without the port
changing a line. `runners/dotnet.yaml` adds that file to the compilation with an
MSBuild targets file, which works the same for a single `.cs` file as for a
`.csproj`. It is the widest of the shims by accident of the mechanism: replacing
the class replaces `Shuffle`, `GetItems` and `Random.Shared` along with the draws.

Go seeds but cannot be taped. `GODEBUG` turns off the automatic seeding that
Go 1.20 introduced, which is enough to make one build repeatable, but there is
no exported way to put a different source behind `math/rand`'s package-level
functions from outside the program. Replaying values to Go needs the build to
cooperate — a `go build -overlay`, or a vendored source — which is the same
wall the compiled languages below run into.

Each is verified by a test that plays a program drawing numbers, twice, and by
a second test asserting the same script fails without the runner — so a shim
that quietly stopped working would be caught as a failure rather than as a
suite that still passes while checking nothing.

### What they do not reach

A shim seeds the language's default generator. It does not reach a generator a
program constructs for itself, or one that seeds itself deliberately:

- Ruby: `Random.new(...)`, `SecureRandom`
- Python: `random.Random(...)` instances, `secrets`, `numpy`
- Node: `crypto.randomBytes`, `crypto.getRandomValues`
- Java: `SecureRandom`, `SplittableRandom`, generators built by
  `RandomGeneratorFactory`, and draws made from inside `java.base` rather than
  from the port — `Collections.shuffle(list, rnd)` and `rnd.ints()` call
  `nextInt` at a call site the agent does not rewrite
- .NET: `System.Security.Cryptography`, `Guid.NewGuid`, and any draw made
  inside a pre-compiled dependency, which was compiled against the real
  `Random` before the shim existed
- Go: `math/rand/v2`, which has no global seed and ignores `GODEBUG` — verified,
  not assumed

A port using one of those needs either its own runner or a
[redaction](script-format.md#redactions) for the values it prints.

Node and .NET are replacements rather than seedings. V8's `Math.random` cannot
be seeded — there is no API, and the engine takes entropy when a context is
created — and the .NET shim has replaced `Random` outright by the time the port
asks for one, so there is nothing left underneath to seed. Both substitute
mulberry32, small enough to read in one sitting and identical on every
platform.

### Not yet shipped

| Language | Mechanism | Why it is not here |
| --- | --- | --- |
| Rust | `[patch.crates-io] getrandom` in `.cargo/config.toml` | build-time, not launch-time |
| Haskell | vendored `random` with `initStdGen = pure (mkStdGen n)` | build-time, not launch-time |
| C / C++ | `-include shim.h`, or a `rand.o` linked ahead of libc | build-time, not launch-time |

These are all the same problem: a compiled language's randomness is decided when
it is built, and a runner only gets to speak at launch. Java and .NET are in the
table above rather than in this one because both compile on the way to running,
which gives a runner a compilation to join; a Rust port arrives as a binary, or
as a build a runner would have to drive.

An untested runner is worse than a missing one — it makes a corpus look seeded
when it is not — so these are listed rather than guessed at.

### Why not `LD_PRELOAD`

It looks like the general answer and is not:

- macOS blocks it under SIP for anything system-provided
- it misses statically linked binaries entirely, and Go and Rust make the
  `getrandom` syscall directly rather than calling libc
- where it does work it gives determinism without cross-language identity,
  which is the same place per-language shims already get to

## What a seed does not buy

The same seed does not mean the same numbers. Each language draws a different
sequence from `PRESCRIPT_SEED=0`, because each has a different generator
underneath. The test fixtures show it plainly — one program, one seed:

| Language | First three draws |
| --- | --- |
| Ruby | `548 715 602` |
| Python | `844 757 420` |
| Node | `358 105 675` |
| Java | `730 240 637` |
| .NET | `358 105 675` |
| Go | `604 940 664` |

.NET agrees with Node because both shims substitute the same generator, for the
reason given above. Java's row is the JDK's own LCG, seeded the ordinary way.
That one agreement is a fact about two shims rather than a property anything
should lean on — a tape is how ports are made to agree on purpose.

So a seed makes each port repeatable, and one expected-output file still cannot
serve them all. That is what a tape is for.

## Tapes

A tape treats randomness as an input channel rather than as a property of the
language: the numbers are recorded once from the reference implementation, and
then read back by every port in turn.

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml \
  --tape tapes/vintbas-seed0.tape -- 33_Dice/ruby/dice.rb
```

`--tape` sets `PRESCRIPT_TAPE` for the child, and the shim that is already
loaded switches from seeding to replaying. Nothing else about the run changes,
and the port is still untouched.

The same tape through the Ruby, Python, Node, Java and .NET runners gives the
same transcript — which is the whole point, and is what
`TestOneTapeGivesEveryLanguageTheSameNumbers` checks.

### Where the numbers come from

`tapes/vintbas-seed0.tape` holds 10,000 values drawn by the reference BASIC
interpreter itself. No instrumentation was needed: a BASIC program that does
nothing but `PRINT RND(1)` in a loop reads the interpreter's sequence straight
out.

```
10 FOR I = 1 TO 10000
20 PRINT RND(1)
30 NEXT I
```

Haskell's `show` for `Float` prints the shortest decimal that reads back as the
same single-precision value, so the printed text is a lossless recording rather
than a rounded one — a port parsing `.9732723` recovers the exact float the
interpreter drew.

Regenerate it with `scripts/record-tape.sh [count] [seed]`, which pins the
interpreter release and the `random` package version it was built against
(`random` 1.2 replaced the generator behind `StdGen`, so an unpinned rebuild
would quietly record a different sequence).

The file is plain text, one value per line, with `#` comments carrying the
provenance:

```
# prescript tape 1
# source: vintage-basic 1.0.3, built as vintbas-1.0.3-1
# random: 1.2.1.2
# seed: 0
# values: 10000
.9732723
.41177148
```

### Running off the end is a failure

A shim that reaches the end of the tape reports it and exits non-zero:

```
prescript: the tape ran out after 10000 values; this port draws more
randomness than the reference did
```

It does not wrap. A port that draws more than the reference did has
restructured how it consumes randomness, and that divergence is itself the
finding — the thing seeding hides behind numbers that are stable but different.
Starting the tape again would turn a clear answer into a transcript that merely
looks wrong somewhere further down.

For the same reason a shim writes down how much it used. `PRESCRIPT_TAPE_USAGE`
names a path, and the file holds the number of values drawn — which is what
separates "this port has a logic bug" from "this port draws in a different
order", and is one of the two axes a divergence is
[classified](comparison.md#what-the-tape-says) by.

`prescript play --tape` sets that variable for itself, so the count is
collected on every taped run rather than only when somebody remembered to ask.
A run that names the variable itself keeps its own path.

The count is rewritten after every draw rather than once on the way out. What
matters is how much had been drawn when the run stopped agreeing with the
transcript, and a run that diverged by hanging is killed — which fires no exit
handler in any of these languages. Rewriting in place needs no truncation: the
count only grows, so its decimal form only gets longer and each write covers
the last.

The file is also written before the first draw, so a port that drew nothing
says so. Nothing at all is written when there is no tape, and unknown must not
be read as zero: a port that drew nothing has said randomness is not implicated
in its divergence, and one that cannot say has said nothing.

### What a tape does not fix

A tape gives every port the same *numbers*. It does not make them the same
*program*: a port that rounds differently, iterates a collection in a different
order, or asks for randomness in a different place will still diverge — and
should, because that is a real difference between the ports.

It also only reaches what the shims reach. Everything in
[What they do not reach](#what-they-do-not-reach) is as invisible to a tape as
it is to a seed.

## Randomness is not the only thing that moves

Transcripts also diverge on locale, timezone, terminal size, hash seeds and
address-space layout, and in practice those cause more churn than the RNG does.
`runners/python.yaml` sets `PYTHONHASHSEED=0` for that reason — a program that
prints a set is otherwise a different program on every run whatever its
generator does — but the rest belongs in a pinned image rather than in a
runner, and that is what `docker/` is: see [container.md](container.md).

Those runners set `inheritEnv: true`, which is a compromise and worth naming: a
Ruby installed by rbenv, a Python by pyenv or a Node by nvm all find their own
standard library through the environment, so replacing it wholesale would make
these runners work only where the language came from the system. Reproducibility
says the child should get exactly what is declared and nothing else. Pinning the
environment properly is a container's job, and until there is one, inheriting is
the honest trade.
