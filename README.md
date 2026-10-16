# por-go

A Go port of *Pool of Radiance* (SSI, 1988), with a modernized interface.

No game data is included. The engine reads the original game files (the
`.DAX` files, `ITEMS`) from your own copy of the game at run time. The rule
tables of `START.EXE` (races, classes, experience, spells ...) live in
`engine/tables.go`.

The 8x8 fonts in `engine/fonts` are from The Ultimate Oldschool PC Font Pack
by VileR; see `engine/fonts/LICENSE.txt`.
