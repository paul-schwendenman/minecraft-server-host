# Plan: Switch Which World the Server Runs

Status: **phase 1 implemented** (2026-09-25), not yet deployed or tried on test.
The plan was reviewed on 2026-09-25; the phase 1 changes it asked for (marked
**Review** below) are in the code too. Phase 2 is proposed. Written 2026-09-24.

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
`minecraft@` services happen to be enabled. Build it in two phases:

- **Phase 1: choose the world at start.** Pick a world in the UI. The control
  lambda sets the tag and starts the instance, and a boot unit on the instance
  starts that world. Switching while the server is running means stopping it
  first.
- **Phase 2: switch while the server is running**, without a restart.

The controls (the **Controls** page of `minecraft-ui/apps/worlds`, the
`ServerStatus` component, and the legacy `apps/manager`) don't change in either
phase. They keep their plain Start/Stop, which boots whatever world the tag
already says. World selection goes on the world pages of the same app, which
already list every world.

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

- `server.properties`, and
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

## Phase 2: switch while the server is running

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

### `minecraftctl world switch <name>`

1. Check the world exists and that its `server.jar` target is present.
2. Refuse if players are online, unless `--force` is given.
3. Stop the running `minecraft@*` (its `ExecStop` saves the world first).
4. Start `minecraft@<name>`.

It could also be used in phase 1: the boot unit would call it instead of
`systemctl start` directly. The player check never fires at boot, since nobody
is online yet.

### Review: decide before building phase 2

- **Player policy.** `world switch` refuses while players are online, but the
  UI was going to warn that switching kicks everyone off. Pick one. Proposed:
  the watcher refuses while players are online (no `--force`), and the maps app
  shows the player count and disables **Play** until the server is empty.
  Forcing a switch stays an SSH-only action.
- **Watcher retries.** A switch refused because players are online is logged
  and retried on the next tick; the tag value isn't marked as handled. A
  successful switch marks it as handled. A startup failure triggers rollback
  and is marked as handled (failed), with no further retries until the tag
  changes (see next point).
- **Failed startup.** `systemctl start` returning doesn't mean Minecraft is up.
  `world switch` should wait for RCON to answer (a few minutes' timeout). If
  the new world doesn't come up, stop it, start the previous world again, and
  report the failure. The watcher then records that tag value as handled
  (failed), so it doesn't retry until the tag changes again.

### Lambda and UI changes in phase 2

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

Phase 2:

0. Settle the player policy, watcher retries and failed-startup handling
   (above).
1. Confirm on test that tag changes reach metadata while the instance runs.
2. `minecraftctl world switch` (no tag handling; see above).
3. The watcher timer (option B) in `packer/`, acting only on tag changes.
4. Lambda returns 202 instead of 409 for a running instance, and the maps app
   allows switching while it runs.

## Open questions

- ~~Should the tag go back to `default` after a session on an old world?~~
  **No, it stays** (sticky). It resets only when the instance is replaced.
- ~~Daily backups for the old worlds?~~ **Yes, every playable world gets the
  daily backup timer.** Restic dedupes, so a world nobody played costs almost
  nothing. This comes from registering every world on a new instance (above).
- Should the Controls page show the selected world, so it's clear what its
  plain **Start** launches (e.g. "Start (default)")? The review suggests it;
  it goes against keeping the controls unchanged. Your call.
