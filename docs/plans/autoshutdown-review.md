# Autoshutdown review

Review of `packer/scripts/minecraft/autoshutdown/autoshutdown.sh` after two
incidents where a prod instance stayed up for hours with nobody playing.

## How it works

`autoshutdown.timer` fires 5 minutes after boot and every 5 minutes after the
previous run. Each run of `autoshutdown.service` (oneshot, `User=minecraft`)
walks this decision tree:

| # | Condition | Action |
|---|-----------|--------|
| 1 | `who` shows a `pts/` session | Remove idle marker, exit 0 (stay up) |
| 2 | No `minecraft@*.service` with sub-state `running` | Power off immediately |
| 3 | `/etc/minecraft.env` unreadable / `minecraftctl` missing | Exit 1 (stay up) |
| 4 | RCON `list` fails or is unparseable | Treat as 0 players |
| 5 | 0 players, no idle marker | Create marker ("first check") |
| 6 | 0 players, marker present | Power off ("second check") |
| 7 | Players > 0 | Remove marker |

Two consecutive idle checks (5-10 minutes) are required on the RCON path. The
"no service running" path has no grace period.

## Incident 1: JRE too old for the server jar

The `minecraft@.service` unit crash-loops (`Restart=on-failure`,
`RestartSec=20`, which is longer than the default 10s start-limit window, so it
never trips to `failed`). `java` exits immediately, so the unit is almost always
in `auto-restart` / `stop-post`, not `running`. By the code, check #2 should
have powered the box off at the first timer run.

The journal for that instance was lost when Terraform replaced it, so **the
cause is unconfirmed**. Candidates that were not ruled out:

- The checker wedged. `Type=oneshot` has no start timeout, and
  `OnUnitActiveSec` counts from the last activation, so a hung
  `minecraftctl rcon send` stops the timer permanently.
- Something in the RCON path failed (see incident 2, which is a confirmed
  cause of the same symptom on that path).
- `poweroff` was issued but the OS hung in shutdown (`ExecStopPost` runs
  `minecraftctl backup create` and others on every stop, including each
  crash-loop iteration).

Also worth checking in S3: every crash-loop iteration runs the `ExecStopPost`
backup hooks, so a long loop may have produced snapshot spam.

## Incident 2: prod stayed up after players left (confirmed)

Read from the persistent journal on prod (boot `-1`, Sep 20 18:01 to Sep 21
01:24 UTC):

- 18:01-20:04 players online; 20:04-00:12 an SSH session was open (48 expected
  skips).
- From 00:16, thirteen consecutive runs logged `No players - first check` and
  then failed: `touch: cannot touch '/srv/minecraft-server/no_one_playing':
  Permission denied`, `status=1/FAILURE`. `set -e` aborted the script, the
  marker was never created, and the "second check" was unreachable.

**Root cause.** The marker lived at the root of the persistent EBS volume, which
was owned by numeric `997:997`. Most files on the volume carry that owner from
an older AMI where `minecraft` was uid 997. `create-minecraft-user.sh` used an
unpinned `useradd -r`, so uid/gid depend on what was installed first; the
current AMI produced `minecraft` = 996 and `caddy` = 997, so the volume root
belonged to `caddy`. `create-world.sh` only chowns the world directory, never
the volume root. A brand-new volume (`root:root`) has the same problem, so the
two-strike path likely never worked on the test environment either.

## Changes

1. **State off the EBS volume.** The marker is now
   `/run/autoshutdown/no_one_playing`, created for the service user by
   `RuntimeDirectory=autoshutdown` (`RuntimeDirectoryPreserve=yes`). It is
   tmpfs, so it is always writable and cannot survive a reboot.
2. **Fail toward shutdown, loudly.** If the marker cannot be written, the check
   is counted as the second strike, logged at `user.err`, and the box powers
   off. Trade-off: with a dead RCON plus an unwritable state dir this shuts
   down after one idle check instead of two.
3. **Stop ownership drift.** `minecraft` is pinned to uid/gid 996 (matches the
   current AMI and the live `default/` and `maps/` data); the Packer build
   fails if that ID is taken. `mount-ebs.sh` does a non-recursive chown of the
   volume root on first boot of each new instance.
5. **Surface failures.** An `ERR` trap logs `Unexpected failure at line N`;
   `mc-healthcheck.sh` reports a failed `autoshutdown.service` and an
   unwritable `/srv/minecraft-server`. This is logging, not alerting; no
   notification channel exists.

Tests: `packer/tests/autoshutdown.bats` moved to the new state dir and gained
cases for state-dir location, state-dir creation, the unwritable-state-dir
regression, and the ERR trap.

## Rollout notes

- These are AMI-baked files: prod is unchanged until a new AMI is built and
  deployed. Until then, `sudo chown minecraft:minecraft /srv/minecraft-server`
  (non-recursive) fixes the immediate symptom on the running instance.
- The uid 996 pin is not validated by a real `packer build` yet.
- Old files on the volume that are `997:997` (jars, logs, `minecraft.env`,
  `world.bak*`) stay mis-owned. They are world-readable and nothing currently
  needs to write them.

## Open: hard backstop (not implemented)

None of the above stops the *next* unknown bug from leaving the instance up
indefinitely. There is no max-uptime limit, alarm, or scheduled stop in
`infra/`; the only remote stop is the manual `/api/stop`. Options to discuss:

- Never-healthy limit: power off if RCON has not answered within N minutes of
  boot (covers a crash-looping server such as incident 1).
- Absolute uptime cap that the SSH check cannot override.
- Make the SSH skip expire, or cap how long it can defer shutdown.
- Bound the checker: `TimeoutStartSec=` on the unit and a `timeout` around
  `minecraftctl rcon send`, so a hung RCON call cannot wedge the timer.
- Fail the unit fast: `StartLimitBurst` / `StartLimitIntervalSec` so a
  crash-looping `minecraft@` ends in `failed`, and stop the `ExecStopPost`
  backup hooks from running on failed starts.
