#!/bin/sh
# par.sh FROM TO [N] [JOBS]: fuzzall.sh over FROM..TO, JOBS seeds at a time
# (each run has its own SAVE scratch); one PASS/FAIL line per seed, unordered.
cd "$(dirname "$0")"
seq "$1" "$2" | xargs -P "${4:-4}" -I{} ./fuzzall.sh {} {} "${3:-80}"
