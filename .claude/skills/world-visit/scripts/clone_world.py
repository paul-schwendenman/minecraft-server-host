#!/usr/bin/env python3
"""Make a playable copy of a save for exploring/screenshots, leaving the original untouched.

  clone_world.py <save-name> [--saves DIR] [--suffix " (screenshots)"] [--pack DIR] [--no-cheats]

- refuses if the destination exists or a running game holds files in the source save
- APFS clone (cp -c): instant, no extra disk until files change
- turns on cheats (allowCommands) in the clone's level.dat only
- copies an optional datapack dir into the clone's datapacks/
- prints the game version to launch (opening in a newer client upgrades the save for good)

Never list game processes with `ps`/`pgrep -fl`: the launcher passes the account
access token on the java command line.
"""
import argparse, gzip, os, shutil, subprocess, sys

sys.dont_write_bytecode = True  # no __pycache__ in the skill dir
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', 'world-explore', 'scripts'))
from mcnbt import parse  # noqa: E402

SAVES = os.path.expanduser('~/Library/Application Support/minecraft/saves')
ALLOW = b'\x01\x00\x0dallowCommands'


def open_by_game(path):
    pids = subprocess.run(['pgrep', 'java'], capture_output=True, text=True).stdout.split()
    for pid in pids:
        out = subprocess.run(['lsof', '-p', pid], capture_output=True, text=True).stdout
        if path + '/' in out:
            return pid
    return None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('save')
    ap.add_argument('--saves', default=SAVES)
    ap.add_argument('--suffix', default=' (screenshots)')
    ap.add_argument('--pack', help='datapack directory to install in the clone')
    ap.add_argument('--no-cheats', action='store_true')
    a = ap.parse_args()

    src = os.path.join(a.saves, a.save)
    dst = src + a.suffix
    if not os.path.exists(os.path.join(src, 'level.dat')):
        sys.exit(f'{src}: no level.dat')
    if os.path.exists(dst):
        sys.exit(f'{dst} already exists')
    pid = open_by_game(src)
    if pid:
        sys.exit(f'game process {pid} has files open in {src}; quit to the title screen first')

    subprocess.run(['cp', '-cR', src, dst], check=True)

    lvl = os.path.join(dst, 'level.dat')
    raw = gzip.decompress(open(lvl, 'rb').read())
    data = parse(raw).get('Data', {})
    version = data.get('Version', {}).get('Name', '?')
    if not a.no_cheats:
        if raw.count(ALLOW) != 1:
            print('warning: allowCommands not found exactly once; cheats left as-is (use Open to LAN)')
        else:
            i = raw.find(ALLOW) + len(ALLOW)
            raw = raw[:i] + b'\x01' + raw[i + 1:]
            open(lvl, 'wb').write(gzip.compress(raw))

    if a.pack:
        os.makedirs(os.path.join(dst, 'datapacks'), exist_ok=True)
        shutil.copytree(a.pack, os.path.join(dst, 'datapacks', os.path.basename(os.path.normpath(a.pack))))

    print(f'cloned {a.save!r} -> {os.path.basename(dst)!r}')
    print(f'level name in game list: {data.get("LevelName")!r} (folder name shows underneath)')
    print(f'cheats: {"on" if not a.no_cheats else "unchanged"}; pack: {a.pack or "none"}')
    print(f'launch Minecraft {version} (a newer client upgrades the save permanently)')


if __name__ == '__main__':
    main()
