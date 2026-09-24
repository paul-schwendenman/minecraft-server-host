# Plan: Shut Down After Background Jobs Finish

Status: **proposed**, not implemented. Came out of moving `old` to prod on
2026-09-24 ([moving-worlds.md](../moving-worlds.md)).

## Problem

We wanted to start a long map build (~2 h) and a backup, disconnect, and have
the instance power off once they finished. There's no supported way to do that:

1. **Autoshutdown doesn't know about jobs.** `autoshutdown.sh` looks at SSH
   sessions, running `minecraft@` services and the RCON player count. A map build,
   world backup or map upload in progress doesn't count, so after you disconnect it
   powers off mid-job: at the first check if no world is running, otherwise at the
   second idle check.
2. **Nothing chains build → upload.** A finished build only reaches S3 when
   `minecraft-map-backup@<world>.timer` next fires (every 12 h), or if you run it
   by hand.
3. **Progress is hard to see.** You have to piece it together from journal
   lines, `ps` and `du`. Restic prints nothing until it's done.

The workaround was to stop `autoshutdown.timer` and run a hand-written waiter
under `systemd-run`. The first version checked with `systemctl is-active`, which
treats a oneshot's `activating` state as not running, so it powered off 3 minutes
in.

## Proposal

### 1. Make autoshutdown job-aware (the core fix)

In `autoshutdown.sh`, before both power-off paths (no services running, and
second idle check), skip shutdown if any job is running:

```bash
BUSY=$(systemctl list-units --type=service --state=activating,active --no-legend \
  'minecraft-map-build@*' 'minecraft-world-backup@*' 'minecraft-map-backup@*' \
  'minecraft-world-backup.service' 'minecraft-map-backup.service' 2>/dev/null | wc -l)
if [[ "${BUSY}" -gt 0 ]]; then
  rm -f "${TOUCH_FILE}"   # restart the idle count once jobs finish
  logger -t autoshutdown "Skipping shutdown: ${BUSY} background job(s) running"
  exit 0
fi
```

With this, "start the job and disconnect" works with no waiter. Once the jobs
finish, the normal rules apply: power off at the next check if no world is running,
otherwise after two idle checks. Also add a bats case in `packer/tests/autoshutdown.bats`.

Also consider a generic hold: `/run/autoshutdown/hold` (tmpfs, so it's gone
after a reboot). Any ad-hoc job can create it, and autoshutdown skips while it
exists. It's simpler than a unit list for one-off work like the S3 copy, but it's
easy to leave behind, so only add it if we also add the max-uptime backstop.

### 2. Upload after a build

Options:

- **a.** `OnSuccess=minecraft-map-backup@%i.service` on `minecraft-map-build@.service`.
  That uploads after every build (every 15 min for registered worlds). Builds skip
  when the world hasn't changed, and `aws s3 sync` only sends changed tiles, but it
  still means an S3 listing every 15 minutes per world.
- **b.** `minecraftctl map build now <world> --upload`, which runs the upload step
  after a successful manual build. Scheduled builds stay as they are.

Leaning **b**: it's opt-in and matches what we did by hand.

### 3. Build progress

Have the map builder (`minecraftctl/pkg/maps/build.go`) write a progress file,
e.g. `/run/minecraftctl/map-build-<world>.json`, with the current map, range
index (`3/7`), step start times and elapsed time per finished step. Then
`minecraftctl map build status <world>` can show it, along with anything waiting
on the lock (tonight `default`'s scheduled build sat waiting for ~2 h behind
`old`).

### Not proposed: a `minecraftctl shutdown --after` command

A formal waiter command (wait for units, run a step, power off, with a cap) would
work, but it duplicates what autoshutdown should already do, and every user has to
remember to use it. With items 1–2, the flow becomes:

```bash
sudo systemctl start --no-block minecraft-world-backup@old
minecraftctl map build now old --upload &   # or via the service
exit                                        # autoshutdown handles the rest
```

## Interactions

- **Max-uptime backstop** ([autoshutdown-review.md](autoshutdown-review.md#deferred-hard-backstop-not-implemented)):
  once jobs can hold the instance up, the backstop matters more. It has to be
  longer than the longest expected job (the first `old` render plus upload took ~1 h 50 min), so
  12 h is still fine.
- **Stuck jobs:** a hung uNmINeD would keep the instance up until the backstop
  fires. Consider `RuntimeMaxSec=` on the build and backup units (e.g. 6 h).

## Related gaps found the same night

These fit in the TODO list rather than this plan:

- Nothing runs `minecraftctl backup create all` on a schedule, so a world's
  config files (`server.properties`, `ops.json`, `map-config.yml`) are only
  backed up when someone does it by hand.
- `backup create all` always exits 3 because the minecraft user can't read
  `caddy/`. Exclude `caddy/` from `all` backups so a clean run exits 0.
