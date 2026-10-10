# TODO

Outstanding work, roughly in priority order. Details live in the linked plans.

## Now

- [x] **Dependabot PRs** (#150, #153–#156, #158, #159): all stale, closed 2026-09-23.
- [ ] **Autoshutdown hard backstop**, layers 1–3 ([autoshutdown-review.md](plans/autoshutdown-review.md#deferred-hard-backstop-not-implemented)):
  - [ ] `max-uptime.timer`: powers off after N hours no matter what (suggested 12h)
  - [ ] Boot watchdog: power off if RCON isn't answering 30–45 min after boot
  - [x] Harden (2026-10-10, not yet on an AMI): `TimeoutStartSec=2min` on `autoshutdown.service`, `timeout 30` around the RCON call, `StartLimitBurst=5` per 10 min on `minecraft@.service`, and `minecraft-stop-gate` skips the `ExecStopPost` backup unless the stop was clean or the server ran 5+ min. Gaps: the limit counts manual and `world switch` starts too (a 6th start inside 10 min needs `systemctl reset-failed`), and a world that crashes 2+ min after each start still loops.
  - Open decisions: the cap value, and whether the watchdog skips shutdown while someone is SSH'd in
- [x] **Check the uid 996 pin**: pinned in `create-minecraft-user.sh` (`df01b65`), and every `packer-build` run since (Sep 21–22, latest `d71e2db`) succeeded with it. Test and prod both run `ami-06396b7e9fb13ffb6` from that latest build (checked 2026-09-23).
- [ ] **Uid drift fixes declined 2026-09-23**: fix ownership on the volume at boot, pin caddy's uid, shut down when `minecraft.env` can't be read. Revisit if an instance stays up again.

- [ ] **Job-aware autoshutdown** ([plan](plans/shutdown-after-jobs-plan.md)): skip shutdown while map builds and backups run, so "start the job and disconnect" works without a hand-written waiter. Then `map build now --upload` and build progress in `map build status`.
- [ ] **Render only new or changed map ranges**: a build skips a map when `level.dat` is older than its manifest, so editing `map-config.yml` on an unchanged world does nothing unless you pass `--force`, which re-renders the whole map (old's overworld: ~58 min) to pick up a few small ranges (hit 2026-10-09 when the old/world.bak ranges were redone). Store a hash of each range (center, radius, zoom, options, format) in the map's `manifest.json`. On build, render ranges whose hash is new or changed even if the world hasn't changed, and skip the base map. Watch out: every range render rewrites `unmined.map.properties.js` with the range's bounds, which is why `buildMap` renders the base map last (`pkg/maps/build.go`). A ranges-only build has to save that file first and restore it afterwards, then re-apply the `maxZoom` patch. Maybe also add `map build now --ranges-only` as a manual escape hatch.
- [x] **Slow starts from boot-time backups** ([writeup](plans/boot-backup-contention.md)): `Persistent=true` backup timers catch up at boot and run alongside Java, so prod takes 60–79s to be ready instead of ~36s (measured 2026-10-09). Done 2026-10-10 (options 1 + 2): `RandomizedDelaySec` on the backup, prune and map timers, and `Nice`/`CPUWeight`/`IOWeight` on their services. Map builds during a switch weren't from `minecraft-override-rebuild.conf` (no install script installs it): `minecraft-map-build@.timer` used `OnBootSec=1min` + `Persistent=true`, so it fired at once on the first start after a rebuild. Now `OnActiveSec=5min`. Pending AMI test: re-measure with the writeup's script.
- [x] **Scheduled `backup create all`**: per-world backups only cover `<world>/world/`, so configs (`server.properties`, `ops.json`, `map-config.yml`) are only saved by manual `all` runs. (`all` now excludes `caddy/`, which made it exit 3.) Done 2026-10-10: the AMI now enables `minecraft-world-backup.timer` (Sunday 03:30, `minecraftctl backup create`) and `minecraft-world-prune.timer` (Sunday 04:00), which was never enabled before either. Pending AMI test.

- [x] **Snapshot the archive worlds on prod** (2026-09-25, `minecraft-prod-backups`): `old` `2c4d15e2`, `world.bak` `2569de9a`, `world.bak2` `092b5292`, `world.bak-1.19.2` `d3e627a5`, `all` `494c2982`.
- [ ] **World switching** ([plan](plans/world-switching-plan.md)): phase 1 adds a **Play** button in the maps app. It calls `/start?world=`, the control lambda sets an `ActiveWorld` EC2 tag and starts the instance, and a `minecraft-active.service` boot unit starts that world. Phase 2 adds `minecraftctl world switch` for switching over SSH. Phase 3 switches from the UI while the server is running: a watcher on the instance calls `world switch`. The controls app stays unchanged. Every world's jar and map are already on prod (checked 2026-09-24).
- [x] **World switch vs. RCON password mismatches**: on prod 2026-09-30, `old` had its own `rcon.password`. Java came up, but every RCON login failed, so the watcher waited the full 5 min, rolled back to `default`, and `minecraft@old` ended up `failed`. Fixed by hand with `config sync old`. Fixed in code 2026-10-10, not yet on an AMI:
  - [x] `minecraft@.service` runs `config sync %i` as `ExecStartPre`. `/etc/minecraft.env` is root-only, so the sync falls back to the `RCON_*` values `EnvironmentFile=` exports.
  - [x] `waitForRcon` gives up at once on `authentication failed`. The rollback world usually fails the same way, so the switch now ends in `ErrRollbackFailed` within seconds.
  - [x] The `ExecStop=` RCON lines are prefixed with `-`.
- [ ] **Check the RCON hardening on test** after the next AMI build:
  - `systemd-analyze verify` on `minecraft@.service` and its drop-ins.
  - Break a world's `rcon.password`, start it, and confirm `ExecStartPre` fixes it (the journal shows `RCON settings from: environment (...)`).
  - Point a world at a missing `server.jar`: it should go `failed` after 5 tries, and autoshutdown should then power off.
  - Stop a world with RCON broken: the unit should end `inactive`, not `failed`, with a fresh `level.dat`.
  - During a stop, confirm `ExecMainExitTimestampMonotonic` is set by the time `minecraft-stop-gate` runs. If it's 0, the gate logs "uptime unknown" and backs up every crash-loop iteration.

- [ ] **Review world log retention**: restic excludes `logs/` and `crash-reports/` for every world, so server logs only live on the data volume. (`old`'s history is already in the pre-rewrite sync at `s3://minecraft-backup-001/logs/`; nothing more to save there.) Decide how to keep logs going forward: include them in `backup create all`, or archive them separately.

## Later

- [ ] **`minecraftctl backup rotate-password`** ([plan](plans/minecraftctl-backup-password-rotation-plan.md)): not urgent.
- [ ] **Test gaps** ([test_plan.md](plans/test_plan.md), Phase 4): Cobra command tests, `maps.go` with uNmINeD mocked, RCON integration, UI E2E. Go coverage is 37%; target is 70%.
- [ ] `map` JSON output: `TODO` at `minecraftctl/cmd/minecraftctl/map.go:343`.
- [ ] Remove the `rebuild-map.sh` / `build-map-manifests.sh` wrappers: point `minecraft-map-build@.service` and `minecraft-override-rebuild.conf` at `minecraftctl` directly (minecraftctl already supports globs, `--non-blocking` and `--update-index`, so the wrappers add nothing), then drop their bats and shellcheck tests. This is the last item in the [migration plan](plans/minecraftctl-migration-plan.md).
- [ ] `minecraftctl doctor`, and `--dry-run` on more than `config sync` ([minecraftctl-plan.md](plans/minecraftctl-plan.md), Phase 7).
- [ ] `minecraftctl map clean` (Phase 10).

## Big ideas

- [ ] **`minecraftctl serve` API**: an API served from the game server itself. So far this is only one line in [minecraftctl-plan.md](plans/minecraftctl-plan.md) (Phase 10). It needs a design doc first: what it exposes, how it authenticates, and how it relates to the control lambda.
- [ ] **Self-bootstrapping `minecraftctl setup`** ([plan](plans/minecraftctl-setup-plan.md)): download the prebuilt CLI onto a fresh Linux box, run one command, and get a working server with autoshutdown (and optionally backups). Single machine only, no web UI or lambdas. Embed the unit files and scripts in the binary and have Packer call `setup`, so there's one install path. Autoshutdown's idle action becomes `poweroff` (EC2) or `stop` (home box/VPS). Wake-on-join comes later.
- [ ] **Create worlds from a UI**: pick a name, version, seed, difficulty and so on, then play it straight away. `minecraftctl` already has the building blocks (`jar download`, `world create --version --seed`). What's missing is a way to trigger them remotely: `minecraftctl serve`, or the same tag/watcher glue as [world switching](plans/world-switching-plan.md) phase 3.

## Housekeeping

- [ ] Delete `old` from test once you're happy with it on prod. Snapshots `b75cdcd0` (world) and `00ccb5fe` (all) are in `minecraft-prod-backups` (2026-09-24).

- [x] Checklists in `minecraftctl-plan.md` and `minecraftctl-migration-plan.md` checked against the code and updated (2026-09-23).
- [ ] Delete stale local branches. Five are gone. `replace-map-viewer` (5 commits) and `testing-backup` (16 commits) are still here, local-only and unmerged: decide whether to keep them or drop them.
- [x] `packer/readme.rst` is stale: it still lists `mcrcon` (no longer installed) and `map-rebuild.timer/service` (now `minecraft-map-build@`), and describes `rebuild-map.sh` as rendering with uNmINeD directly. Those were already gone; rechecked 2026-10-10 against the install scripts and units and fixed the stop drop-in (world backup only, no maps), the timer list and schedules, and the Null Builder section.
- [x] Low priority: on every new instance, `create-world.sh` runs `chown -R` on the Terraform world (`default`). chown bumps every file's ctime, so that world's next restic backup rereads all of it (prod 2026-09-29: 930 files, 1.4 GiB in 11s, 152 KiB stored). Other worlds aren't affected. Fix: only chown files with the wrong owner (`find ... \( ! -user minecraft -o ! -group minecraft \) -exec chown -h minecraft:minecraft {} +`), rather than `--ignore-ctime`. Done 2026-10-10 (`fix_owner` in `create-world.sh`, with bats tests). Pending AMI test.
- [x] Clean build state out of the AMI: the test instance's journal (2026-09-29) still had three ~5 min boots from Packer builds, so the AMI ships the builder's journal and probably its `/etc/machine-id`. At the end of `minecraft.pkr.hcl`, run `journalctl --rotate --vacuum-time=1s` and truncate `/etc/machine-id` so each instance starts with its own ID and an empty journal. Done 2026-10-10: the last `minecraft.pkr.hcl` step also runs `cloud-init clean`. Pending AMI test.
- [ ] Pin the packer version in CI. check against the other versions in tool-versions too like terraform
