# Moving a World Between Servers

Runbook for copying a world from one environment to another (e.g. test → prod),
giving it a map, and getting it into the destination's backups. Written from
moving the original 1.16.4 world (`old`) from test to prod on 2026-09-24.

Hosts are reached with `ssh -i ~/.ssh/minecraft-packer.pem ubuntu@<ip>`. Public
IPs change on every start; look them up with:

```bash
AWS_PROFILE=minecraft AWS_REGION=us-east-2 aws ec2 describe-instances \
  --query 'Reservations[].Instances[].[Tags[?Key==`Name`]|[0].Value,State.Name,PublicIpAddress]' \
  --output text
```

## Before you start: keep both instances up

`autoshutdown.sh` runs every 5 minutes and skips shutdown only while someone has
an interactive SSH session (a `pts/` entry in `who`). **If no `minecraft@`
service is running, it powers off at the first check** — it does not know about
copies, map builds or backups in progress.

- Stay SSH'd in to both hosts for the whole copy, or
- `sudo systemctl stop autoshutdown.timer` on the host. `stop` (not `disable`)
  lasts until the next boot, so the timer comes back on its own.

## 1. Check the destination

```bash
minecraftctl world list          # on both hosts
df -h /srv/minecraft-server       # room for the world on the destination?
id minecraft                      # uid must match on both (pinned to 996)
ls /opt/minecraft/jars/           # destination has the world's server jar?
```

The world dir's `server.jar` is a symlink into `/opt/minecraft/jars/`, so the
destination needs `minecraft_server_<version>.jar` for that world to start.

## 2. Copy via S3

Test and prod each have their own backup bucket and restic password, so a
snapshot from one can't be restored on the other. The EC2 roles can only reach
their own env's buckets, so go through the **source** bucket and hand the
destination a presigned URL. Traffic stays inside AWS (5.9 GB took a few
minutes each way).

On the source host (world not running):

```bash
sudo tar -C /srv/minecraft-server -cpf - <world> | zstd -3 -T0 \
  | aws s3 cp - s3://minecraft-test-backups/transfer/<world>.tar.zst --expected-size 7000000000
```

From your laptop, presign it and extract on the destination:

```bash
URL=$(AWS_PROFILE=minecraft aws s3 presign s3://minecraft-test-backups/transfer/<world>.tar.zst --expires-in 3600)
ssh ... ubuntu@<dest> "test ! -e /srv/minecraft-server/<world> && \
  curl -fsS '$URL' | zstd -d | sudo tar -C /srv/minecraft-server -xpf -"
```

`tar -p` keeps ownership, which is fine as long as the uids match (step 1).
Then check it and delete the transfer object:

```bash
sudo du -sh /srv/minecraft-server/<world>
sudo find /srv/minecraft-server/<world> ! -user minecraft | head   # expect nothing
minecraftctl world list
aws s3 rm s3://minecraft-test-backups/transfer/<world>.tar.zst
```

**Why not attach the source EBS volume to the destination?** It means stopping
the source, both instances being in the same AZ, and both volumes probably having
the same filesystem UUID (built from the same image), which XFS won't mount
twice without `-o nouuid`. Then everything has to be put back. S3 is simpler.

## 3. Back it up on the destination

Two different backups, and you want both:

| Command | Covers |
|---|---|
| `minecraftctl backup create <world>` | only `<world>/world/` (tag `<world>`) |
| `minecraftctl backup create all` | all of `/srv/minecraft-server` except logs and crash reports (tag `all`) |

Only the per-world backup has a timer, and only for registered worlds. **A world's
`server.properties`, `ops.json`, `whitelist.json` and `map-config.yml` are only in
`all` snapshots**, which nothing runs on a schedule. After a move, run both:

```bash
sudo systemctl start --no-block minecraft-world-backup@<world>.service
sudo systemd-run --unit=backup-all-manual --uid=minecraft --gid=minecraft \
  -p EnvironmentFile=/etc/minecraft.env -p Environment=HOME=/home/minecraft \
  /usr/local/bin/minecraftctl backup create all
```

Confirm the snapshot saved and reached S3:

```bash
sudo -u minecraft minecraftctl backup list <world>
aws s3 ls s3://minecraft-prod-backups/snapshots/      # from the laptop
```

Expect `backup create all` to end with **exit status 3** and the unit marked
failed: `/srv/minecraft-server/caddy/` belongs to caddy and the minecraft user
can't read it. Restic exits 3 when it saved the snapshot but skipped unreadable
files; look for `snapshot <id> saved` in the journal. Caddy re-issues its certs,
so losing that directory isn't a problem.

