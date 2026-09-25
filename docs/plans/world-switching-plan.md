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

Choose the world when the server starts, and store the choice in one place
instead of in whichever services happen to be enabled.

### 1. Active world stored in an EC2 tag

The instance carries a tag `ActiveWorld=<name>`. Reasons to use a tag:

- The lambda can set it while the instance is stopped, before calling
  `StartInstances`. A file on the data volume can't be written then.
- The instance can read it at boot from instance metadata, with no API call and
  no extra IAM permissions, as long as tags are exposed to metadata.

Terraform (`infra/modules/mc_stack`, `aws_instance.minecraft`):

```hcl
metadata_options {
  http_tokens            = "required"
  instance_metadata_tags = "enabled"
}
```

Check that changing `metadata_options` updates the instance in place rather than
replacing it.

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

Worlds are no longer enabled one by one:

- `minecraftctl world register` / `world create` stop enabling
  `minecraft@<world>`. They still enable the map-build, world-backup and
  map-backup timers.
- Migration: on the next boot of each env, `systemctl disable minecraft@default`
  (and any other enabled world). This could go in a packer provisioning step or
  in `minecraft-active.service` itself: disable every other `minecraft@*` unit
  before starting the active one. The second option is self-healing, so it's
  preferred.

Nothing else needs to change. `autoshutdown.sh` and `mc-healthcheck.sh` already
look at any `minecraft@*` unit.

### 3. `minecraftctl world switch <name>`

For when you're already SSH'd in:

1. Check the world exists and that its `server.jar` target is present.
2. Stop the running `minecraft@*` (its `ExecStop` saves the world first).
3. Start `minecraft@<name>`.
4. Update the tag so the next boot runs the same world. The instance role needs
   `ec2:CreateTags` on its own instance (condition on `ec2:ResourceTag` or the
   instance ARN). Or skip this step and let the tag win on the next boot. Decide
   when implementing.

Refuse if players are online unless `--force` is given.

### 4. Control lambda: `/start?world=<name>`

- Optional `world` query param. If given, validate it against the world list
  (see 5), `CreateTags` on the instance, then `StartInstances`.
- If the instance is already running and a different world is asked for, return
  409 for now. Switching a running server from the web can come later (it would
  need the `minecraftctl serve` API or SSM Run Command).
- `/status` returns the current `ActiveWorld` so the UI can show it.
- IAM (`infra/modules/api_lambda`): add `ec2:CreateTags` scoped to the instance
  ARN, with `aws:TagKeys` limited to `ActiveWorld`.

### 5. UI: world picker next to Start

The worlds lambda already serves `world_manifest.json`, but that only lists
worlds **with a published map**. Options:

- Only offer worlds that have maps. It's simple, and it pushes us to give the
  old worlds maps anyway.
- Or have the world backup/map jobs also write a list of all worlds to S3.

Start with the first option. `old` has a map; the `.bak` worlds need
`map-config.yml` and a first render.

Show which world is running on the status card, and preselect the current
`ActiveWorld` in the picker.

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

1. Snapshot the archive worlds (manual, now).
2. Check jars and give the old worlds maps.
3. Terraform: `instance_metadata_tags`, lambda `CreateTags`.
4. Packer: `minecraft-active.service`; `world register` stops enabling
   `minecraft@`.
5. Control lambda `/start?world=` and `ActiveWorld` in `/status`.
6. UI picker.
7. `minecraftctl world switch`.

## Open questions

- Should the tag go back to `default` after a session on an old world, or stay
  where it was left? (Proposed: stay; the UI shows it.)
- Should the old worlds get the daily backup timer once they're playable, or
  only manual snapshots after each session? Registering them adds a daily job
  per world that mostly backs up nothing new; restic dedupes, so it's cheap.
