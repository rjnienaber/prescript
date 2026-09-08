# Makes a Perl port's randomness repeatable, in one of two ways.
#
# With PRESCRIPT_TAPE set, rand returns values recorded from the reference
# BASIC interpreter, in order, so this port and every other port of the same
# program draw the same numbers. Without it, Perl's own generator is seeded
# from PRESCRIPT_SEED, which makes this port repeatable but leaves it drawing a
# different sequence from every other language. See docs/determinism.md.
#
# Activated by PERL5OPT=-M<this module>, which Perl loads before it compiles
# the program, so the program does not have to know it exists and does not have
# to be edited. That ordering is what makes the override work at all:
# CORE::GLOBAL::rand is consulted when a call to rand is compiled, so an
# override installed afterwards would arrive too late for every call site in
# the program.
#
# What this does not reach: Math::Random::MT and the other generators from
# CPAN, /dev/urandom read directly, and anything whose randomness comes from a
# compiled XS module rather than from Perl's own rand.

package prescript;

use strict;
use warnings;

# Core, and needed for autoflush below: the draw count has to reach the disk
# before the run it describes is killed.
use IO::Handle;

# Package variables rather than lexicals, and set inside the BEGIN block below
# rather than here. The whole shim runs at compile time, and a module's own
# runtime statements -- these declarations included -- do not run until after
# it has been compiled, so a `my $drawn = 0` here would execute after the
# override that depends on it and reset what BEGIN had set up.
our @values;
our $drawn;
our $usage;

# Running off the end is an error rather than a wrap. A port that draws more
# than the reference did has restructured how it consumes randomness, which is
# a finding; silently starting the tape again would turn that finding into a
# transcript that merely looks wrong somewhere further on.
sub next_value {
    if ( $drawn >= @values ) {
        die "prescript: the tape ran out after "
          . scalar(@values)
          . " values; this port draws more randomness than the reference did\n";
    }

    my $value = $values[ $drawn++ ];
    write_usage();
    return $value;
}

# The count is rewritten after every draw rather than written once at exit,
# because the number that matters is how much had been drawn when the run
# stopped agreeing with the transcript -- and a run that diverged by hanging is
# killed, which runs no END block at all.
#
# Rewriting in place is safe without truncating: the count only ever grows, so
# its decimal form only ever gets longer, and each write covers the last.
sub write_usage {
    return unless $usage;

    seek $usage, 0, 0;
    print {$usage} "$drawn\n";
}

sub load_tape {
    my ($path) = @_;

    open my $tape, '<', $path
      or die "prescript: cannot read the tape at $path: $!\n";
    while ( my $line = <$tape> ) {
        $line =~ s/^\s+|\s+$//g;
        next if $line eq '' || $line =~ /^#/;
        push @values, $line + 0;
    }
    close $tape;
}

BEGIN {
    @values = ();
    $drawn  = 0;

    my $tape = $ENV{PRESCRIPT_TAPE};

    if ( defined $tape && $tape ne '' ) {
        load_tape($tape);

        # How much of the tape a port used is the number that separates a real
        # logic bug from a port that restructured its draws, so it is written
        # down rather than left to be inferred from the transcript. Written from
        # the start, so that a port which drew nothing at all says so rather
        # than saying nothing.
        my $path = $ENV{PRESCRIPT_TAPE_USAGE};
        if ( defined $path && $path ne '' ) {
            open $usage, '>', $path
              or die "prescript: cannot write the draw count to $path: $!\n";
            $usage->autoflush(1);
            write_usage();
        }

        # rand's two shapes, in terms of one uniform value: no argument (or 0,
        # which Perl treats as 1) for a float in [0,1), and an expression for
        # [0,n). srand becomes a no-op rather than an error, because a port that
        # seeds itself is asking for a sequence it has already been given.
        no warnings 'once';
        *CORE::GLOBAL::rand = sub (;$) {
            my $limit = @_ ? $_[0] : 1;
            $limit = 1 unless $limit;
            return next_value() * $limit;
        };
        *CORE::GLOBAL::srand = sub (;$) { return 0 };
    }
    else {
        my $seed = $ENV{PRESCRIPT_SEED};
        $seed = 0 unless defined $seed && $seed =~ /^-?[0-9]+$/;
        srand($seed);
    }
}

1;
