---
name: world-explore
description: Find things inside a Minecraft Java save without opening the game - where players spent time, bases, trading halls, storage and redstone builds, beacons, map walls, named pets and their owners, nether portals and ice-road highways, sign text, per-player play time and deaths, and player inventories/ender chests - and turn the finds into suggested map-config.yml ranges. Use when asked where a build is in a world, what's in a world, who played how much, what a player was carrying, or to pick focused render areas (ranges) for a world's map config.
---

# World explore

Read-only analysis of a save directory (the one with `level.dat`). Pure Python 3, no deps.
Handles 1.16 through 1.20.4 saves. The future home for this is `minecraftctl world scan` /
`world players`; until then use the bundled script.

```bash
S=.claude/skills/world-explore/scripts/scan.py
python3 $S hotspots  <save>...                 # where players spent time (fast, start here)
python3 $S builds    <save>... [--json DIR]    # player-built areas + signs/banners/heads/discs/books
python3 $S entities  <save>...                 # named mobs, pets by owner, map walls, armor stands, master villagers
python3 $S blocks    <save>...                 # redstone / decor / treasure blocks; nether ice roads
python3 $S roof      <save>...                 # nether roof builds, roof portals and entities
python3 $S halls     <save>...                 # villager job-site clusters
python3 $S portals   <save>...                 # portals (both dims), lodestones, bed clusters, player beds/last pos
python3 $S maps      <save>...                 # map items: explored areas, locked copies, where maps hang
python3 $S mapconfig <save>...                 # suggested map-config.yml ranges
python3 $S players   <save>...                 # play time, km, blocks mined, deaths, last death spot (1.19+)
python3 $S inventory <save>... --player NAME   # armor, inventory, ender chest, shulker contents
```

Rough cost on a 6 GB world (8 cores): hotspots, halls, portals, maps, players a few seconds;
builds, entities, blocks 10-25 s each. A good first pass: `hotspots`, then `builds` and
`entities`, and read them side by side. A hotspot with a build and named pets next to it is a home.

Run scripts from a file, never `python3 - <<EOF`: multiprocessing on macOS can't start
workers from stdin, and the pool hangs.

## Where saves are

- Local client: `~/Library/Application Support/minecraft/saves/<name>/`
- Server: `/srv/minecraft-server/<world>/world/` (usercache.json one level up gives player names).
  Run there, or pull a copy via the restic snapshots (see `docs/moving-worlds.md`).

## Reading the output

- **hotspots**: chunk `InhabitedTime`, hours any player was within ~8 chunks (not summed
  across players). A base is a plateau. The script reports one spot per local peak. It also catches
  builds with no block entities (houses, statues) that `builds` misses.
- **builds**: weights: beacon 10, conduit 8, ender chest 6 (never worldgen), shulker,
  brewing, enchanting 4, jukebox, head, book-on-lectern, hopper 3, comparator, banner, sign 2.
  Worldgen look-alikes count 0: spawners, unopened loot chests (`LootTable` still set) and
  plain banners (villages; only patterned or named banners score). A spawner with hoppers
  around it is flagged `(grinder?)`.
- **entities**: pets are tamed animals only (wolves/cats/parrots by Owner, horse family by
  Tame); projectiles also carry an Owner and are ignored. The owner tells you whose base a spot is.
  `map wall` = item frames holding filled maps. `master villagers` = level 5, a working trading hall.
  Chest minecarts are skipped (mineshafts), hopper minecarts kept.
- **blocks**: reads only block palette entries (`Name` tags), so items in chests don't
  count. Chunks within 128 blocks of ancient cities and trail ruins are dropped, because both
  generate lamps, pistons, comparators and glazed terracotta. In the nether, long thin
  `ice_road` clusters are highways: the x/z extents give their length.
- **roof**: nether chunks with blocks above the bedrock ceiling (y >= 128), where vanilla
  generates only mushrooms (ignored). Also lists roof portals and roof entities. Magma-block
  platforms or hundreds of portal blocks = gold farm. Check what height a nether build is at:
  the usual `nether` map renders with `topY: 68` and can't show anything on the roof.
  `mapconfig` emits a separate `nether_roof` map (`bottomY: 127`, `topY: 255`) when the roof has builds.
- **maps**: `locked` maps are cartography-table copies (map walls, map art). Clusters of map
  centers show where people explored at each scale. "Hung in item frames" are map walls.
- **halls**: job-site POIs by profession. `librarian` = lectern, `farmer` = composter,
  `fisherman` = barrel. A tight, flat cluster of one profession is a trading hall; many
  composters at one Y is a villager crop farm; big "fisherman" clusters are usually barrel storage.
- **portals**: nether coords x8 = overworld. Nether portals at round distances along the
  axes (±1000, ±2000) mean a nether highway, and each end is an overworld outpost.
  Bed clusters are mostly natural villages; large ones near builds are breeders.
- Sign text usually names a build: lead with it. `--json` dumps every block entity as
  `[kind, x, y, z, label]`, where label is sign text, banner name, head owner, disc, book title or spawner mob.
- **players**: play time is connected time, AFK included. Separate saves of the same world
  at different dates overlap, so don't add them up blindly.
- Player names: unknown UUIDs print as 8-char prefixes. Looking them up through Mojang's
  API sends the UUIDs to Mojang, so ask first.

## Map config ranges

`mapconfig` proposes `ranges` for `map-config.yml` (format: `docs/map-build-config.rst`):
one square per top build, villager cluster (>=15 job sites), hotspot (>= `--min-hours`, default 24)
and nether portal hub. A hotspot never names a merged range: builds and halls do. Squares merge
only when one's center lies inside the other. It prints the region-tile cost per map;
ranges render with `zoomin` above the map default and are snapped out to 512-block regions
(`minecraftctl/pkg/maps/build.go`), so cost grows fast with radius.

Treat the output as a draft: rename ranges (names come from the most label-like sign, else
coordinates), drop ones nobody cares about, and paste into the world's `map-config.yml` by
hand. Don't overwrite an existing config. Tuning: `--top`, `--zoomin`, `--min-radius`, `--pad`.

## Keeping finds

Record finds as you go (coords, what's there, sign text) in a notes file the user chooses.
For a teleport book of the finds, see the `world-visit` skill.
