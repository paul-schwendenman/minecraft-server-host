# Plan: RCON mismatch fixes + autoshutdown hardening

Untracked working plan for branch `paul-schwendenman/rcon-hardening`. Delete
before merging (or fold anything worth keeping into `docs/plans/`).

Two `docs/todo.md` items, picked together because they touch the same files
(`minecraft@.service`, the RCON client, `autoshutdown.*`) and the same failure
shape: a broken RCON or a crash-looping world keeps things hanging instead of
failing cleanly.

1. **World switch vs. RCON password mismatches**: all three sub-items.
2. **Autoshutdown hard backstop, layer 3 only** ("Harden"). Layers 1–2
   (`max-uptime.timer`, boot watchdog) stay open: they still have undecided
   values (cap hours, SSH exemption). Layer 3 has none.

**Stack:** this is the bottom branch. `paul-schwendenman/boot-backups`
(`~/orca/workspaces/minecraft-server-host/boot-backups/PLAN.md`) sits on top,
so one AMI build from that branch tests both.

Background: `docs/plans/autoshutdown-review.md` § "Deferred: hard backstop",
and the 2026-09-30 `old` incident in `docs/todo.md`.

---

## Part A: RCON password mismatches

### A1. Fail fast on `authentication failed`

**Now:** `waitForRcon` (`minecraftctl/pkg/worlds/switch.go:360`) treats every
`rconSend("list")` error as "not up yet" and retries every 5s until the timeout
(5 min). A wrong password never fixes itself.

**Change:**
- `pkg/rcon/client.go`: add `var ErrAuthFailed` (or re-export
  `rcon.ErrAuthFailed` from gorcon v1.4.0, `rcon.go:67`). In
  `NewClientWithConfig`, wrap so that `errors.Is(err, ErrAuthFailed)` works
  (it's already wrapped with `%w`, so check that gorcon returns its sentinel
  unwrapped from `Dial`; if it does, just document it and add the helper
  `rcon.IsAuthError(err)`).
- `waitForRcon`: on an auth error, return at once with a clear message:
  `"%s: RCON rejected the password; run 'minecraftctl config sync %s'"`.
  Rollback then kicks in straight away instead of after 5 min.
- Consider a new sentinel in `pkg/worlds` (`ErrRconAuth`?) so
  `world_switch.go` can give it its own exit code. Optional; the existing
  `ErrStartFailed` / `ErrRollbackFailed` wrapping may be enough.

**Tests** (`pkg/worlds/switch_test.go`): `waitUp` is already injectable on the
`switcher`, but `waitForRcon` itself calls `rconSend` directly. Make
`rconSend` a parameter (or a package var) so a test can return an auth
error and assert that `waitForRcon` returns without sleeping. Use a fake
`sleep`/clock rather than real 5s waits.

### A2. Sync RCON config before a world starts

**Now:** `config sync` logic lives inline in the Cobra command
(`cmd/minecraftctl/config.go:100-180`), and `--dry-run` defaults to **true**
(it prints "Use --dry-run=false to apply changes").

**Change:**
- Extract the sync into a package function, e.g.
  `config.SyncRcon(propsPath, envPath string, dryRun bool) (changes []Change, err error)`
  (`pkg/config` or `pkg/properties`; pick whichever doesn't create an import
  cycle). The command becomes a thin printer around it.
- Call it on **every** start path, not just `world switch`. The simplest hook
  is the unit itself:
  ```ini
  # minecraft@.service
  ExecStartPre=-/usr/local/bin/minecraftctl config sync %i --dry-run=false
  ```
  The `-` prefix means an unreadable `minecraft.env` (see the uid-drift
  incident) doesn't stop the world starting; it just logs. This covers
  `minecraft-active.service`, `world switch`, the watcher and manual
  `systemctl start` in one place.
- Check: `ExecStartPre` runs as `User=minecraft` under
  `ProtectSystem=full`. `server.properties` lives in
  `/srv/minecraft-server/%i`, which isn't under `/usr`, `/boot` or `/etc`, so it
  should be writable. `/etc/minecraft.env` needs to be readable by
  `minecraft`; verify the mode on prod.
- Maybe: also call `SyncRcon` in-process at the start of `switchWorld` (after
  `checkStartable`) so `--dry-run` switches can report "would fix RCON
  password". Nice to have, not required once ExecStartPre exists.

**Tests:** table test for `SyncRcon` (no changes / port differs / password
differs / enable-rcon off / env missing keys). Existing command tests, if any,
should still pass.

### A3. Don't let a broken RCON fail the stop

**Now:** all three `ExecStop=` lines in `minecraft@.service` go through RCON.
If RCON fails, the stop fails and the unit ends up `failed`.

**Change (recommended):** prefix each with `-`:
```ini
ExecStop=-/usr/local/bin/minecraftctl rcon send "say SERVER SHUTTING DOWN. Saving map..."
ExecStop=-/usr/local/bin/minecraftctl rcon send "save-all"
ExecStop=-/usr/local/bin/minecraftctl rcon send "stop"
```
If `stop` doesn't land, systemd sends SIGTERM after the ExecStop commands and
Java's shutdown hook saves the world (`SuccessExitStatus=143` already covers
the exit code). Also consider `TimeoutStopSec=` (e.g. 90s) so a wedged JVM
gets SIGKILL in bounded time instead of the default 90s… check what the
current default is on the AMI and whether a large world's save needs longer.

Alternative: drop the ExecStop lines and rely on SIGTERM only. That's simpler,
but you lose the in-game warning. Keep the `-` version.

---

## Part B: Autoshutdown hardening (layer 3)

### B1. Bound `autoshutdown.service`

