# Plan: Switch Which World the Server Runs

Status: **proposed**, not implemented. Written 2026-09-24.

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

- **Phase 1: choose the world at boot.** Set the tag, start the server, and
  that world comes up. Switching means changing the tag and stopping and
  starting the server.
- **Phase 2: switch while the server is running**, without a restart.

The controls app (`minecraft-ui/apps/manager`) doesn't change in either phase.
It keeps its plain Start/Stop.

## Phase 1: choose the world at boot

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

A tag missing or empty means `default`.

### 2. Boot unit that starts only the active world

New `minecraft-active.service` (oneshot, `RemainAfterExit=yes`,
`WantedBy=multi-user.target`):

1. Read `ActiveWorld` from IMDSv2
   (`/latest/meta-data/tags/instance/ActiveWorld`), falling back to `default`.
2. Check that `/srv/minecraft-server/<world>/world/level.dat` exists. If it
   doesn't, log it and fall back to `default`, so a typo can't leave the server
   up with no world running (autoshutdown would power it off at the next check).
3. `systemctl start minecraft@<world>.service`.

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
- No migration is needed. This ships as a new AMI, and changing `ami` replaces
  the instance, so the root volume (where `systemctl enable` symlinks live)
  starts with no world enabled. The one thing that re-enables a world is
  `user_data`: every new instance runs `create-world.sh <world_name>`, which
  calls `minecraftctl world register`. So the `register` change has to ship in
  the same AMI as `minecraft-active.service`.

Nothing else needs to change. `autoshutdown.sh` and `mc-healthcheck.sh` already
look at any `minecraft@*` unit.

### 3. Switching in phase 1

From a laptop, with the server stopped:

```bash
AWS_PROFILE=minecraft AWS_REGION=us-east-2 aws ec2 create-tags \
  --resources <instance-id> --tags Key=ActiveWorld,Value=world.bak2
```

Then press Start in the controls app as usual. To go back, set the tag to
`default` (or delete it).

If the server is already running, either stop it and start it again, or SSH in
and swap the service by hand. That lasts until the next boot, when the tag wins
again:

```bash
sudo systemctl stop minecraft@default
sudo systemctl start minecraft@world.bak2
```

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

### Setting the tag without the AWS CLI

Once B works, the only thing a web client needs to do is write the tag:

- Control lambda: `POST /world` with `{"world": "<name>"}`. Validate the name
  against `world_manifest.json`, then `CreateTags` (IAM in
  `infra/modules/api_lambda`: `ec2:CreateTags` on the instance ARN, `aws:TagKeys`
  limited to `ActiveWorld`). `/status` returns the current `ActiveWorld`.
- Where the button lives is still open. The controls app stays unchanged. The
  maps app (`apps/worlds`) already lists every world, since
  `world_manifest.json` holds every world with a published map, so a "play this
  world" action there is the obvious candidate. It would be the first write
  action in that app.

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
  but names like `2022-bak2` would read better in the picker. If we rename,
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
3. Terraform: `instance_metadata_tags = "enabled"`.
4. Packer: `minecraft-active.service`; `world register` / `world create` stop
   enabling `minecraft@`.
5. Try it on test: tag a second world, start, confirm it comes up; delete the
   tag, confirm `default` comes up.

Phase 2:

1. Confirm on test that tag changes reach metadata while the instance runs.
2. `minecraftctl world switch` (no tag handling; see above).
3. The watcher timer (option B) in `packer/`, acting only on tag changes.
4. `POST /world` on the control lambda, and decide where the button lives.

## Open questions

- Should the tag go back to `default` after a session on an old world, or stay
  where it was left? (Proposed: stay. In phase 1 that means remembering to set
  it back.)
- Should the old worlds get the daily backup timer once they're playable, or
  only manual snapshots after each session? Registering them adds a daily job
  per world that mostly backs up nothing new; restic dedupes, so it's cheap.
