#!/bin/sh
# both.sh NAME KEYS... : run the original and the Go engine on the same key script, then diff.
cd "$(dirname "$0")"
n=$1; shift
.venv/bin/python run.py game --keys "$*" --out out/$n.emu.json --png out/$n.emu.png >/dev/null 2>out/$n.emu.log
go run ../../cmd/trace -keys "$*" -out out/$n.go.json -png out/$n.go.png 2>out/$n.go.log
tail -1 out/$n.emu.log; tail -1 out/$n.go.log
.venv/bin/python compare.py out/$n.emu.json out/$n.go.json