`packer/scripts/minecraft/autoshutdown/autoshutdown.service`:
```ini
TimeoutStartSec=2min
```
A oneshot that hangs blocks every later timer firing
(`OnUnitActiveSec` counts from activation, and the unit never finishes).

### B2. `timeout` around the RCON call

`autoshutdown.sh`:
```bash
OUTPUT=$(timeout 30 minecraftctl rcon send "list" || true)
```
Note: gorcon already has 5s dial/read deadlines (`DefaultDialTimeout`,
`DefaultDeadline`), so this is defence in depth against something hanging
*outside* the socket (env loading, a future code change). It's cheap, so do it.

Related: when RCON auth fails, the script currently parses no count,
logs it, sets `COUNT=0` and counts towards shutdown. That's the right
fail-safe direction; leave it, but make the log line say "RCON auth failed"
when the output/err says so (A1 gives a recognisable message).

### B3. End crash loops: `StartLimitBurst` on `minecraft@.service`

```ini
[Unit]
StartLimitIntervalSec=10min
StartLimitBurst=5
```
With `Restart=on-failure` / `RestartSec=20`, a JRE or world crash currently
loops forever, which keeps `minecraft@X` "activating"/"running" often enough
that autoshutdown may never see zero running services. After 5 failures in
10 min the unit goes `failed`, `list-units --state=running` returns 0, and
autoshutdown powers off on its next tick.

Check that `world switch` rollback and `waitForRcon` already treat `failed` as
terminal (they do: `switch.go:369`).

### B4. Skip ExecStopPost backups/map builds on failed starts

Two drop-ins run on every stop, including each crash-loop iteration:
- `worlds/minecraft-override-world-backup.conf` → `minecraftctl backup create %i`
- `maps/minecraft-override-rebuild.conf` → `rebuild-map.sh`, `build-map-manifests.sh`

systemd passes `$SERVICE_RESULT`, `$EXIT_CODE`, `$EXIT_STATUS` to
`ExecStopPost`. Gate on them:
```ini
ExecStopPost=/bin/sh -c '[ "$SERVICE_RESULT" = success ] || { echo "skipping backup: $SERVICE_RESULT"; exit 0; }; exec /usr/local/bin/minecraftctl backup create %i'
```
**Open decision:** `SERVICE_RESULT != success` also covers a world that
ran fine for hours and then crashed. Skipping the backup there loses the
session. Options:
- (a) gate on `success` only. Simple, but misses the crash-after-play backup.
- (b) skip only if the unit was up for less than N minutes. That needs a
  timestamp: an `ExecStartPost` touches `/run/minecraft/%i.started`, and
  ExecStopPost compares mtimes. More moving parts.
- (c) put the gate in minecraftctl: `backup create --skip-if-unchanged` (restic
  already dedups, so the cost of a pointless backup is mostly time and CPU at
  boot, not storage).
Leaning (a) for map builds (a crash never warrants a re-render) and (b) or (c)
for backups. Decide before implementing.

The map-rebuild drop-in still goes through the wrapper scripts; don't
remove them here (that's its own todo item), just gate them.

---

## Files touched

| File | Parts |
|------|-------|
| `minecraftctl/pkg/rcon/client.go` (+ test) | A1 |
| `minecraftctl/pkg/worlds/switch.go` (+ `switch_test.go`) | A1 |
| `minecraftctl/cmd/minecraftctl/config.go` | A2 |
| `minecraftctl/pkg/config/…` or `pkg/properties/…` (new `SyncRcon` + test) | A2 |
| `packer/scripts/minecraft/minecraft@.service` | A2, A3, B3 |
| `packer/scripts/minecraft/autoshutdown/autoshutdown.service` | B1 |
| `packer/scripts/minecraft/autoshutdown/autoshutdown.sh` | B2 |
| `packer/scripts/minecraft/worlds/minecraft-override-world-backup.conf` | B4 |
| `packer/scripts/minecraft/maps/minecraft-override-rebuild.conf` | B4 |
| `packer/tests/*.bats` | any unit-file assertions that exist |
| `docs/todo.md`, `docs/plans/autoshutdown-review.md` | tick off / note layer 3 done |

## Suggested order / commits

1. A1: RCON auth fail-fast (Go only, fully unit-testable).
2. A2: extract `SyncRcon` + `ExecStartPre`.
3. A3 + B3: `minecraft@.service` (`-` on ExecStop, StartLimit*, maybe TimeoutStopSec).
4. B1 + B2: autoshutdown timeout.
5. B4: ExecStopPost gating (after the open decision).
6. Docs: todo + autoshutdown-review.

## Validation

- `cd minecraftctl && go fmt ./... && go test ./...`
- `cd packer && packer fmt . && bats tests/*.bats` (shellcheck runs in there)
- `systemd-analyze verify` the changed units on a test instance, since unit
  syntax errors aren't caught locally.
- On **test** after the AMI build:
  - Break the password on purpose: change `rcon.password` in a world's
    `server.properties`, `systemctl start minecraft@<w>`, confirm
    ExecStartPre fixes it and RCON answers.
  - Bypass the sync (`minecraftctl config set`, then start with the
    ExecStartPre removed, or point it at a bad env) and run `world switch`:
    it should roll back in seconds, not 5 min.
  - Crash loop: point a world at a missing `server.jar`, start it, and
    confirm it reaches `failed` after 5 tries and autoshutdown then powers off.
  - Stop with RCON broken: unit ends `inactive`, not `failed`, and the world
    still saved (check `level.dat` mtime).

## Open decisions

- B4: how to gate backups on crash-after-play (a/b/c above).
- A3: `TimeoutStopSec` value, or leave the default.
- A2: whether `world switch --dry-run` should also report RCON drift.
