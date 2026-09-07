# Seeds Ruby's default random number generator, so a port that calls rand
# produces the same sequence on every run and on every machine.
#
# Activated by RUBYOPT=-r<this file>, which Ruby loads before the program, so
# the program does not have to know it exists and does not have to be edited.
#
# What this does not reach: Random.new(...) and SecureRandom, both of which
# seed themselves. A port using either needs its own answer; see
# docs/determinism.md.
srand(Integer(ENV.fetch("PRESCRIPT_SEED", "0")))
