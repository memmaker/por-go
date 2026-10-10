# emu: run the original Pool of Radiance under Unicorn

Runs START.EXE plus GAME.OVR in real mode, with every overlay resident and the
hardware primitives intercepted. It emits a trace with the same record format
as the Go engine's trace mode, so the two can be diffed.

## Setup (once)

    cd por-go/decomp/emu
    python3 -m venv .venv && .venv/bin/pip install -r requirements.txt

## Commands (from por-go/decomp/emu)

| What | Command |
|---|---|
| Original, title to save A loaded | `.venv/bin/python run.py game --keys "enter l a" --out out/emu.json --png out/emu.png` |
| Same, PNG after record N | `... --png-at 2100:out/title.png` |
| Same, snapshot at stop | `... --save-snap out/menu.snap` |
| Call one routine from a snapshot | `.venv/bin/python run.py call ovr25:0DE3 3 6 --snap out/menu.snap` |
| Go engine, same script | `go run ../../cmd/trace -keys "enter l a" -out out/go.json -png out/go.png` |
| First divergence | `.venv/bin/python compare.py out/emu.json out/go.json [--only DisplayString,ReadKey]` |
| Start from a save fixture | `POR_SAVES=$PWD/saves/finale ./both.sh finale enter l a ...` (both sides copy the fixture over SAVE) |
| Build a fixture | `cd ../.. && go run ./cmd/fixture -out decomp/emu/saves/X -block N -walls A,B,C -x X -y Y` (see saves/build.sh) |
| List an area script | `cd ../.. && go run ./cmd/ecl DISK BLOCK` |
| Go PNG after record N | `go run ../../cmd/trace ... -png-at 21987:out/go.png` (Go counts records without BIOS ones) |
| Key parser self-test | `.venv/bin/python keys.py` and `go test ../../cmd/trace` |

`all.sh` runs every scenario of scenarios.txt. A name `NAME@FIX` runs with
`POR_SAVES=saves/FIX` (see saves/README.md).

Run `run.py game -h` / `run.py call -h` for all the flags. `--script FILE`
reads key tokens one or more per line, with `#` comments. Key tokens are
enter esc space bs tab up down left right home end pgup pgdn ins del f1..f10,
a single character, or 0xHHHH (a raw BIOS word).

Call-mode args go in Pascal order: a word (`123`, `0x7B`), `ptr:SEG:OFS` (SEG
may be `DS`), `str:TEXT` or `buf:N`. Add `--near` for near routines and
`--poke DS:5376=03` to patch memory first. The output has AX and DX, the bytes
popped, a diff of DS and of the game, area, aux, ecl and party records, the
contents of each buf, and the trace.

## Determinism model (shared with Go trace mode, engine/trace.go)

- **Seed:** `Randomize` (0AF8:1247) sets RandSeed DS:4428 to `--seed` (default 0x12345678). `Random(n)` is TP's LCG: `seed = seed*0x08088405+1`, result `(seed>>16) % n`.
- **Time:** virtual ms is the sum of Delay(ms) calls, plus 10 ms for each Ticks() call.
- **Keys:** a key becomes visible to KeyPressed on the poll after one that saw none. ReadKey takes the next key at once. The run ends when ReadKey finds the script empty, or after 200 idle polls.
- **Copy protection:** ovr03:00FD is skipped on both sides.
- **Saves:** writes go to scratch/SAVE, a fresh copy of SAVE made on each run. Go uses a temp copy.

## Trace records

Each record is `[addr, name, args...]`, where addr is the original routine
(`0709:0626`, or `ovrNN:OOOO` for overlays). BIOS and DOS calls show up as
`INT10`, `int21` and similar; Go emits only `INT16 ReadKey`. Picture arguments
are `HxW:md5[:10]` of the decoded colour indices. Intercepted primitives live
in prims.py and are skipped. Probes are traced, then run normally.

## Known gaps

- compare.py by default ignores BIOS/DOS records and the sound driver (08AE), because Go does not emit them. Go has no DisplaySpaces, sound, 08D4 or 0D71 hooks, since those live in files that were off limits or use other coordinate frames. The original's DisplayChar(count=40) shows up as 40 single-character records in Go.
- The Go Flip hash covers a screen rectangle; the emulator hashes the page record. Picture hashes cover frame 0 only.
- Divergences the first run found, still unfixed:
  1. Init loads `COMSPR` blocks 0..11 (plus 128+). Go `init.go` passes `i+0xD`.
  2. Go `LoadIconPic` defers `ClearKeyboard`, which eats scripted keys during Init. The original does not drain there.
