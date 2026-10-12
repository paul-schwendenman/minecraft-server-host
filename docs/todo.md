# TODO

Outstanding work, roughly in priority order. Details live in the linked plans.

## Now

- [ ] **Confirm autoshutdown powers off after the last SSH logout**: the new sshd-title check (`ami-0364c0b1da0d06f88`) skipped shutdown during SSH sessions on test and prod (2026-10-12). Test stopped after logout, but EC2 shows "User initiated" either way. On its next boot, `journalctl -b -1 -t autoshutdown` should end with a "shutting down now" line.
- [ ] **Autoshutdown hard backstop**, layers 1–3 ([autoshutdown-review.md](plans/autoshutdown-review.md#deferred-hard-backstop-not-implemented)):
  - [ ] `max-uptime.timer`: powers off after N hours no matter what (suggested 12h)
  - [ ] Boot watchdog: power off if RCON isn't answering 30–45 min after boot
  - Hardening is done (2026-10-10). Known gaps: the `StartLimitBurst` limit counts manual and `world switch` starts too (a 6th start inside 10 min needs `systemctl reset-failed`), and a world that crashes 2+ min after each start still loops.
  - Open decisions: the cap value, and whether the watchdog skips shutdown while someone is SSH'd in
- [ ] **Remove the legacy uid remaps** (`TODO`s in `mount-ebs.sh` and on `LegacyIDs` in `minecraftctl/pkg/backup/backup.go`):
  - [ ] `mount-ebs.sh`: ready to remove. Test and prod both booted on a 25565 AMI on 2026-10-12 and have no 996/997 files left. Keep it only if a volume might be restored from an EBS snapshot older than that.
  - [ ] `LegacyIDs`/`RemapOwners`: once no snapshot in either repo has files at 996/997 (prod's first all-25565 `all` snapshot is `9ba33f4c`, 2026-10-12; everything older still has them). Prune keeps a world's last snapshots however old they are, so check every snapshot (`restic ls --json <id>`), don't wait for rotation.
- [ ] **Uid drift fixes declined 2026-09-23**: shut down when `minecraft.env` can't be read. Revisit if an instance stays up again. (Fixing old owners at boot is now done for 996/997 by `mount-ebs.sh`; caddy's uid is still unpinned, covered by `configure-caddy.sh`'s recursive chown.)

- [ ] **Job-aware autoshutdown** ([plan](plans/shutdown-after-jobs-plan.md)): skip shutdown while map builds and backups run, so "start the job and disconnect" works without a hand-written waiter. Then `map build now --upload` and build progress in `map build status`.
- [ ] **Render only new or changed map ranges**: a build skips a map when `level.dat` is older than its manifest, so editing `map-config.yml` on an unchanged world does nothing unless you pass `--force`, which re-renders the whole map (old's overworld: ~58 min) to pick up a few small ranges (hit 2026-10-09 when the old/world.bak ranges were redone). Store a hash of each range (center, radius, zoom, options, format) in the map's `manifest.json`. On build, render ranges whose hash is new or changed even if the world hasn't changed, and skip the base map. Watch out: every range render rewrites `unmined.map.properties.js` with the range's bounds, which is why `buildMap` renders the base map last (`pkg/maps/build.go`). A ranges-only build has to save that file first and restore it afterwards, then re-apply the `maxZoom` patch. Maybe also add `map build now --ranges-only` as a manual escape hatch.

- [ ] **World switching** ([plan](plans/world-switching-plan.md)): phase 1 adds a **Play** button in the maps app. It calls `/start?world=`, the control lambda sets an `ActiveWorld` EC2 tag and starts the instance, and a `minecraft-active.service` boot unit starts that world. Phase 2 adds `minecraftctl world switch` for switching over SSH. Phase 3 switches from the UI while the server is running: a watcher on the instance calls `world switch`. The controls app stays unchanged. Every world's jar and map are already on prod (checked 2026-09-24).
- [ ] **Finish checking the 2026-10-10 fixes**: the rest passed on test 2026-10-12, and prod has been on the same changes since then (`ami-0364c0b1da0d06f88`).
  - A crash-looping world goes `failed` after 5 starts (checked) and autoshutdown then powers off (not tried, because it would shut down test).
  - Stop-backup lock timeout: a stop during `backup prune` ended cleanly, but prune (15s) finished before the stop backup needed the lock, so the 1m give-up path hasn't run. To test it, hold an exclusive lock for over a minute.
  - `create-world.sh` only chowns wrong-owner files: on either host's next boot, `default`'s first backup shouldn't report every file changed. The first 25565 boot doesn't count, because `mount-ebs.sh`'s 996/997 remap touched every file.
  - Boot-backup contention: first boots on fresh root volumes (no timer stamps to catch up on) don't count: test was 54s to ready with 0 backups before ready, and prod's userspace took 1min 9s, including the remap. Re-measure on prod's first start of a later day.

- [ ] **Review world log retention**: restic excludes `logs/` and `crash-reports/` for every world, so server logs only live on the data volume. (`old`'s history is already in the pre-rewrite sync at `s3://minecraft-backup-001/logs/`; nothing more to save there.) Decide how to keep logs going forward: include them in `backup create all`, or archive them separately.

## Later

- [ ] Per-world timers (`minecraft-world-backup@`, `minecraft-map-backup@`) are enabled by `world register` during cloud-init, after `timers.target` is reached, so on a replaced instance they stay inactive until the next boot (seen on test 2026-10-12). Start them as well as enabling them, or accept that the first boot only gets stop-time backups. The `enableWorldTimers` comment ("they'll start on their schedule") is wrong.
- [ ] **`minecraftctl backup rotate-password`** ([plan](plans/minecraftctl-backup-password-rotation-plan.md)): not urgent.
- [ ] **Test gaps** ([test_plan.md](plans/test_plan.md), Phase 4): Cobra command tests, `maps.go` with uNmINeD mocked, RCON integration, UI E2E. Go coverage is 37%; target is 70%.
- [ ] `map` JSON output: `TODO` at `minecraftctl/cmd/minecraftctl/map.go:343`.
- [ ] Remove the `rebuild-map.sh` / `build-map-manifests.sh` wrappers: point `minecraft-map-build@.service` and `minecraft-override-rebuild.conf` at `minecraftctl` directly (minecraftctl already supports globs, `--non-blocking` and `--update-index`, so the wrappers add nothing), then drop their bats and shellcheck tests. This is the last item in the [migration plan](plans/minecraftctl-migration-plan.md).
- [ ] `minecraftctl doctor`, and `--dry-run` on more than `config sync` ([minecraftctl-plan.md](plans/minecraftctl-plan.md), Phase 7).
- [ ] `minecraftctl map clean` (Phase 10). Include tiles in a format the map no longer uses. Switching `imageformat` re-renders the whole map (`pkg/maps/build.go`) but leaves the old-format tiles behind, and `backup-maps.sh` only adds to S3, so they pile up in both places. On 2026-10-10 the jpeg -> webp switch for world.bak and world.bak-1.19.2 left 19,092 jpeg tiles (~200 MB), which were deleted by hand on the server and in `s3://minecraft-prod-maps/maps/<world>/`. Better: delete them in the format-change path of `buildMap`, and have the map upload drop them from S3 (`--delete` scoped to the map's `tiles/`, or an explicit `aws s3 rm` of the old extension).

## Big ideas

- [ ] **`minecraftctl serve` API**: an API served from the game server itself. So far this is only one line in [minecraftctl-plan.md](plans/minecraftctl-plan.md) (Phase 10). It needs a design doc first: what it exposes, how it authenticates, and how it relates to the control lambda.
- [ ] **Self-bootstrapping `minecraftctl setup`** ([plan](plans/minecraftctl-setup-plan.md)): download the prebuilt CLI onto a fresh Linux box, run one command, and get a working server with autoshutdown (and optionally backups). Single machine only, no web UI or lambdas. Embed the unit files and scripts in the binary and have Packer call `setup`, so there's one install path. Autoshutdown's idle action becomes `poweroff` (EC2) or `stop` (home box/VPS). Wake-on-join comes later.
- [ ] **Create worlds from a UI**: pick a name, version, seed, difficulty and so on, then play it straight away. `minecraftctl` already has the building blocks (`jar download`, `world create --version --seed`). What's missing is a way to trigger them remotely: `minecraftctl serve`, or the same tag/watcher glue as [world switching](plans/world-switching-plan.md) phase 3.

## Housekeeping

- [ ] Delete `old` from test once you're happy with it on prod. Snapshots `b75cdcd0` (world) and `00ccb5fe` (all) are in `minecraft-prod-backups` (2026-09-24).

- [ ] Delete stale local branches. Five are gone. `replace-map-viewer` (5 commits) and `testing-backup` (16 commits) are still here, local-only and unmerged: decide whether to keep them or drop them.
- [ ] Pin the packer version in CI. check against the other versions in tool-versions too like terraform
