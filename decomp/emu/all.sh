#!/bin/sh
# all.sh [NAME...]: run the scenarios of scenarios.txt through both.sh; print a pass/fail line each.
# A scenario named NAME@FIX starts from the save fixture saves/FIX (see saves/README.md).
cd "$(dirname "$0")"
grep -v '^#' scenarios.txt | while IFS=: read -r n k; do
	[ -z "$n" ] && continue
	fix=
	case $n in *@*) fix=${n#*@}; n=${n%@*} ;; esac
	if [ $# -gt 0 ] && ! echo " $* " | grep -q " $n "; then continue; fi
	if POR_SAVES=${fix:+$PWD/saves/$fix} ./both.sh "$n" $k > out/$n.cmp 2>&1; then echo "PASS $n"; else echo "FAIL $n: $(grep -m1 'diverge\|differs' out/$n.cmp)"; fi
done
