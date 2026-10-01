---
name: world-visit
description: Set up an old Minecraft Java save to walk around in and screenshot without damaging it - APFS clone, cheats on, a datapack teleport book of saved locations, the right client version - and afterwards copy that session's screenshots into per-world folders. Use when the user wants to revisit, tour, explore in-game, or take screenshots in an old/archived/backup world, or sort their Minecraft screenshots by world.
---

# World visit

macOS client only. Scripts are pure Python 3. Pair with `world-explore` to find what's worth visiting.

## 1. Clone the save (never play the original)

```bash
V=.claude/skills/world-visit/scripts
python3 $V/clone_world.py "<save>" --pack <datapack dir>   # -> "<save> (screenshots)"
```

It refuses if the clone exists or a game process has the save open, makes an APFS clone
(`cp -c`, no extra disk), sets `allowCommands` in the clone's `level.dat`, installs the
pack, and prints the version to launch.

- **Launch exactly that version** (Launcher > Installations > New). A newer client
  upgrades the save permanently. Check the installed ones in `~/Library/Application Support/minecraft/versions/`.
- Every save shows its `LevelName` in the world list, so clones look like the original.
  The folder name underneath tells them apart.
- **Never run `ps -ef` / `pgrep -fl` on the game**: the launcher puts the account access
  token on the java command line. Use `pgrep java` + `lsof -p <pid>` (as the script does).

## 2. Teleport book

Write a `locations.json` (format in `make_book.py`'s docstring), usually from `world-explore`
finds, then:

```bash
python3 $V/make_book.py locations.json --out <dir>   # <dir>/<world>/oldworlds-book/
```

In game: `/reload`, then `/function oldworlds:book` (written book) or
`/function oldworlds:places` (same links as chat).

- **Books don't open in spectator.** Use F3+N to flip spectator <-> previous mode, or use
  the chat links, which work in any mode. Don't put a "spectator" button in the book.
- Y values from scans are rough, so teleports can land inside blocks. Spectator fixes that.
- Format is pre-1.20.5 item NBT (`clickEvent`/`hoverEvent`, `functions/` dir). 1.20.5+
  needs `written_book_content`, and 1.21.5+ renames the click/hover fields: update the script before using it there.
- The datapack lands in the clone's `datapacks/`. If the game is open, `/reload` picks up a regenerated pack.

## 3. Screenshots

F1 hides the HUD, F2 saves to the shared `~/Library/Application Support/minecraft/screenshots/`
(not per world). Afterwards:

```bash
python3 $V/sort_screenshots.py --since YYYY-MM-DD --map 1.16.4=old --map 1.19=world.bak --dest ~/Pictures/... --dry-run
```

- Logs never name the save folder (every world says "world"). The script assigns each
  screenshot by the integrated-server version running at the time, so this only works if each
  version maps to one save. Cross-check with the clone's `level.dat` mtime against the session end.
- Multiplayer sessions ("Connecting to <host>") are reported, not copied. Ask the user
  which world the server was running.
- Copies only; skips existing. Delete originals only when asked, and only after
  `cmp` confirms each copy.
