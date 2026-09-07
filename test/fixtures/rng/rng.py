# A port in miniature: it asks something, then prints numbers it drew at
# random. Without a seed those numbers differ on every run, which is what makes
# a program like this impossible to script.
import random
import sys

print("HOW MANY? ", end="", flush=True)
count = int(sys.stdin.readline().strip())
print(" ".join(str(random.randint(0, 999)) for _ in range(count)))
