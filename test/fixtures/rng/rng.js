// A port in miniature: it asks something, then prints numbers it drew at
// random. Without a seed those numbers differ on every run, which is what makes
// a program like this impossible to script.
"use strict";

const readline = require("node:readline");

process.stdout.write("HOW MANY? ");

const input = readline.createInterface({ input: process.stdin });
input.once("line", (line) => {
  const count = parseInt(line, 10);
  // INT(RND(1) * 1000), the way a port translates it. Written the same way in
  // every language here, so a tape gives all of them the same numbers.
  const drawn = Array.from({ length: count }, () => Math.floor(Math.random() * 1000));
  process.stdout.write(drawn.join(" ") + "\n");
  input.close();
  process.exit(0);
});
