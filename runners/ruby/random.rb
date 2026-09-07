# Makes a Ruby port's randomness repeatable, in one of two ways.
#
# With PRESCRIPT_TAPE set, rand returns values recorded from the reference
# BASIC interpreter, in order, so this port and every other port of the same
# program draw the same numbers. Without it, Ruby's own generator is seeded
# from PRESCRIPT_SEED, which makes this port repeatable but leaves it drawing a
# different sequence from every other language. See docs/determinism.md.
#
# Activated by RUBYOPT=-r<this file>, which Ruby loads before the program, so
# the program does not have to know it exists and does not have to be edited.
#
# What this does not reach: Random.new(...), Random.rand, SecureRandom, and
# Array#sample and #shuffle, which draw from Random::DEFAULT rather than
# through Kernel#rand.

module PrescriptTape
  def self.load(path)
    @values = []
    File.foreach(path) do |line|
      line = line.strip
      next if line.empty? || line.start_with?("#")

      @values << Float(line)
    end
    @drawn = 0
  end

  # Running off the end is an error rather than a wrap. A port that draws more
  # than the reference did has restructured how it consumes randomness, which
  # is a finding; silently starting the tape again would turn that finding into
  # a transcript that merely looks wrong somewhere further on.
  def self.next_value
    if @drawn >= @values.length
      raise "prescript: the tape ran out after #{@values.length} values; " \
            "this port draws more randomness than the reference did"
    end

    value = @values[@drawn]
    @drawn += 1
    write_usage
    value
  end

  def self.drawn
    @drawn
  end

  # The count is rewritten after every draw rather than written once at exit,
  # because the number that matters is how much had been drawn when the run
  # stopped agreeing with the transcript -- and a run that diverged by hanging
  # is killed, which runs no at_exit handler at all.
  #
  # Rewriting in place is safe without truncating: the count only ever grows,
  # so its decimal form only ever gets longer, and each write covers the last.
  def self.record_usage(path)
    @usage = File.open(path, "w")
    @usage.sync = true
    write_usage
  end

  def self.write_usage
    return unless @usage

    @usage.seek(0)
    @usage.write("#{@drawn}\n")
  end
end

if (tape = ENV["PRESCRIPT_TAPE"]) && !tape.empty?
  PrescriptTape.load(tape)

  module Kernel
    # Kernel#rand's shapes, in terms of one uniform value: no argument or 0 for
    # a float in [0,1), an integer for [0,n), a float for [0.0,f), a range for
    # one of its members.
    def rand(limit = nil)
      value = PrescriptTape.next_value

      case limit
      when nil, 0 then value
      when Range
        first = limit.first
        count = limit.size
        count.nil? || count.zero? ? nil : first + (value * count).to_i
      when Float then value * limit
      else (value * limit.to_i).to_i
      end
    end
  end

  # How much of the tape a port used is the number that separates a real logic
  # bug from a port that restructured its draws, so it is written down rather
  # than left to be inferred from the transcript. Written from the start, so
  # that a port which drew nothing at all says so rather than saying nothing.
  if (usage = ENV["PRESCRIPT_TAPE_USAGE"]) && !usage.empty?
    PrescriptTape.record_usage(usage)
  end
else
  srand(Integer(ENV.fetch("PRESCRIPT_SEED", "0")))
end
