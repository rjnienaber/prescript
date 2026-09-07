# Determinism

A program that draws random numbers prints something different every time it
runs, and a script that says what it should print is therefore wrong the moment
it is written. Most of the programs this tool exists to test are games, and
most games roll dice, so this is not an edge case — it is the majority of the
corpus.

The fix is to seed the language, not the program. A port should not have to be
edited to be testable: the moment a test requires changing the thing under
test, it is no longer testing the thing that shipped.

## What a runner does

Every language has some hook that runs code before the program does. A runner
sets the environment variable that activates it and points it at a small shim
file kept beside the runner:

```yaml
# runners/ruby.yaml
version: "0.8"
executable: ruby
env:
  RUBYOPT: "-r${runnerDir}/ruby/seed.rb"
  PRESCRIPT_SEED: "0"
inheritEnv: true
```

```ruby
# runners/ruby/seed.rb
srand(Integer(ENV.fetch("PRESCRIPT_SEED", "0")))
```

`${runnerDir}` is what makes that path work from any directory; see
[script-format.md](script-format.md#runnerdir). Every shim reads the same
`PRESCRIPT_SEED`, so one variable seeds a whole corpus run.

Play a script through one and the program is untouched:

```
prescript play scripts/33-dice.yaml --runner runners/ruby.yaml -- 33_Dice/ruby/dice.rb
```

## What ships

| Language | Hook | Reaches |
| --- | --- | --- |
| Ruby | `RUBYOPT=-r<shim>` → `srand` | `rand`, `Kernel#rand` |
| Python | `sitecustomize.py` on `PYTHONPATH` → `random.seed` | the `random` module's functions |
| Node | `NODE_OPTIONS=--require <shim>` → replaces `Math.random` | `Math.random` |
| Go | `GODEBUG=randautoseed=0` | `math/rand`'s global functions |

Each is verified by a test that plays a program drawing three numbers, twice,
and by a second test asserting the same script fails without the runner — so a
shim that quietly stopped working would be caught as a failure rather than as a
suite that still passes while checking nothing.

### What they do not reach

A shim seeds the language's default generator. It does not reach a generator a
program constructs for itself, or one that seeds itself deliberately:

- Ruby: `Random.new(...)`, `SecureRandom`
- Python: `random.Random(...)` instances, `secrets`, `numpy`
- Node: `crypto.randomBytes`, `crypto.getRandomValues`
- Go: `math/rand/v2`, which has no global seed and ignores `GODEBUG` — verified,
  not assumed

A port using one of those needs either its own runner or a
[redaction](script-format.md#redactions) for the values it prints.

Node is the one replacement rather than a seeding: V8's `Math.random` cannot be
seeded — there is no API, and the engine takes entropy when a context is
created — so the shim substitutes mulberry32, small enough to read in one
sitting and identical on every platform.

### Not yet shipped

| Language | Mechanism | Why it is not here |
| --- | --- | --- |
| .NET | `DOTNET_STARTUP_HOOKS` | no toolchain to verify it against |
| Java | `-javaagent` instrumenting `Math.random` / `Random` | no toolchain to verify it against |
| Rust | `[patch.crates-io] getrandom` in `.cargo/config.toml` | build-time, not launch-time |
| Haskell | vendored `random` with `initStdGen = pure (mkStdGen n)` | build-time, not launch-time |
| C / C++ | `-include shim.h`, or a `rand.o` linked ahead of libc | build-time, not launch-time |

The first two are runners waiting for a machine to test them on. The last three
are a different problem: a compiled language's randomness is decided when it is
built, and a runner only gets to speak at launch. Those need the build to
cooperate, which is what makes the pinned image in the container work
(issue #15) the place they belong.

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

The same seed does not mean the same numbers. Ruby, Python, Node and Go each
draw a different sequence from `PRESCRIPT_SEED=0`, because each has a different
generator underneath. The test fixtures show it plainly — one program, one
seed, four answers:

| Language | First three draws |
| --- | --- |
| Ruby | `684 559 629` |
| Python | `864 394 776` |
| Node | `358 105 675` |
| Go | `81 887 847` |

So a seed makes each port repeatable, and one expected-output file still cannot
serve them all. That is what a recorded tape of random values is for
(issue #14): treat randomness as an input channel, record what the reference
BASIC consumed, and replay the same values to every port. When a port drains
the tape at a different rate, that divergence is itself the finding — seeding
hides exactly that behind numbers that are stable but different.

## Randomness is not the only thing that moves

Transcripts also diverge on locale, timezone, terminal size, hash seeds and
address-space layout, and in practice those cause more churn than the RNG does.
`runners/python.yaml` sets `PYTHONHASHSEED=0` for that reason — a program that
prints a set is otherwise a different program on every run whatever its
generator does — but the rest belongs in a pinned image rather than in a runner
(issue #15).

Those runners set `inheritEnv: true`, which is a compromise and worth naming: a
Ruby installed by rbenv, a Python by pyenv or a Node by nvm all find their own
standard library through the environment, so replacing it wholesale would make
these runners work only where the language came from the system. Reproducibility
says the child should get exactly what is declared and nothing else. Pinning the
environment properly is a container's job, and until there is one, inheriting is
the honest trade.