Once both snapshots are in S3 the destination owns the world, and the source
copy can be deleted.

## 4. Give it a map

A world with no `map-config.yml` has no map. Size the world first; the bounding
box is misleading because elytra trails generate lots of near-empty regions:

```bash
cd /srv/minecraft-server/<world>/world/region
sudo ls | wc -l                                   # total regions
sudo find . -name '*.mca' -size +4M | wc -l      # well-explored regions
sudo find . -name '*.mca' -size -100k | wc -l    # thin trail regions
```

`old` had 3,339 regions spanning ~68k × 66k blocks, but only ~700 over 4 MB, so
the real work was ~3–4× `default` (289 regions), not 11×. uNmINeD only renders
chunks that exist.

Settings that worked for a large world (see [map-build-config.rst](map-build-config.rst)):

- `zoomout: 6` — a ~68k-block world fits on screen fully zoomed out.
- `zoomin: 1` for the whole map. Each extra level roughly quadruples output, so
  spend extra zoom only on `ranges`.
- `ranges` at radius ~1024 with `zoomin: 2` over spawn and the densest spots.
  To find candidates, cluster the region files over 6 MB. Range bounds are snapped
  out to region boundaries, so they come out a bit bigger than the radius.

Kick off a one-off build without enabling the timer:

```bash
sudo systemctl start --no-block minecraft-map-build@<world>.service
```

What to expect:

- `ExecStartPre` RCON errors ("connection refused") are normal when the world
  isn't running; those steps are allowed to fail.
- Map builds share one lock. **A long build blocks every other world's scheduled
  map build**, including `default`, until it finishes.
- Builds are incremental and skipped when the world's mtime hasn't changed, so a
  first render is a one-time cost. An interrupted build picks up where it stopped.
- Rough speed on a t3.medium: ~5 min per range. `old` took an estimated ~2–2.5 h
  in total; the full overworld pass is most of it.
- Rendering doesn't upload anything. The map reaches S3 (and the site) only when
  `minecraft-map-backup@<world>.service` runs.

## 5. Tracking progress

```bash
# State of the jobs; oneshot services show "activating" while running
systemctl list-units 'minecraft-*@<world>*' --all

# Which map/range it's on (one "rendering range" / "rendering base map" line per step)
sudo journalctl -u minecraft-map-build@<world> -b -o short-iso | grep -E 'rendering (range|base map)'

# The uNmINeD process currently running, and tile output so far
ps -o etime,pcpu,args -C unmined-cli
sudo du -sh /srv/minecraft-server/maps/<world>/*

# Snapshot result (restic doesn't log progress while running)
sudo journalctl -u minecraft-world-backup@<world> | tail
```

## 6. Shutting down when the work is done

Until there's a built-in option (see
[plans/shutdown-after-jobs-plan.md](plans/shutdown-after-jobs-plan.md)), stop
the autoshutdown timer and run a one-off waiter:

```bash
sudo systemctl stop autoshutdown.timer
sudo systemd-run --unit=old-migration-finish /bin/bash -c '
running() { case $(systemctl show -p ActiveState --value $1) in activating|active|reloading|deactivating) return 0;; *) return 1;; esac; }
deadline=$(($(date +%s)+14400))                     # 4h hard cap
while running minecraft-world-backup@<world>.service || running minecraft-map-build@<world>.service; do
  [ $(date +%s) -ge $deadline ] && { logger -t old-migration "cap hit"; systemctl poweroff; exit 0; }
  sleep 60
done
systemctl start minecraft-map-backup@<world>.service  # upload the map
logger -t old-migration "done, powering off"
systemctl poweroff'
```

Watch it with `systemctl status old-migration-finish` and `journalctl -t old-migration -f`.
Cancel with `sudo systemctl stop old-migration-finish && sudo systemctl start autoshutdown.timer`.

> **Pitfall:** don't use `systemctl is-active` to test whether a oneshot job is
> still running. While it runs, the unit is `activating`, and `is-active` counts
> only `active`, so it reports the job as finished straight away. The first waiter
> on 2026-09-24 did this: it uploaded a near-empty map and powered off 3 minutes
> in, killing the backup and the build. Check `ActiveState` as above.

## 7. Afterwards

- `minecraftctl world register <world>` to make it a regular world (service plus
  map-build, world-backup and map-backup timers). It uses the same ports as the
  other worlds (25565, RCON 25575), so only one world can run at a time.
- Delete the source copy once the destination's snapshots are confirmed.
