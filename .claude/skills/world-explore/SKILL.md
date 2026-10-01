---
name: world-explore
description: Find things inside a Minecraft Java save without opening the game - trading halls, bases, storage rooms, nether portals and highways, sign text, per-player play time, and player inventories/ender chests - and turn the finds into suggested map-config.yml ranges. Use when asked where a build is in a world, what's in a world, who played how much, what a player was carrying, or to pick focused render areas (ranges) for a world's map config.
---

# World explore

Read-only analysis of a save directory (the one with `level.dat`). Pure Python 3, no deps.
Handles 1.16 through 1.20.4 saves. The future home for this is `minecraftctl world scan` /
`world players`; until then use the bundled script.

```bash
S=.claude/skills/world-explore/scripts/scan.py
python3 $S halls     <save>...                 # villager job-site clusters
python3 $S portals   <save>...                 # portals (both dims), bed clusters, player beds/last pos
python3 $S builds    <save>... [--json DIR]    # player-built areas + sign text (~20s for a 6 GB world)
python3 $S mapconfig <save>...                 # suggested map-config.yml ranges
python3 $S players   <save>...                 # play time, km, blocks mined, deaths
python3 $S inventory <save>... --player NAME   # armor, inventory, ender chest, shulker contents
```

## Where saves are

- Local client: `~/Library/Application Support/minecraft/saves/<name>/`
- Server: `/srv/minecraft-server/<world>/world/` (usercache.json one level up gives player names).
  Run there, or pull a copy via the restic snapshots (see `docs/moving-worlds.md`).

## Reading the output

- **halls**: job-site POIs by profession. `librarian` = lectern, `farmer` = composter,
  `fisherman` = barrel. A tight, flat cluster of one profession is a trading hall; many
  composters at one Y is a villager crop farm; big "fisherman" clusters are usually barrel storage.
- **portals**: nether coords x8 = overworld. Nether portals at round distances along the
  axes (±1000, ±2000) mean a nether highway, and each end is an overworld outpost.
  Bed clusters are mostly natural villages; large ones near builds are breeders.
- **builds**: scored by hoppers, furnaces, shulkers, brewing stands and signs (things
  worldgen rarely makes). Sign text usually names the build: lead with it.
- **players**: play time is connected time, AFK included. Separate saves of the same world
  at different dates overlap, so don't add them up blindly.
- Player names: unknown UUIDs print as 8-char prefixes. Looking them up through Mojang's
  API sends the UUIDs to Mojang, so ask first.

## Map config ranges

`mapconfig` proposes `ranges` for `map-config.yml` (format: `docs/map-build-config.rst`):
one square per top build, villager cluster (>=15 job sites) and nether portal hub. Squares merge
only when one's center lies inside the other. It prints the region-tile cost per map;
ranges render with `zoomin` above the map default and are snapped out to 512-block regions
(`minecraftctl/pkg/maps/build.go`), so cost grows fast with radius.

Treat the output as a draft: rename ranges (names come from the most label-like sign, else
coordinates), drop ones nobody cares about, and paste into the world's `map-config.yml` by
hand. Don't overwrite an existing config. Tuning: `--top`, `--zoomin`, `--min-radius`, `--pad`.

## Keeping finds

Record finds as you go (coords, what's there, sign text) in a notes file the user chooses.
For a teleport book of the finds, see the `world-visit` skill.
