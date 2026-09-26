# Plan: Switch Which World the Server Runs

Status: **phase 1 implemented** (2026-09-25), not yet deployed or tried on test.
The plan was reviewed on 2026-09-25; the phase 1 changes it asked for (marked
**Review** below) are in the code too. **Phase 2 implemented** (2026-09-25),
not yet tried on test. Phase 3 is proposed. Written 2026-09-24.

## Problem

Prod holds every world we've played: `old` (the original 1.16.4 world),
`world.bak`, `world.bak2`, `world.bak-1.19.2` and `default`, the current one.
We want to play the old ones again sometimes, without SSH.

How it works today:

- Each world has its own `minecraft@<world>.service`. At boot, systemd starts
  every world whose service is enabled (`WantedBy=multi-user.target`).
  `minecraftctl world register` enables it, so today that's `default`.
- **Only one world can run at a time.** Every world uses 25565 and RCON 25575,
  and each takes a 1.5 GB heap on a t3.medium. Switching worlds is doable;
  running them side by side isn't.
- The control lambda's `/start` only starts the EC2 instance. It has no world
  parameter, and its role can only `StartInstances`, `StopInstances` and
  `DescribeInstances`.
- Switching by hand works, but the change sticks across reboots:

  ```bash
  sudo systemctl disable --now minecraft@default
  sudo systemctl enable --now minecraft@world.bak2
  ```

## Proposal

Store the active world in one place, an EC2 tag, instead of in whichever
`minecraft@` services happen to be enabled. Build it in three phases:

- **Phase 1: choose the world at start.** Pick a world in the UI. The control
  lambda sets the tag and starts the instance, and a boot unit on the instance
  starts that world. Switching while the server is running means stopping it
  first.
- **Phase 2: `minecraftctl world switch`.** Switch worlds on a running server
  from the CLI, over SSH.
- **Phase 3: switch from the UI while the server is running**, without a
  restart: a watcher on the instance, the lambda and the maps app.

World selection lives in the maps app (`minecraft-ui/apps/worlds`): a **Play**
button on each world page, and a split **Start** button on the **Controls**
page (see open questions). The legacy `apps/manager` doesn't change; its plain
Start boots whatever world the tag already says.

## Phase 1: choose the world at start

### 1. Active world stored in an EC2 tag

The instance carries a tag `ActiveWorld=<name>`. Reasons to use a tag:

- It can be set while the instance is stopped. A file on the data volume can't
  be written then.
- The instance can read it at boot from instance metadata, with no API call and
  no extra IAM permissions, as long as tags are exposed to metadata.

Terraform (`infra/modules/mc_stack`, `aws_instance.minecraft`):

```hcl
metadata_options {
  http_tokens            = "required"
  instance_metadata_tags = "enabled"
}
```

Roll it out together with the new AMI, which replaces the instance anyway, so
it doesn't matter whether `metadata_options` alone would update it in place.

A tag missing or empty means the default world (see below).

**Review: keep Terraform from removing the tag.** `aws_instance.minecraft` sets
`tags = { Name = ... }`, so Terraform treats `ActiveWorld` as drift and deletes
it on the next `apply`. Add it to `ignore_changes`, and check with
`terraform plan` on a tagged instance that no tag change is planned:

```hcl
lifecycle {
  ignore_changes = [associate_public_ip_address, tags["ActiveWorld"]]
}
```

**Persistence.** The selection is sticky: it survives stop/start cycles and
reboots. Replacing the instance (a new AMI) resets it to the default world,
because the new instance has no tag. That reset is intentional; pick the world
again after a replacement.

### 2. Boot unit that starts only the active world

New `minecraft-active.service` (oneshot, `RemainAfterExit=yes`,
`WantedBy=cloud-final.service`, like `dyndns.service`):

