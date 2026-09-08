-- Makes a Lua port's randomness repeatable, in one of two ways.
--
-- With PRESCRIPT_TAPE set, math.random returns values recorded from the
-- reference BASIC interpreter, in order, so this port and every other port of
-- the same program draw the same numbers. Without it, Lua's own generator is
-- seeded from PRESCRIPT_SEED, which makes this port repeatable but leaves it
-- drawing a different sequence from every other language. See
-- docs/determinism.md.
--
-- Activated by LUA_INIT=@<this file>, which the standalone interpreter runs
-- before the program, so the program does not have to know it exists and does
-- not have to be edited. Lua looks for a version-specific LUA_INIT_5_4 first
-- and falls back to LUA_INIT, so the plain name works whichever 5.x is
-- installed.
--
-- What this does not reach: a generator a port builds for itself with
-- math.randomseed's return values or an RNG from LuaRocks, and os.time or
-- os.clock used as a value rather than as a seed.

local tape = os.getenv("PRESCRIPT_TAPE")

if not tape or tape == "" then
  local seed = tonumber(os.getenv("PRESCRIPT_SEED") or 0) or 0
  math.randomseed(math.tointeger(seed) or 0)
  return
end

local values = {}
for line in io.lines(tape) do
  -- A fresh local rather than the loop variable, which 5.5 makes const.
  local trimmed = line:match("^%s*(.-)%s*$")
  if trimmed ~= "" and trimmed:sub(1, 1) ~= "#" then
    values[#values + 1] = tonumber(trimmed)
  end
end

local drawn = 0

-- How much of the tape a port used is the number that separates a real logic
-- bug from a port that restructured its draws, so it is written down rather
-- than left to be inferred from the transcript.
--
-- The count is rewritten after every draw rather than written once at exit,
-- because the number that matters is how much had been drawn when the run
-- stopped agreeing with the transcript -- and a run that diverged by hanging
-- is killed, which runs no exit handler at all. Rewriting in place is safe
-- without truncating: the count only ever grows, so its decimal form only ever
-- gets longer, and each write covers the last.
local usage_path = os.getenv("PRESCRIPT_TAPE_USAGE")
local usage = nil
if usage_path and usage_path ~= "" then
  usage = io.open(usage_path, "w")
  if usage then
    usage:setvbuf("no")
  end
end

local function write_usage()
  if not usage then
    return
  end
  usage:seek("set", 0)
  usage:write(drawn, "\n")
end

-- Written from the start, so that a port which drew nothing at all says so
-- rather than saying nothing.
write_usage()

-- Running off the end is an error rather than a wrap. A port that draws more
-- than the reference did has restructured how it consumes randomness, which is
-- a finding; silently starting the tape again would turn that finding into a
-- transcript that merely looks wrong somewhere further on.
local function next_value()
  if drawn >= #values then
    io.stderr:write(("prescript: the tape ran out after %d values; this port " ..
      "draws more randomness than the reference did\n"):format(#values))
    os.exit(1)
  end

  drawn = drawn + 1
  local value = values[drawn]
  write_usage()
  return value
end

-- math.random's three shapes, in terms of one uniform value: no argument for a
-- float in [0,1), one for an integer in [1,m], two for an integer in [m,n].
math.random = function(m, n)
  local value = next_value()

  if m == nil then
    return value
  end

  if n == nil then
    m, n = 1, m
  end

  return m + math.floor(value * (n - m + 1))
end

-- A port that seeds itself is asking for a sequence it has already been given.
math.randomseed = function() end
