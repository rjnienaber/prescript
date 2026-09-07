// Replaces Math.random with a seeded generator, so a port that calls it
// produces the same sequence on every run and on every machine.
//
// Activated by NODE_OPTIONS=--require <this file>, which Node runs before the
// program, so the program does not have to know it exists and does not have to
// be edited. It reaches ESM programs too: --require runs ahead of the module
// loader, and Math is the same object either way.
//
// V8's own Math.random cannot be seeded -- there is no API for it, and the
// engine seeds itself from entropy when a context is created -- so this is a
// replacement rather than a seeding. The algorithm is mulberry32, chosen
// because it is short enough to read in one sitting and its whole state is a
// single 32-bit word, which is what makes the sequence identical everywhere.
//
// What this does not reach: crypto.randomBytes, crypto.getRandomValues, and
// anything a dependency bundles for itself. A port using those needs its own
// answer; see docs/determinism.md.
"use strict";

let state = (Number(process.env.PRESCRIPT_SEED || "0") + 0x9e3779b9) >>> 0;

Math.random = function random() {
  state = (state + 0x6d2b79f5) >>> 0;
  let t = state;
  t = Math.imul(t ^ (t >>> 15), t | 1);
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};
