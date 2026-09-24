# TODO

Outstanding work, roughly in priority order. Details live in the linked plans.

## Now

- [x] **Dependabot PRs** (#150, #153–#156, #158, #159): all stale, closed 2026-09-23.
- [ ] **Autoshutdown hard backstop**, layers 1–3 ([autoshutdown-review.md](plans/autoshutdown-review.md#deferred-hard-backstop-not-implemented)):
  - [ ] `max-uptime.timer`: powers off after N hours no matter what (suggested 12h)
  - [ ] Boot watchdog: power off if RCON isn't answering 30–45 min after boot
  - [ ] Harden: `TimeoutStartSec=` on `autoshutdown.service`, `timeout` around the RCON call, `StartLimitBurst` on `minecraft@.service`, skip `ExecStopPost` backups when a start fails
  - Open decisions: the cap value, and whether the watchdog skips shutdown while someone is SSH'd in
- [x] **Check the uid 996 pin**: pinned in `create-minecraft-user.sh` (`df01b65`), and every `packer-build` run since (Sep 21–22, latest `d71e2db`) succeeded with it. Test and prod both run `ami-06396b7e9fb13ffb6` from that latest build (checked 2026-09-23).
- [ ] **Uid drift fixes declined 2026-09-23**: fix ownership on the volume at boot, pin caddy's uid, shut down when `minecraft.env` can't be read. Revisit if an instance stays up again.

- [ ] **Job-aware autoshutdown** ([plan](plans/shutdown-after-jobs-plan.md)): skip shutdown while map builds and backups run, so "start the job and disconnect" works without a hand-written waiter. Then `map build now --upload` and build progress in `map build status`.
- [ ] **Scheduled `backup create all`**: per-world backups only cover `<world>/world/`, so configs (`server.properties`, `ops.json`, `map-config.yml`) are only saved by manual `all` runs. Also exclude `caddy/` so `all` stops exiting 3.

- [ ] **Review world log retention**: restic excludes `logs/` and `crash-reports/` for every world, so server logs only live on the data volume. `old` has 934 logs (2020-03-21 to 2025-10-24, 3.9 MB) that are in no backup. Before deleting `old` from test, archive them (laptop and/or S3). Then decide how to keep logs going forward: include them in `backup create all`, or archive them separately.

## Later

- [ ] **`minecraftctl backup rotate-password`** ([plan](plans/minecraftctl-backup-password-rotation-plan.md)): not urgent.
- [ ] **Test gaps** ([test_plan.md](plans/test_plan.md), Phase 4): Cobra command tests, `maps.go` with uNmINeD mocked, RCON integration, UI E2E. Go coverage is 37%; target is 70%.
- [ ] `map` JSON output: `TODO` at `minecraftctl/cmd/minecraftctl/map.go:343`.
- [ ] Remove the `rebuild-map.sh` / `build-map-manifests.sh` wrappers: point `minecraft-map-build@.service` and `minecraft-override-rebuild.conf` at `minecraftctl` directly (minecraftctl already supports globs, `--non-blocking` and `--update-index`, so the wrappers add nothing), then drop their bats and shellcheck tests. This is the last item in the [migration plan](plans/minecraftctl-migration-plan.md).
- [ ] `minecraftctl doctor`, and `--dry-run` on more than `config sync` ([minecraftctl-plan.md](plans/minecraftctl-plan.md), Phase 7).
- [ ] `minecraftctl map clean` (Phase 10).

## Big ideas

- [ ] **`minecraftctl serve` API**: an API served from the game server itself. So far this is only one line in [minecraftctl-plan.md](plans/minecraftctl-plan.md) (Phase 10). It needs a design doc first: what it exposes, how it authenticates, and how it relates to the control lambda.

## Housekeeping

- [ ] Delete `old` from test once you're happy with it on prod. Snapshots `b3d4542b` (world) and `69bbeeb7` (all) are in `minecraft-prod-backups` (2026-09-24). Archive its logs first (see "Review world log retention").

- [x] Checklists in `minecraftctl-plan.md` and `minecraftctl-migration-plan.md` checked against the code and updated (2026-09-23).
- [ ] Delete stale local branches. Five are gone. `replace-map-viewer` (5 commits) and `testing-backup` (16 commits) are still here, local-only and unmerged: decide whether to keep them or drop them.
- [ ] `packer/readme.rst` is stale: it still lists `mcrcon` (no longer installed) and `map-rebuild.timer/service` (now `minecraft-map-build@`), and describes `rebuild-map.sh` as rendering with uNmINeD directly.
