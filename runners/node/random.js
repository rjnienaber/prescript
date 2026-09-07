// Makes a Node port's randomness repeatable, in one of two ways.
//
// With PRESCRIPT_TAPE set, Math.random returns values recorded from the
// reference BASIC interpreter, in order, so this port and every other port of
// the same program draw the same numbers. Without it, Math.random becomes a
// seeded generator of its own, which makes this port repeatable but leaves it
// drawing a different sequence from every other language. See
// docs/determinism.md.
//
// Activated by NODE_OPTIONS=--require <this file>, which Node runs before the
// program, so the program does not have to know it exists and does not have to
// be edited. It reaches ESM programs too: --require runs ahead of the module
// loader, and Math is the same object either way.
//
// V8's own Math.random cannot be seeded -- there is no API, and the engine
// takes entropy when a context is created -- so both modes replace it rather
// than seed it.
//
// What this does not reach: crypto.randomBytes, crypto.getRandomValues, and
// anything a dependency bundles for itself.
"use strict";

const tapePath = process.env.PRESCRIPT_TAPE;

if (tapePath) {
  const fs = require("node:fs");

  const values = fs
    .readFileSync(tapePath, "utf8")
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line !== "" && !line.startsWith("#"))
    .map(Number);

  let drawn = 0;

  // How much of the tape a port used is the number that separates a real logic
  // bug from a port that restructured its draws, so it is written down rather
  // than left to be inferred from the transcript.
  //
  // It is rewritten after every draw rather than once on exit, because the
  // number that matters is how much had been drawn when the run stopped
  // agreeing with the transcript -- and a run that diverged by hanging is
  // killed, which fires no exit handler at all. Rewriting at offset 0 is safe
  // without truncating: the count only ever grows, so its decimal form only
  // ever gets longer and each write covers the last.
  const usagePath = process.env.PRESCRIPT_TAPE_USAGE;
  const usageFd = usagePath ? fs.openSync(usagePath, "w") : null;

  const writeUsage = () => {
    if (usageFd !== null) {
      fs.writeSync(usageFd, `${drawn}\n`, 0);
    }
  };

  // Written from the start, so that a port which drew nothing at all says so
  // rather than saying nothing.
  writeUsage();

  Math.random = function random() {
    // Running off the end is an error rather than a wrap. A port that draws
    // more than the reference did has restructured how it consumes
    // randomness, which is a finding; silently starting the tape again would
    // turn that finding into a transcript that merely looks wrong later on.
    //
    // Reported and exited rather than thrown, because a thrown error arrives
    // as a stack trace through Math.random, and the line that says what went
    // wrong scrolls off the top of the failure report.
    if (drawn >= values.length) {
      process.stderr.write(
        `prescript: the tape ran out after ${values.length} values; ` +
          "this port draws more randomness than the reference did\n",
      );
      process.exit(1);
    }

    const value = values[drawn++];
    writeUsage();
    return value;
  };
} else {
  // mulberry32: short enough to read in one sitting, and its whole state is a
  // single 32-bit word, which is what makes the sequence identical everywhere.
  let state = (Number(process.env.PRESCRIPT_SEED || "0") + 0x9e3779b9) >>> 0;

  Math.random = function random() {
    state = (state + 0x6d2b79f5) >>> 0;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
