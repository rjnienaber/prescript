# A port in miniature: it asks something, then prints numbers it drew at
# random. Without a seed those numbers differ on every run, which is what makes
# a program like this impossible to script.
print "HOW MANY? "
count = Integer($stdin.gets.strip)
puts (1..count).map { rand(1000) }.join(" ")
