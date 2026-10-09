# Slow starts: backups running at boot

Found 2026-10-09 on prod (`i-01f026b9a32978903`, 2 vCPU / 4 GB, AMI `minecraft-ubuntu-1790698722`). Nothing changed yet; this records the evidence and the options in case the journal is lost.

## Symptom

Starts felt slow. From boot to `INF world is up`, prod took about 36s on a clean boot, but 60–79s on most boots, and the slow time has been creeping up.

## Evidence

Per-boot numbers from `journalctl --list-boots`: time from the first journal line to `minecraft-active.sh: INF world is up`, and how many backup jobs ran before that point.

| Boot (UTC) | Ready | World backups before ready | Map backups before ready |
|---|---|---|---|
| 2026-09-29 17:20 | 62s | 0 | 0 |
| 2026-09-30 01:39 | **36s** | 0 | 0 |
| 2026-10-01 04:34 | 62s | 5 | 1 |
| 2026-10-02 01:32 | 60s | 5 | 2 |
| 2026-10-03 01:08 | 62s | 5 | 2 |
| 2026-10-03 18:31 | 59s | 0 | 3 |
| 2026-10-04 01:25 | 64s | 5 | 2 |
| 2026-10-05 01:05 | 65s | 5 | 2 |
| 2026-10-05 01:46 | **37s** | 0 | 0 |
| 2026-10-06 04:00 | 64s | 5 | 2 |
| 2026-10-07 03:38 | 69s | 5 | 2 |
| 2026-10-08 03:17 | 64s | 5 | 2 |
| 2026-10-09 01:41 | 79s | 5 | 2 |

- The two fast boots are the only ones with no backups running. The cleanest comparison is 2026-10-05: 65s with backups, then 37s on a second start 15 minutes later when nothing was left to catch up.
- The extra time is spent before Java prints its first line. From `INF starting world` to `Starting minecraft server version`, clean boots took about 21s and contended boots took 41–56s. Minecraft's own `Done (Xs)` also rose, from about 1.0s to 2.2–4.4s.
- The 2026-09-29 62s boot had no backups. It was the first start after the AMI rebuild, so the volume was being read cold (EBS lazy-loads from the snapshot). It is a different cause.
- The 2026-10-03 18:31 boot had no world backups (they had already run that day) but still had 3 map backups, and still took 59s. Map backups alone are enough to slow the start.
- `minecraft-map-build@default` is not involved. It starts at `OnBootSec=1min`, after the world is mostly up.

## Cause

Both backup timers use `Persistent=true`, and the instance is almost always off at their scheduled times:

- `packer/scripts/minecraft/worlds/minecraft-world-backup@.timer`: `OnCalendar=daily` (00:00 UTC), one per world. Prod has 5 worlds: `default`, `old`, `world.bak`, `world.bak2`, `world.bak-1.19.2`.
- `packer/scripts/minecraft/maps/minecraft-map-backup@.timer`: `OnCalendar=*-*-* 00,12:00:00`.
- `minecraft-world-prune.timer` (Sunday 04:00) also catches up at boot, so expect Sunday-after starts to be worse.

So on the first start of each day, systemd catches up every missed timer at once: 5 restic world backups and the map backups, all in parallel with the JVM starting on 2 vCPUs. Growing backups likely explain the upward creep (60s → 79s).

Most of that boot-time work is redundant:

- `minecraft@.service` already backs up the running world on stop (`ExecStopPost` in `worlds/minecraft-override-world-backup.conf`). The daily catch-up for the active world repeats the backup from the last shutdown.
- The archive worlds aren't running and don't change, so their backups find nothing new. restic still has to scan them.

## Options

Not chosen yet. They can be combined.

1. **Delay the catch-up.** Add `RandomizedDelaySec=` (for example 10min) to the backup timers. With `Persistent=true` the catch-up still happens, just spread out after the world is up. This is a one-line change per timer. It works because systemd applies the random delay to catch-up runs too.
2. **Lower backup priority.** Add `Nice=10`, `CPUWeight=20`, `IOWeight=20` (or `IOSchedulingClass=idle`) to `minecraft-world-backup@.service` and `minecraft-map-backup@.service`. When they overlap with Minecraft, Minecraft wins. This also helps with the backups that run while people are playing.
3. **Order after the world is up.** Add `After=minecraft-active.service` to the backup services. `minecraft-active.service` stays `activating` until `INF world is up`, so the backups would wait for it. This is the most targeted fix, but the backups would then depend on world switching.
4. **Stop backing up what doesn't change.** Disable the daily timers for the archive worlds, and rely on the `ExecStopPost` backup for the active world. Maybe keep a less frequent safety net. This removes most of the work instead of moving it. Check it against the **Scheduled `backup create all`** todo item first.
5. **Bigger instance.** Doesn't fix the cause; listed only for completeness.

A reasonable first step is 1 + 2: small, low-risk, and easy to verify with the same measurement below.

## Measuring it again

On the instance:

```bash
for b in $(seq -14 0); do
  t0=$(journalctl -b $b -o short-unix --no-pager | head -1 | cut -d. -f1)
  up=$(journalctl -b $b -o short-unix --no-pager | grep -m1 "INF world is up" | cut -d. -f1)
  [ -n "$up" ] || continue
  bk=$(journalctl -b $b -o short-unix --no-pager | awk -v u=$up '$1+0<=u' | grep -c "Backing up world:")
  mb=$(journalctl -b $b -o short-unix --no-pager | awk -v u=$up '$1+0<=u' | grep -c "backup-maps.sh.*Backup complete")
  echo "boot $b ready=$((up-t0))s world-backups=$bk map-backups=$mb"
done
```

Time spent before the instance's kernel starts (EC2 provisioning) isn't in the journal, so the UI's real wait is a little longer than "ready".
