# Port checklist

What is left before the Go port plays the whole game like the DOS
original. Ticked items are ported and checked in a headless run
(`./build_n_run.sh snap ...`). Routine addresses are from GAME.OVR / START.EXE.

## Already running
- [x] Start-up, title, graphics, text, boxes (START 0709/03F1/0464, ovr02, ovr12)
- [x] Party menu, load/save game (ovr26, ovr27, ovr29, ovr30, ovr35)
- [x] Game loop, movement, turning, doors (bash/pick/knock), area changes (ovr04, ovr15)
- [x] Clock and timed spell effects (ovr21:0000/02F2/03D5)
- [x] ECL interpreter, all opcodes 0x00-0x3D dispatched (ovr08)
- [x] 3D view, wall sets, animated pictures (ovr26 wait loop)

## Rules and support
- [x] ovr25 rules: affects, RollHit, AttackHits, SavingThrow, Damage, Heal, Revive, strength (engine/rules.go; combat-map calls are placeholders in combatmap.go)
- [x] ovr24 RecomputeCharacter 0000 (engine/charcalc.go)
- [x] ovr22 money: pool, share, trade, drop, take (engine/money.go)
- [x] ovr22 RandomItem 1010, 19B2 gems/jewelry appraisal (engine/randitem.go)
- [x] START item/character helpers: 0442 ItemName, 0BB8 ApplyItems (+0000..03A0, 10E6..13AB), 1509 RemoveItem, 15F7 AddItemCopy, 1693 BottomMessage, 2199 FindAffect, 2228 ApplyDamage
- [x] START 2C48 SelectCharacter
- [x] START 1A29 flyMissile (+ ovr31 line walker 0000/01A0/02CC), 23B7, 2433 nearEnemies, 255B rangeTo, 26C2 casterLevel (engine/combatmap.go; map calls are placeholders until ovr32, DS:65BD combatant table filled by ovr13)
- [x] START 1892 icon frames, 09E3 combat panel
- [x] Sound effects 08AE:00DC (PC speaker and Tandy sequencer, engine/sound.go; the songs are Go data generated from START.EXE by cmd/sounddata, so playing them needs no exe)

## Adventuring commands
- [x] ovr16 Encamp menu 1E6F: save, view, magic, rest (ovr21), alter (engine/camp.go; Alter Icon waits on ovr17:382D)
- [x] ovr16 Cast 04A1, memorise, scribe, display affects (engine/camp.go)
- [x] ovr23 spells: tables 3413, list/menu, CastSpell, effects of every spell (engine/spells1.go, spells2.go)
- [x] ovr20 View character: sheet, items, ready, trade, drop, halve, join, sell, identify, money trade/drop (engine/view.go, money.go)
- [x] ovr20 SpellList 299C (engine/view.go)
- [x] ovr07 Shop (053E, 0361, 0239, 0030): buy, view, take, pool, share, appraise (engine/shop.go)
- [x] ovr05 Temple (0D09, 09D7, cures 0249..0928): heal, cures, raise dead (engine/temple.go)
- [x] ovr28 Area map 0000 + party icon (ovr27:0000), 00BA:2B5E icons (engine/areamap.go)

## Character management (party menu)
- [x] ovr17 Create character (race, class, alignment, rolls, name, icon) (engine/create.go)
- [x] ovr17 Modify, Drop, Add, Remove, Train (levels, new spells) (engine/create.go, train.go)
- [x] ovr17:3291 RemoveSelected (engine/party.go)
- [x] ovr18 BuildCharIcon 0482 (engine/cmap.go)
- [x] ovr18 (engine/story.go); SaveAreaState 22D5 (the "Insert disk #" prompt) is dropped: all files are in one folder
- [x] ovr31/32/33 combat map, near list, icons (engine/cmap.go)

## Combat
- [x] ovr13 affect handlers (31A2 init, all handlers) (engine/affects1.go, affects2.go)
- [x] ovr14 combat turn: movement, attacks, aiming, spell targets (engine/combatturn1.go, combatturn2.go)
- [x] ovr06 after combat (1527, 04C9, 0000, 0355, 11AC, 12E8, 0916, 0ECD): result, experience, NPC share, loot menu (engine/aftercombat.go)
- [x] ovr11 combat setup: dungeon/wilderness maps, formations (engine/combatsetup.go)
- [x] ovr09 main loop, combat menus (engine/combat.go)
- [x] ovr10 monster AI and Quick (engine/combatai.go); START 2B88, 2E8B, 2F57
- [x] Full fight checked headless: `./build_n_run.sh snap OUT game A KEYS combat`
- [x] DuelSetup ovr08:1A6F (the CPIC clone part) (engine/story.go)