1. Read `ActiveWorld` from IMDSv2
   (`/latest/meta-data/tags/instance/ActiveWorld`), falling back to
   `MC_DEFAULT_WORLD` from `/etc/minecraft.env` (written by `user_data` from
   Terraform's `world_name`), or `default` if that isn't set.
2. Check that the world can start. If it can't, log it and fall back to the
   default world, so a typo can't leave the server up with no world running
   (autoshutdown would power it off at the next check). If the default can't
   start either, fail without starting anything.
3. `systemctl start minecraft@<world>.service`.

**Review: what "can start" means.** The name must be a plain directory name
(no `/`, not starting with `.`), and the world dir must have:

- `eula.txt` with `eula=true`: without it Minecraft exits straight away;
- `server.properties` with `enable-rcon=true`: Minecraft would start without
  it (it writes a default file with RCON off), but autoshutdown reads the
  player count over RCON and would power off with people playing, and
  `ExecStop` saves and stops the world over RCON; and
- `server.jar` resolving to a file that exists (a symlink into
  `/opt/minecraft/jars/`). This catches a missing jar at boot instead of in a
  restart loop.

**Not** `world/level.dat`: Minecraft writes that on the world's first start,
so a world made by `world create` doesn't have one yet. (The first version
checked `level.dat`, which would have started nothing on an empty data volume.)
Test both cases:

- an **empty data volume**: `user_data` creates `default` and the boot unit
  starts it, generating the world;
- a **replacement instance** with the existing prod-like volume: the tagged
  world starts, and a missing tag gives `default`.

