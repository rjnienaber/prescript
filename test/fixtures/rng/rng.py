# A port in miniature: it asks something, then prints numbers it drew at
# random. Without a seed those numbers differ on every run, which is what makes
# a program like this impossible to script.
import random
import sys

print("HOW MANY? ", end="", flush=True)
count = int(sys.stdin.readline().strip())
# INT(RND(1) * 1000), the way a port translates it. Written the same way in
# every language here, so a tape gives all of them the same numbers.
print(" ".join(str(int(random.random() * 1000)) for _ in range(count)))