## Story and endings
- [x] ovr19 ProgramEight 02A4, endgame 06B3 (engine/story.go)
- [x] ovr34 0004 (affects2.go), 00F7 (story.go)
- [x] Routine coverage audit: every overlay routine found by Ghidra, prologue scan and xref.tsv is named in the Go code (START.EXE not audited)

## Extras (not in the original)
- [x] Minimap over the roster (Tab / gamepad Y)
- [x] Area dropped from the command bar; the minimap shows every wall as Area did (unseen squares dim)
- [x] Full-screen map (M, or a click on the minimap): cursor, legend, Label / Remove per square, kept in MAPNOTES.TXT; New Phlan's shops, temples, inns, taverns and offices come labelled (from the block's step-event ONGOTO), labelled squares are dots on the minimap (engine/fullmap.go)
- [x] Party overview (P / gamepad X)
- [x] Gamepad controls (d-pad walks in the 3D view, moves the selection in menus)
- [x] Selection indicator on every menu (blue background; the item keeps its colours, white hotkey and green rest, list lines their own colour)
- [x] build_n_run.sh for playtesting
- [x] Text fonts: Ctrl-L / Ctrl-Shift-L cycle the game's font and 56 8x8 PC fonts (Oldschool PC Font Pack, CC BY-SA 4.0) in real upper/lower case; text on screen is redrawn, the choice is kept in FONT.CFG; fonts in the game folder's FONTS/*.f8 join the cycle after the game's own (The Dark Queen of Krynn's 8x12 GAME.FON via decomp/tools/dqfon2f8.py; Champions and Death Knights of Krynn use POR's font unchanged), never built in, `-import` copies them (engine/fonts.go)
- [x] Journal texts: when the text window names a journal entry, tavern tale or proclamation, the text is shown in a pager (all proclamations, journal entries and tavern tales, built in from engine/journal/JOURNAL.TXT) (engine/journal.go)
- [x] Journal pictures: the maps and sketches of the printed journal show inside their entries, full 300 dpi scans drawn over the screen at the window's resolution (ega.Overlay; snap writes a 4x _hi.png), never split across pages (one that overruns shrinks into the rows left when they hold half of it, else starts the next page; one taller than wide shows at full width in a page-high window that Up/Down and the wheel scroll); lines that fit keep their spacing (entry 53 table) (engine/journal/*.PNG)
- [x] One party purse (engine/money.go): gold, gems and jewelry belong to the party, not to characters; coins of every kind count as gold, found money goes straight into it, shops, temples, training and identifying pay from it, Appraise sells the party's gems and jewelry; Pool, Share, Take money, Trade and Drop money are gone; a character's record coins (starting gold, old saves) join the purse when they join the party; the purse is kept after the save game data; ECL scripts see it as the selected member's platinum and gold
- [x] Shop and temple show the party's gold (also on the shop's Buy list)
- [x] Input latency: the window polls input at 120 Hz and draws a key's result in the same frame (it waits up to 8 ms for the engine to settle, ega.Screen state); held arrows repeat like the PC keyboard while the engine waits; a key finishes slowly printed text at once
- [x] Enter confirms the highlighted item on every menu bar (lists, member selection, aiming and resting keep the game's own Enter)
- [x] Up/Down move the highlight on every menu bar like Left/Right (Home/End still step the member); each menu keeps its own highlight (the original shares one), Yes/No keeps its preselected No
- [x] Keyboard help (?) at any prompt, keys grouped by where they work; pages (key = next, Up/PgUp = back, Esc = close)
- [x] F1-F6 open the inventory of member 1-6 (also switch members there and on the View page); P toggles inventory / View page
- [x] Display dropped from the camp Magic menu: spell effects are on the View page (Effects line; a Magic item lists them all when they do not fit)
- [x] Quickstart on the empty party menu: six pre-rolled members (engine/quickstart.go)
- [x] One set of navigation keys on every menu (engine/nav.go): arrows move the highlight (Up/Down step the member where menus take the keypad), Esc picks a menu's Exit
- [x] Mouse: hover highlights menu items, click picks items, list lines (twice = Enter), party members, combat squares (step, aim cursor, target); wheel, right button = Esc
- [x] Resizable window, picture kept 4:3 (the hint line under the picture is gone: it broke the look; ? lists the keys)
- [x] Browser build for goplay (https://ruzzoli.de/goplay/): `-import` finds the game in what the player drops, `GOPLAY_DATA` is the data dir (cmd/por/import.go)
- [x] Context-sensitive adventure bar (engine/cmdbar.go): Cast only when the selected member has a spell to cast
- [x] Walk on the full map (engine/walk.go): the party walks to the cursor square, stopping at text, pictures, menus, monsters, a blocked step or any key
- [x] Down steps backwards in the 3D view, facing kept (the original turns round); the step scripts see the wall behind, not the one ahead
- [x] Help and journal boxes end above the menu row, so their bottom border shows
- [x] Shop Buy list: opens at the top on the first usable item (the original starts on the second, top left from the last list); F1-F6 change the buyer; goods the buyer cannot ready are grey and passed over by the cursor (engine/shop.go canUseItem, ListNode.Dim)
- [x] -import copies ITEMS (the base item table: without it every item was the wrong class)
- [x] Classic mode removed: the modern behaviour everywhere, the original-only code paths are gone (parity traces retired, PARITY_CHECKLIST.md)
- [x] Camp: the roster replaces the minimap; F1-F6 pick the member the camp and Magic menus act for; the menus list only what that member can do (Magic for casters; Cast with memorized spells where casting is allowed; Memorize with spell slots; Scribe for magic-users carrying a scroll)
- [x] Memorize list: each spell shows how many are chosen (xN); Clear (shown while any are chosen) drops the whole selection
- [x] Rest menu "Heal Memorize Wait Night|Day Exit" replaces the days/hours/minutes editor: Heal works like the later games' Fix (clerics cast their Cure Light Wounds, memorize more in every first-level slot and rest to learn them while that helps, then relearn their own spells; natural rest, 1 HP a day, does the rest), Memorize the party's study time, Wait 1 hour (timed script events), Night/Day until 20:00/6:00
- [x] Day sky from 6:00 to 20:00 (the scripts' own check is hour < 14; patched in opCompare)
- [x] Item names end in Angband-style base stats: "(1d8)" weapon damage, "<7>" armour class given (the font has no square brackets) (shop names stop before the price)
- [x] Shop Buy list names the buyer (top right; F1-F6 change it); a missing ITEMS file (imports made before it was copied) is reported at start: every item would be the wrong class
- [x] Shop Buy list Look (L): a box with the item's rules stats (damage, vs large, range and shots, ammunition, AC, movement, who can use it) and what special items do (oil, holy water, mirror, holy symbol, silver, scrolls) (engine/itemlook.go)
- [x] Map and minimap doors are light brown (Amiga palette 0xA84) instead of light cyan, which ran together with the white walls
- [x] Icon colours recolour Amiga icons too: EGA colour numbers map one-to-one onto the Amiga icon palette slots (Amiga table 0x8B10E vs DS:3CF5), so defaults match and every step shows a new colour
- [x] Amiga combat heads: the Amiga has one head per picture, used for both poses (the action pose took the DOS head before); chained colour changes end the same on Amiga pixels as on DOS ones
- [x] Amiga ground tile sheets (SQRPACI, DUNGCOM, WILDCOM, RANDCOM, BACPAC): each bitplane runs through all frames, and they use the terrain palette at 0x54020 (re-import the Amiga disks to get them); sprites (missiles, hit flashes) restore the Amiga floor under them, not EGA colours
- [x] Amiga combat icons (CHEAD, CBODY, CPIC, COMSPR, ICON) use the terrain palette too: the Amiga combat screen (0x39F0E) and icon editor (0x2EB02) set it; the part table at 0x8B10E gives each part's light/dark slots, as amigaIconColour maps them
- [x] Combat screen full width: a 12x7-square view in one box (the side pane is gone); row 23 shows the current combatant's name, HP, AC and damage (or status); combat messages use rows 23-24 and go into a battle log, which the Log command shows (scroll with Up/Down, PgUp/PgDn, wheel) (engine/combat.go, engine/cmap.go)
- [x] Combat Move shows every square the character can still reach (dotted cyan frames: 2 moves a step, 3 diagonally, ground no dearer than the moves left); a click on one walks there step by step by the cheapest path, each step as the keypad takes it (free attacks, clouds), stopping when the way is blocked (engine/combat.go moveReach)

## Idiomatic Go
- [x] Character record as a typed struct (engine/char.go charRecord): named fields in the DOS layout, read and written with encoding/binary (saves and MON*CHA blocks still load byte for byte), no Raw byte array or offset constants; checked with cmd/trace against the previous build on all scenarios plus 164 fuzz scripts (identical)
- [x] Item and Affect records as typed structs (Item, affectRecord in engine/files.go): readRecords/writeRecords for .itm/.spc, MON ITM/SPC and ECL ITEM blocks, no Raw arrays or it*/af* constants; cmd/trace identical to the previous build on all scenarios plus fuzz scripts
- [x] CombatInfo as named fields (Spell, Delay, Moves, Facing, Fleeing, ...) instead of a Raw [0x16]byte with ci* offsets; cmd/trace identical
- [x] Stinking-cloud records (cloud) as named fields (Ground, Covered, X, Y, Num); no Raw arrays left in engine records
- [x] Pascal "var" out-params turned into Go results where the callee only writes them (attack/turn done flags, spell targeting ok, rest/identify/AC/AI-step flags); unused DS:6E31 dropped. Cursor-state in/out params (ListMenu index/redraw, aim found) stay pointers
