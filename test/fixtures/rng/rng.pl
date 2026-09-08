# A port in miniature: it asks something, then prints numbers it drew at
# random. Without a seed those numbers differ on every run, which is what makes
# a program like this impossible to script.
use strict;
use warnings;

$| = 1;
print "HOW MANY? ";
my $count = int( <STDIN> );
# INT(RND(1) * 1000), the way a port translates it. Written the same way in
# every language here, so a tape gives all of them the same numbers.
print join( " ", map { int( rand() * 1000 ) } 1 .. $count ), "\n";
