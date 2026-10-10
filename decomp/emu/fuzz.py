"""fuzz.py SEED [N]: print a random key script that loads save A and then plays N keys."""
import random, sys
r = random.Random(int(sys.argv[1]))
n = int(sys.argv[2]) if len(sys.argv) > 2 else 60
moves = ['up'] * 6 + ['left', 'right', 'down']
other = ['enter', 'esc', 'space', 'y', 'n', 'a', 'c', 'v', 'e', 's', 'l', 'f', 't', 'r', 'p', 'q', 'x', 'b', 'd', 'm', 'o', 'u', 'h', 'i', 'g', 'w', '1', '2', '3', ',', '.']
keys = ['enter', 'l', 'a']
for _ in range(n):
    keys.append(r.choice(moves) if r.random() < 0.6 else r.choice(other))
print(' '.join(keys))
