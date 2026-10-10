---
name: ghidra
description: Decompile Pool of Radiance overlay / START.EXE routines to pseudo-C with Ghidra (headless GhidraMCP) as a reading aid while porting. Use when a routine in por-go/decomp is long or tangled and its structure is hard to see from the asm listing.
---

# Ghidra for the POR port

Ghidra 12.1 (brew) + GhidraMCP 6.0 headless server, fed the raw code blobs
in `por-go/decomp/out` (`ovr/ovrNN.bin`, `START.bin`).

## Start the server (once per session)

```bash
curl -s localhost:8090/check_connection || (PORT=8090 ~/tools/ghidra-mcp/run-headless.sh > ~/tools/ghidra-mcp/server-por.log 2>&1 &)
```

Port 8090 is POR's own server: the EOB session uses 8089, and a shared
server switches the current program under you.
Wait until `check_connection` answers. The `ghidra` MCP server in
`.mcp.json` bridges to it; curl works just as well and needs no restart.

## Load + analyse a blob

```bash
B=/Users/felix/Projects/por-go/decomp/out
curl -s -X POST -H 'Content-Type: application/json' \
  -d "{\"file\":\"$B/ovr/ovr17.bin\",\"language\":\"x86:LE:16:Real Mode\",\"compiler_spec\":\"default\"}" \
  localhost:8090/load_program
curl -s -X POST localhost:8090/run_analysis
curl -s "localhost:8090/list_functions?limit=200"
curl -s "localhost:8090/decompile_function?address=0000:3291"
```

Loading another blob makes it current; pass `program=ovr17.bin` to pick
one explicitly. Addresses are `0000:OFS` = the `ovrNN:OFS` of view.py.
If a routine was not found, `POST /create_function` with `address`.

## What to trust

The blobs are unrelocated, so Ghidra cannot see segments:
- **Far calls are wrong.** `lcall` targets (other overlays, START, the RTL
  0AF8:xxxx) show up as `FUN_0000_xxxx` in this blob. Always take call
  targets from `python3 decomp/tools/view.py ovrNN LO HI` (it annotates
  them) or `decomp/out/xref.tsv`.
- **Far pointers lose their segment** (`*(int *)0x5d92` = DS:5D92 is right,
  `es:` derefs through `les` are shown as near). DS offsets are fine.
- Turbo Pascal nested procedures (parent bp pushed as hidden arg) and
  string RTL calls decompile to noise; read them in the asm.
- Arguments: Pascal pushes left to right, so Ghidra's `param_1` is the
  LAST Pascal argument. Rename in your head before porting.

- Auto-analysis misses routines that are only reached by far calls
  (ovr17:056C CreateCharacter was folded into 0AEA, locals shown as
  `unaff_BP + -n`). Seeing `unaff_BP` means a wrong function start:
  `create_function` at the real entry (from xref.tsv / view.py) first.
- Ghidra cannot answer "where is this local written?". Grep the listing:
  `view.py ovrNN LO HI | grep "bp - 0x17\]"`.

Use the C for control flow (loops, if/else nesting, switch shape), then
port 1:1 from the asm. Never port from the C alone.
