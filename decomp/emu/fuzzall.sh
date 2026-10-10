#!/bin/sh
# fuzzall.sh FROM TO [N]: fuzz seeds FROM..TO through both.sh; one PASS/FAIL line each.
cd "$(dirname "$0")"
s=$1
while [ "$s" -le "$2" ]; do
	k=$(python3 fuzz.py $s ${3:-80})
	if ./both.sh fuzz$s $k > out/fuzz$s.cmp 2>&1; then echo "PASS fuzz$s"; else echo "FAIL fuzz$s: $(grep -m1 'diverge\|differs' out/fuzz$s.cmp)"; fi
	s=$((s+1))
done