**Ordering on first boot.** On a new instance the data volume isn't in
`/etc/fstab` until `mount-ebs.sh` runs from `user_data`, which is late in boot
(cloud-init's final stage). Today that doesn't matter, because `register`,
called from `user_data`, starts the world itself. Once it stops doing that,
`minecraft-active.service` could run before the volume is mounted, find no
world, start nothing, and autoshutdown would power off within 5 minutes. So
order the unit `After=cloud-final.service`, plus
`RequiresMountsFor=/srv/minecraft-server` for later boots. cloud-final runs on
every boot but only runs the user scripts on the first, so the delay on later
boots is small. Test the first boot on test specifically.

Worlds are no longer enabled one by one:

- `minecraftctl world register` / `world create` stop enabling
  `minecraft@<world>`. They still enable the map-build, world-backup and
  map-backup timers.
- **Review: enable timers, never start them.** `minecraft-map-build@.timer` has
  `Requires=minecraft@%i.service` and `WantedBy=minecraft@%i.service`, so
  *starting* it starts that world's server. Enabling only adds it to the
  world's `.wants`, so it runs while that world runs, which is what we want.
  `register` and `create` only enable. `minecraftctl map build enable <world>`
  used `EnableNow`, which would start a second world alongside the running
  one; it now enables, and starts the timer only if that world is running.
- **Review: register every world on a new instance.** `user_data` only
  registers `world_name`, so after an instance replacement the other worlds
  have no backup timers. `create-world.sh` (run from `user_data`) now also
  registers every other world on the volume (any dir with
  `server.properties`), warning and carrying on if one fails. This also gives
  every playable world a daily backup (see open questions).
- No migration is needed. This ships as a new AMI, and changing `ami` replaces
  the instance, so the root volume (where `systemctl enable` symlinks live)
  starts with no world enabled. The one thing that re-enables a world is
  `user_data`: every new instance runs `create-world.sh <world_name>`, which
  calls `minecraftctl world register`. So the `register` change has to ship in
  the same AMI as `minecraft-active.service`.

Nothing else needs to change. `autoshutdown.sh` and `mc-healthcheck.sh` already
look at any `minecraft@*` unit.

### 3. Control lambda: `POST /start?world=<name>`

- `world` is optional. Without it, `/start` does what it does today, so the
  controls app keeps working unchanged.
- With it:
  1. Validate the name against `world_manifest.json` in the maps bucket.
     Unknown name: 400; manifest unreadable: 503. The `api_lambda` module
     receives `map_bucket_name`, but the control lambda didn't get it as an
     environment variable; pass `MAPS_BUCKET` and `MAP_PREFIX` (done in the
     phase 1 code).
  2. If the instance is running or pending and `ActiveWorld` is a different
     world: 409 ("stop the server first"). Same world: nothing to do, return
     the usual response.
  3. `CreateTags` `ActiveWorld=<name>` on the instance.
  4. `StartInstances`.
- Accepted edge cases, with no extra coordination in phase 1:
  - Two concurrent requests are last-write-wins on the tag.
  - The tag can differ from the running world for a while: after the boot unit
    falls back to the default, or after a manual swap over SSH. `/status`
    reports the tag, not what's actually running.
- `/status` also returns the current `ActiveWorld` (read from the instance's
  tags, which `DescribeInstances` already includes).
- IAM (`infra/modules/api_lambda`): add `ec2:CreateTags` on the instance ARN,
  with `aws:TagKeys` limited to `ActiveWorld`. The role can already read the
  maps bucket.

### 4. UI: "Play this world" in the maps app

- The maps app already fetches `world_manifest.json` through the worlds lambda,
  so every world in the list and on its detail page can get a **Play** button
  that calls `POST /api/start?world=<name>`. Both apps route `/api/*` to the
  same control API, so this needs no infra change.
- Show the active world and the server state (from `/api/status`), so it's
  clear which world is running. Disable **Play** on other worlds while the
  server is running, and explain why, instead of relying on the 409.
- This is the first write action in the maps app. The control API has no auth
  today, so anyone who can open the maps site can start the server, the same
  as anyone who can open the controls app.

Switching without the UI still works: set the tag with
`aws ec2 create-tags --resources <instance-id> --tags Key=ActiveWorld,Value=<name>`
and start the server. Over SSH, `systemctl stop minecraft@<a>` and
`systemctl start minecraft@<b>` swap worlds until the next boot, when the tag
wins again.

## Switching while the server is running (phases 2 and 3)

The lambda can't reach services on the instance, so something on the instance
has to do the switch. Options considered:

| Option | How the switch happens | Time | New pieces |
|---|---|---|---|
| **A. Set the tag and reboot** | Set `ActiveWorld`, then `RebootInstances`. The reboot is graceful, so `minecraft@`'s `ExecStop` saves the world, and the phase 1 boot unit reads the new tag. | ~1–2 min | Almost nothing. A reboot keeps the public IP, so DNS stays valid. |
| **B. Watch the tag on the instance** | A timer checks `ActiveWorld` in metadata every minute or so and switches when the tag changes. | ≤1 min plus world load | A small timer. Whatever changes the world only writes the tag. |
| **C. SSM Run Command** | The lambda runs `minecraftctl world switch <name>` on the instance through SSM. | Seconds plus world load | The SSM agent, an instance role policy, and IAM for `ssm:SendCommand`. A whole new way to reach the instance. |
| **D. `minecraftctl serve`** | An API on the instance does the switch. | Instant | The large "Big ideas" item in the TODO. |

**Chosen: B.** Boot and a live switch go through the same code: at boot the
check simply starts whatever the tag says. **A** is the fallback if B turns out
to be fiddly.

Before building B, check on test that a tag change reaches instance metadata
without a reboot. AWS says it does.

### Keep `minecraftctl` platform-agnostic

The tag is AWS glue, and probably not how switching works long term (see D).
`minecraftctl` knows about worlds and systemd, not EC2, and it should stay that
way even though the rest of the project is AWS-specific. So the layers are:

- **`minecraftctl world switch <name>`**: stops the running world and starts
  another. It knows nothing about tags.
- **Platform glue in `packer/`**: the phase 1 boot unit and the B watcher.
  These are the only pieces that read `ActiveWorld`, and both call
  `minecraftctl world switch` to do the switch. If switching later moves to
  `minecraftctl serve` or something else, the glue gets deleted and
  `minecraftctl` doesn't change.

**The watcher reacts to tag changes, not to a mismatch.** It remembers the last
tag value it acted on (in `/run`, so a boot starts fresh) and only switches when
the tag changes. A switch done by hand with `minecraftctl world switch` over
SSH then stays in place until the tag changes or the instance reboots, when the
tag wins again. Nothing on the instance writes tags, and the instance role
needs no `ec2:CreateTags`.

## Phase 2: `minecraftctl world switch <name>`

The CLI on its own, as its own PR. It's useful over SSH straight away, and
phase 3 builds on it.

1. Check that the world can start (see below).
2. Refuse if players are online, unless `--force` is given.
3. Stop every running `minecraft@*` (its `ExecStop` saves the world first).
4. Start `minecraft@<name>` and wait until it's up.
5. If it doesn't come up, stop it and start the previous world again.

If `<name>` is already the only world running, it does nothing and exits 0.
If nothing is running (at boot), it just starts `<name>`. `--dry-run` prints
what it would stop and start.

**One "can start" check, in Go.** `why_not_startable` in
`minecraft-active.sh` moved to `worlds.CheckStartable` (`pkg/worlds/switch.go`):
valid name, world dir exists, `eula=true`, `enable-rcon=true`, `server.jar`
resolves. Its bats tests became Go tests. (Not exposed as `world check`;
`world switch --dry-run` covers it.)

**Autoshutdown guard.** `autoshutdown.sh` powers off straight away if no
`minecraft@*` unit is `running`. During a switch there's a gap: the old world
is `deactivating` while it saves (slow on a 6 GB world like `old`), and a new
world that crash-loops sits in `activating (auto-restart)`. If the timer fires
then, the instance powers off mid-switch. **Decided: an flock.** `switch`
holds `/run/minecraft-switch.lock` (`pkg/lock`) for as long as it runs, and
autoshutdown skips its check while `flock -n` can't take it, as it already
does for SSH sessions. Autoshutdown holds the lock until it exits, so a switch
can't start between its check and its shutdown decision. flock releases the
lock if either dies, and it stops two switches running at once (the second
waits 15 s, then exits 3). Autoshutdown runs as `minecraft` and can't create
files in `/run`, so a tmpfiles.d entry creates the lock file at boot. (A manual swap over SSH was already safe
because of the SSH check.)

**Waiting for "up".** `systemctl start` returning doesn't mean Minecraft is up.
Wait for RCON to answer `list`, with `--timeout` (default about 5 minutes; a
first load of an old world is slow). RCON settings come from
`/etc/minecraft.env` and are shared by every world, so `rcon.NewClient()`
needs nothing per-world. It gives up early if the unit goes `inactive` or
`failed`.

**Finding the running world.** `systemd.ListUnits` lists `minecraft@*.service`
in any state but stopped. Stopping every match also cleans up two worlds
running at once. Only a world that's `active` is checked for players and
rolled back to; one that's crash-looping (`activating`) or stopping isn't. A
target that's crash-looping, or running alongside another world, is stopped
and started fresh (the other world's `ExecStop` sends `stop` to the shared
RCON port, which may be the target's). Only the target running alone and up
is a no-op.

**Exit codes.** The phase 3 watcher needs to tell outcomes apart without
parsing stderr, so give them distinct, documented codes:

| Code | Outcome | Watcher does |
|---|---|---|
| 0 | Switched, or already running it | Marks the tag handled |
| 1 | Any other error | Retries next tick |
| 2 | World can't start; nothing changed | Marks the tag handled (failed) |
| 3 | Refused: players online, player count unreadable, or another switch running; nothing changed | Retries next tick |
| 4 | New world failed, previous world restored (or nothing was running) | Marks the tag handled (failed) |
| 5 | Rollback failed too, nothing running | Marks handled, logs loudly |

An unreadable player count (RCON down on an `active` world) refuses rather
than guessing the server is empty.

**`--force`.** SSH-only (the watcher never passes it). Warns players with a
`say`, waits `--warn-delay` (default 10 s), then stops.

**Decided: the boot unit calls `world switch`.** `minecraft-active.sh` now
reads the tag, then runs `world switch <tag>`, falling back to
`world switch <default>`. So a tagged world that starts but doesn't come up
(corrupt save, crash on load) now falls back to the default too, where
before it would crash-loop. The bats tests cover just the tag and fallback.

**Decided: `world start` and `world restart` refuse** while a different world
is running, and point to `world switch`.

## Phase 3: watcher, lambda and UI

**Player policy.** `world switch` refuses while players are online, but the UI
was going to warn that switching kicks everyone off. Proposed: the watcher
refuses (no `--force`), and the maps app shows the player count and disables
**Play** until the server is empty. Forcing a switch stays an SSH-only action.

**Watcher.** A timer (option B) in `packer/` that acts only on tag changes, as
above, and handles `world switch`'s exit codes as in the phase 2 table. A
refused switch isn't marked handled, so it retries; a failed one is, so it
doesn't retry until the tag changes again.

**Lambda and UI.**

- `/start?world=` on a running instance with a different world: set the tag
  and return 202 instead of 409. The watcher does the switch.
- The maps app enables **Play** on other worlds while the server is running
  and empty (see the player policy above).

## Worlds on prod (checked 2026-09-24)

| World | Jar | `world/` size |
|---|---|---|
| `default` | (current) | 1.5 GB |
| `old` | 1.16.4 | 5.9 GB |
| `world.bak` | 1.19 | 2.7 GB |
| `world.bak2` | 1.19 | 691 MB |
| `world.bak-1.19.2` | 1.19.2 | 1.6 GB |

Every jar is present in `/opt/minecraft/jars/`. Every world has a
`map-config.yml` and a published map, so all five are in
`s3://minecraft-prod-maps/maps/world_manifest.json`.

## Before switching to an old world

- **Server jar:** `server.jar` in each world dir is a symlink into
  `/opt/minecraft/jars/`. If the target jar is missing, the world won't start.
  Never point an old world at a newer jar: Minecraft upgrades the save in place
  and can't downgrade it.
- **Snapshot it first.** The `.bak` worlds are archives. Take a per-world restic
  snapshot before anyone plays in them (see below).
- **Consider renaming.** `world.bak-1.19.2` is a legal systemd instance name,
  but names like `2022-bak2` would read better in the maps app. If we rename,
  do it before registering timers or building maps, since both use the name.

### Snapshotting the archive worlds

`minecraftctl backup create <world>` snapshots `<world>/world/` with tag
`<world>`. Only registered worlds get the daily timer, so the archives have to
be done by hand. The weekly prune (`--keep-daily 7 --keep-weekly 4
--keep-monthly 3`, grouped by host and path) always keeps the newest snapshot
per path, so a lone archive snapshot won't be pruned.

`backup create all` also covers each world's `server.properties`, `ops.json` and
`map-config.yml`.

## Order of work

Phase 1:

1. ~~Snapshot the archive worlds~~: done 2026-09-25 (IDs in the TODO).
2. ~~Check jars and give the old worlds maps~~: already done on prod.
3. Terraform: `instance_metadata_tags = "enabled"`; `ec2:CreateTags` for the
   control lambda; `MAPS_BUCKET` for the control lambda. **Review:**
   `ignore_changes` for `tags["ActiveWorld"]`.
4. Packer: `minecraft-active.service`; `world register` / `world create` stop
   enabling `minecraft@`. Ship as a new AMI. **Review:** the "can start" check
   (`server.properties` plus a resolvable jar, not `level.dat`); register every
   world on a new instance; `map build enable` stops starting the timer.
5. Control lambda: `/start?world=`, `ActiveWorld` in `/status`.
6. Maps app: **Play** button and active-world/server state.
7. Try it on test:
   - pick a second world, confirm it comes up;
   - press Start on the Controls page, confirm the same world comes up again;
   - set the tag to a name that doesn't exist, confirm `default` comes up;
   - point a world's `server.jar` at a missing jar, confirm the fallback;
   - a new instance on an **empty** data volume (world gets generated);
   - a new instance on an **existing** volume (all worlds registered, tagged
     world starts);
   - `terraform plan` on a tagged instance plans no tag change.

Phase 2 (`minecraftctl world switch`):

1. ~~Settle the lock vs marker, the boot unit and `world start` questions~~:
   flock; boot unit uses `switch`; `start` refuses.
2. ~~Move the "can start" check to `pkg/worlds`; add list-units to
   `pkg/systemd`~~: done.
3. ~~`world switch` with the RCON wait, rollback and exit codes~~: done.
4. ~~Autoshutdown guard~~: done.
5. Try it on test (new AMI):
   - `world switch` between two worlds, with and without players online;
   - `--force` with a player online (they see the warning);
   - point a world's `server.jar` at a missing jar and switch to it (exit 2);
   - make a world crash on start and switch to it (rolls back, exit 4);
   - watch `journalctl -t autoshutdown` skip during a slow switch;
   - reboot with the tag set, and with a bad tag (falls back to default).

Phase 3:

1. Settle the player policy.
2. Confirm on test that tag changes reach metadata while the instance runs.
3. The watcher timer (option B) in `packer/`, acting only on tag changes.
4. Lambda returns 202 instead of 409 for a running instance, and the maps app
   allows switching while it runs.

## Open questions

- ~~Should the tag go back to `default` after a session on an old world?~~
  **No, it stays** (sticky). It resets only when the instance is replaced.
- ~~Daily backups for the old worlds?~~ **Yes, every playable world gets the
  daily backup timer.** Restic dedupes, so a world nobody played costs almost
  nothing. This comes from registering every world on a new instance (above).
- ~~Should the Controls page show the selected world?~~ **Yes, as a split
  button** (built 2026-09-26, `StartButton` in `libs/ui`). The status shows
  "World: <active world>"; the main **Start** does a plain `/start` into it.
  The dropdown lists the worlds
  from `/api/worlds`; choosing one starts it straight away via
  `/start?world=`. If the world list can't load, it's a plain Start. It only
  shows while the server is stopped; phase 3 could reuse it as "Switch to…".
