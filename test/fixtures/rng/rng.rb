# A port in miniature: it asks something, then prints numbers it drew at
# random. Without a seed those numbers differ on every run, which is what makes
# a program like this impossible to script.
print "HOW MANY? "
count = Integer($stdin.gets.strip)
# INT(RND(1) * 1000), the way a port translates it. Written the same way in
# every language here, so a tape gives all of them the same numbers.
puts (1..count).map { (rand * 1000).to_i }.join(" ")
