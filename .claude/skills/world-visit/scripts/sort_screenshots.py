#!/usr/bin/env python3
"""Copy screenshots into per-world folders, working out the world from the game logs.

  sort_screenshots.py --since 2026-09-30 --map 1.16.4=old --map 1.19=world.bak [--dest DIR] [--dry-run]

Logs never name the save folder (every world calls itself "world"), so a screenshot is
assigned by the version of the integrated server running when it was taken. That only
works when each version maps to one save. Multiplayer sessions ("Connecting to <host>")
are listed separately for a manual call. Existing files are skipped; originals stay put.
"""
import argparse, collections, glob, gzip, os, re, shutil

MC = os.path.expanduser('~/Library/Application Support/minecraft')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--since', required=True, help='YYYY-MM-DD; earlier logs are ignored')
    ap.add_argument('--map', action='append', required=True, metavar='VERSION=WORLD')
    ap.add_argument('--dest', default=os.path.expanduser('~/Pictures/Minecraft worlds'))
    ap.add_argument('--mc', default=MC)
    ap.add_argument('--dry-run', action='store_true')
    a = ap.parse_args()
    worlds = dict(m.split('=', 1) for m in a.map)

    logs = sorted(f for f in glob.glob(f'{a.mc}/logs/*.log.gz') if os.path.basename(f) >= a.since)
    logs.append(f'{a.mc}/logs/latest.log')
    cur, shots, remote = None, {}, collections.defaultdict(list)
    for f in logs:
        with (gzip.open(f, 'rt', errors='replace') if f.endswith('.gz') else open(f, errors='replace')) as fh:
            for line in fh:
                if m := re.search(r'Starting integrated minecraft server version (\S+)', line):
                    cur = ('local', m.group(1))
                elif m := re.search(r'Connecting to ([^,\s]+)', line):
                    cur = ('remote', m.group(1))
                elif 'Stopping server' in line:
                    cur = None
                elif m := re.search(r'Saved screenshot as (\S+\.png)', line):
                    if cur and cur[0] == 'local' and cur[1] in worlds:
                        shots[m.group(1)] = worlds[cur[1]]
                    elif cur and cur[0] == 'remote':
                        remote[cur[1]].append(m.group(1))
                    else:
                        remote[f'unmatched ({cur[1] if cur else "no session"})'].append(m.group(1))

    done = collections.Counter()
    for name, w in sorted(shots.items()):
        src, dst = f'{a.mc}/screenshots/{name}', f'{a.dest}/{w}/{name}'
        if not os.path.exists(src):
            done[w, 'gone'] += 1
        elif os.path.exists(dst):
            done[w, 'already there'] += 1
        else:
            if not a.dry_run:
                os.makedirs(os.path.dirname(dst), exist_ok=True)
                shutil.copy2(src, dst)
            done[w, 'would copy' if a.dry_run else 'copied'] += 1
    for (w, what), n in sorted(done.items()):
        print(f'{w}: {n} {what}')
    for where, names in remote.items():
        print(f'not assigned, {where}: {len(names)} ({names[0]} .. {names[-1]})')
    print(a.dest)


if __name__ == '__main__':
    main()
