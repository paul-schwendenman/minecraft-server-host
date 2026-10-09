#!/usr/bin/env python3
"""Find things in a Minecraft Java save without opening the game.

Read-only. Works on 1.16-1.20.4 saves (pre-1.20.5 item format; chunk formats from 1.16 on).

  scan.py halls     <world>...   villager job-site clusters (trading halls, crop farms, barrel rooms)
  scan.py portals   <world>...   nether portals (overworld + nether), bed clusters, player spawn/last position
  scan.py builds    <world>...   player-built areas scored by hoppers/furnaces/shulkers/signs, with sign text
  scan.py mapconfig <world>...   suggested map-config.yml ranges around builds, halls and nether portal hubs
  scan.py players   <world>...   play time, distance, blocks mined, deaths per player
  scan.py inventory <world>... --player NAME|UUID   inventory, armor, ender chest, shulker contents
  scan.py villagers <world>... [--near=X,Z]        villager food/inventory, breeding readiness, free beds

<world> is a save directory (the one holding level.dat). Player names come from a
usercache.json next to the save (server layout), the local client's, or --usercache.
"""
import argparse, collections, glob, gzip, json, os, struct, sys, zlib
from multiprocessing import Pool

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from mcnbt import parse, region_chunks  # noqa: E402

CLIENT_USERCACHE = os.path.expanduser('~/Library/Application Support/minecraft/usercache.json')


# ---------- helpers

def load_names(worlds, extra):
    names = {}
    paths = [extra] if extra else []
    for w in worlds:
        paths += [os.path.join(w, '..', 'usercache.json'), os.path.join(w, 'usercache.json')]
    paths.append(CLIENT_USERCACHE)
    for p in paths:
        try:
            for e in json.load(open(p)):
                names.setdefault(e['uuid'], e['name'])
        except Exception:
            pass
    return names


def who(names, uuid):
    return names.get(uuid, uuid[:8])


def pois(dim_dir, keep):
    out = []
    for f in glob.glob(os.path.join(dim_dir, 'poi', '*.mca')):
        for c in region_chunks(f):
            for sec in c.get('Sections', {}).values():
                for r in sec.get('Records', []):
                    if keep(r['type']):
                        out.append((tuple(r['pos']), r['type'].split(':')[-1]))
    return out


def cluster(items, grid, pos=lambda i: i[0]):
    """Group items whose x/z grid cells touch."""
    cells = collections.defaultdict(list)
    for it in items:
        p = pos(it)
        cells[(p[0] // grid, p[2] // grid)].append(it)
    seen, out = set(), []
    for k in cells:
        if k in seen:
            continue
        stack, members = [k], []
        seen.add(k)
        while stack:
            a = stack.pop()
            members += cells[a]
            for dx in (-1, 0, 1):
                for dz in (-1, 0, 1):
                    n = (a[0] + dx, a[1] + dz)
                    if n in cells and n not in seen:
                        seen.add(n)
                        stack.append(n)
        out.append(members)
    return sorted(out, key=len, reverse=True)


def centre(ps):
    xs, ys, zs = (sorted(p[i] for p in ps) for i in range(3))
    m = len(ps) // 2
    return xs[m], ys[m], zs[m]


def player_files(w):
    for f in glob.glob(os.path.join(w, 'playerdata', '*.dat')):
        try:
            yield os.path.basename(f)[:-4], parse(gzip.decompress(open(f, 'rb').read()))
        except Exception:
            pass


# ---------- halls

POI_SKIP = {'home', 'nether_portal', 'bee_nest', 'beehive', 'meeting', 'lodestone', 'lightning_rod'}


def cmd_halls(a, names):
    for w in a.worlds:
        pts = pois(w, lambda t: t.split(':')[-1] not in POI_SKIP)
        print(f'== {w}: {len(pts)} job sites')
        for m in cluster(pts, 24)[:a.top]:
            if len(m) < 5:
                break
            ps = [p for p, _ in m]
            x, _, z = centre(ps)
            ys = [p[1] for p in ps]
            span = (max(p[0] for p in ps) - min(p[0] for p in ps), max(p[2] for p in ps) - min(p[2] for p in ps))
            kinds = collections.Counter(t for _, t in m).most_common(6)
            print(f'  {len(m):4d} @ {x}, {min(ys)}..{max(ys)}, {z}  span {span[0]}x{span[1]}  '
                  + ', '.join(f'{t}:{n}' for t, n in kinds))
    print('\nlibrarian=lectern, fisherman=barrel (often storage), farmer=composter. '
          'Tight, flat, mostly one profession = trading hall.')


# ---------- portals

def cmd_portals(a, names):
    for w in a.worlds:
        print(f'== {w}')
        for sub, label in (('', 'overworld'), ('DIM-1', 'nether')):
            ps = [p for p, _ in pois(os.path.join(w, sub), lambda t: t.endswith('nether_portal'))]
            cl = cluster(ps, 8, pos=lambda p: p)
            print(f'  {label} portals ({len(cl)}): ' + '  '.join('%d,%d,%d' % centre(m) for m in cl))
        beds = cluster([p for p, _ in pois(w, lambda t: t.endswith(':home'))], 16, pos=lambda p: p)
        big = [m for m in beds if len(m) >= a.min_beds]
        print(f'  bed clusters >= {a.min_beds} (villages or breeders): '
              + '  '.join('%d@%d,%d,%d' % ((len(m),) + centre(m)) for m in big[:a.top]))
        for uuid, d in player_files(w):
            sp = (d.get('SpawnX'), d.get('SpawnY'), d.get('SpawnZ'))
            pos = [int(v) for v in d.get('Pos', [0, 0, 0])]
            print(f'    {who(names, uuid):16} bed {sp}  last {pos} {d.get("Dimension")}')
    print('\nNether coords x8 = overworld. Portals at round nether distances suggest a highway.')


# ---------- builds

WEIGHT = {'hopper': 3, 'furnace': 2, 'smoker': 3, 'blast_furnace': 3, 'shulker_box': 4, 'brewing_stand': 4,
          'enchanting_table': 4, 'sign': 2, 'barrel': 1, 'chest': 1}
PLAYERISH = {'hopper', 'furnace', 'smoker', 'blast_furnace', 'shulker_box', 'sign', 'brewing_stand', 'enchanting_table'}


def kind(be_id):
    i = be_id.split(':')[-1]
    for k in ('blast_furnace', 'shulker_box', 'brewing_stand', 'enchanting_table', 'hopper', 'smoker', 'furnace',
              'sign', 'barrel', 'chest'):
        if k in i:
            return k
    return None


def sign_text(b):
    def one(s):
        try:
            j = json.loads(s)
        except Exception:
            return s
        if isinstance(j, str):
            return j
        if isinstance(j, dict):
            return j.get('text', '') + ''.join(e.get('text', '') if isinstance(e, dict) else str(e)
                                               for e in j.get('extra', []))
        return str(j)
    lines = b['front_text'].get('messages', []) if 'front_text' in b else [b.get(f'Text{n}', '') for n in range(1, 5)]
    return ' | '.join(x for x in (one(l).strip() for l in lines) if x)


def scan_region(path):
    out = []
    data = open(path, 'rb').read()
    if len(data) < 8192:
        return out
    for k in range(1024):
        off = int.from_bytes(data[k * 4:k * 4 + 3], 'big')
        if not off:
            continue
        p = off * 4096
        ln = struct.unpack_from('>i', data, p)[0]
        try:
            raw = zlib.decompress(data[p + 5:p + 4 + ln])
        except Exception:
            continue
        # cheap byte filter: skip chunks with nothing player-ish before a full parse
        if not any(w in raw for w in (b'hopper', b'furnace', b'shulker', b'sign', b'brewing', b'enchanting')):
            continue
        try:
            c = parse(raw)
        except Exception:
            continue
        for b in c.get('block_entities') or c.get('Level', {}).get('TileEntities', []):
            k_ = kind(b.get('id', ''))
            if k_:
                out.append([k_, b['x'], b['y'], b['z'], sign_text(b) if k_ == 'sign' else None])
    return out


def built_areas(w, json_dir=None):
    files = glob.glob(os.path.join(w, 'region', '*.mca'))
    with Pool() as pool:
        ents = [e for r in pool.map(scan_region, files, chunksize=4) for e in r]
    if json_dir:
        out = os.path.join(json_dir, f'blocks_{os.path.basename(os.path.normpath(w))}.json')
        json.dump(ents, open(out, 'w'))
        print(f'wrote {out}')
    ranked = []
    for m in cluster(ents, 32, pos=lambda e: (e[1], e[2], e[3])):
        cnt = collections.Counter(e[0] for e in m)
        if sum(cnt[k] for k in PLAYERISH) < 5:
            continue
        ranked.append((sum(WEIGHT[k] * v for k, v in cnt.items()), m, cnt))
    ranked.sort(key=lambda r: -r[0])
    return ents, ranked


def cmd_builds(a, names):
    for w in a.worlds:
        ents, ranked = built_areas(w, a.json)
        print(f'== {w}: {len(ents)} block entities, {len(ranked)} built areas')
        for _, m, cnt in ranked[:a.top]:
            print('  %d, %d, %d  ' % centre([(e[1], e[2], e[3]) for e in m])
                  + ' '.join(f'{k}:{v}' for k, v in cnt.most_common()))
            signs = collections.Counter(e[4] for e in m if e[4]).most_common(6)
            if signs:
                print('      signs: ' + ' // '.join(s for s, _ in signs))


# ---------- mapconfig

REGION = 512


def slug(s):
    out = ''.join(ch.lower() if ch.isalnum() else '_' for ch in s.replace("'", ''))
    return '_'.join(p for p in out.split('_') if p)[:32]


def area_name(signs, fallback):
    # Prefer a sign that reads like a label ("Trading Hub", "Paul's Shops!") over prices/enchant lists
    for s, _ in collections.Counter(signs).most_common():
        first = s.split(' | ')[0]
        if sum(ch.isalpha() for ch in first) >= 5 and not any(ch.isdigit() for ch in first):
            return slug(first)
    return fallback


def to_range(name, pts, min_radius, pad):
    xs, zs = [p[0] for p in pts], [p[2] for p in pts]
    cx, cz = (min(xs) + max(xs)) // 2, (min(zs) + max(zs)) // 2
    half = max(max(xs) - min(xs), max(zs) - min(zs)) // 2 + pad
    return {'name': name, 'center': [cx, cz], 'radius': max(min_radius, -(-half // 64) * 64), 'n': len(pts)}


def merge_ranges(rs, min_radius, pad):
    """Merge ranges when one's center falls inside the other's square; keep the bigger one's name.
    Merely touching squares stay separate: a merged square costs more tiles than two small ones."""
    rs = sorted(rs, key=lambda r: -r['n'])
    changed = True
    while changed:
        changed = False
        for i in range(len(rs)):
            for j in range(i + 1, len(rs)):
                a, b = rs[i], rs[j]
                reach = max(a['radius'], b['radius'])
                if abs(a['center'][0] - b['center'][0]) <= reach and abs(a['center'][1] - b['center'][1]) <= reach:
                    pts = [(a['center'][0] + sx * a['radius'] - sx * pad, 0, a['center'][1] + sz * a['radius'] - sz * pad)
                           for sx in (-1, 1) for sz in (-1, 1)]
                    pts += [(b['center'][0] + sx * b['radius'] - sx * pad, 0, b['center'][1] + sz * b['radius'] - sz * pad)
                            for sx in (-1, 1) for sz in (-1, 1)]
                    m = to_range(a['name'], pts, min_radius, pad)
                    m['n'] = a['n'] + b['n']
                    m['why'] = '; '.join(x for x in (a.get('why'), b.get('why')) if x)
                    rs[i] = m
                    del rs[j]
                    changed = True
                    break
            if changed:
                break
    return rs


def regions(r):
    span = lambda c: (-(-(c + r['radius']) // REGION)) - ((c - r['radius']) // REGION)
    return span(r['center'][0]) * span(r['center'][1])


def emit(map_name, rs, zoomin):
    print(f'  # {map_name}: {len(rs)} ranges, {sum(regions(r) for r in rs)} region tiles at zoomin {zoomin}')
    print('    ranges:')
    for r in rs:
        print(f'      - name: {r["name"]}  # {r.get("why", "")}')
        print(f'        center: [{r["center"][0]}, {r["center"][1]}]')
        print(f'        radius: {r["radius"]}')
        print(f'        zoomin: {zoomin}')


def cmd_mapconfig(a, names):
    for w in a.worlds:
        rs = []
        _, ranked = built_areas(w)
        for _, m, cnt in ranked[:a.top]:
            pts = [(e[1], e[2], e[3]) for e in m]
            x, _, z = centre(pts)
            r = to_range(area_name([e[4] for e in m if e[4]], f'base_{x}_{z}'), pts, a.min_radius, a.pad)
            r['why'] = ' '.join(f'{k}:{v}' for k, v in cnt.most_common(4))
            rs.append(r)
        for m in cluster(pois(w, lambda t: t.split(':')[-1] not in POI_SKIP), 24)[:a.top]:
            if len(m) < 15:
                break
            ps = [p for p, _ in m]
            top_kind, n = collections.Counter(t for _, t in m).most_common(1)[0]
            label = {'librarian': 'trading_hall', 'farmer': 'crop_farm', 'fisherman': 'barrels'}.get(top_kind, top_kind)
            x, _, z = centre(ps)
            r = to_range(f'{label}_{x}_{z}', ps, a.min_radius, a.pad)
            r['why'] = f'{len(m)} job sites, {n} {top_kind}'
            rs.append(r)
        over = merge_ranges(rs, a.min_radius, a.pad)

        nether = []
        for m in cluster([p for p, _ in pois(os.path.join(w, 'DIM-1'), lambda t: t.endswith('nether_portal'))], 64,
                         pos=lambda p: p):
            x, _, z = centre(m)
            r = to_range(f'portal_{x}_{z}', m, a.min_radius // 2, a.pad)
            r['why'] = f'{len(m)} portal blocks'
            nether.append(r)
        nether = merge_ranges(nether, a.min_radius // 2, a.pad)[:a.top]

        print(f'# {w}: suggested ranges, review before pasting into map-config.yml')
        print('maps:\n  - name: overworld\n    dimension: overworld')
        emit('overworld', over, a.zoomin)
        if nether:
            print('  - name: nether\n    dimension: nether')
            emit('nether', nether, a.zoomin)
        print()


# ---------- players

def cmd_players(a, names):
    for w in a.worlds:
        rows = []
        for f in glob.glob(os.path.join(w, 'stats', '*.json')):
            d = json.load(open(f))
            s = d.get('stats', {})
            c = s.get('minecraft:custom', {})
            ticks = c.get('minecraft:play_time', c.get('minecraft:play_one_minute', 0))
            km = sum(v for k, v in c.items() if k.endswith('_one_cm')) / 100000
            rows.append((ticks / 72000, who(names, os.path.basename(f)[:-5]), km,
                         sum(s.get('minecraft:mined', {}).values()), c.get('minecraft:deaths', 0)))
        rows.sort(reverse=True)
        print(f'== {w}')
        for h, n, km, mined, deaths in rows:
            print(f'  {n:16} {h:7.1f} h  {km:7.1f} km  {mined:7d} mined  {deaths:4d} deaths')
        print(f'  total {sum(r[0] for r in rows):.1f} h, {len(rows)} players')
    print('\nPlay time is time connected, AFK included.')


# ---------- inventory

def item_name(it):
    n = it['id'].split(':')[-1]
    tag = it.get('tag', {})
    disp = tag.get('display', {}).get('Name')
    if disp:
        try:
            j = json.loads(disp)
            disp = j.get('text', '') if isinstance(j, dict) else str(j)
        except Exception:
            pass
        n += f' "{disp}"'
    ench = tag.get('Enchantments') or tag.get('StoredEnchantments') or []
    if ench:
        n += ' [' + ', '.join(f"{e['id'].split(':')[-1]} {e['lvl']}" for e in ench) + ']'
    return n


def show_items(items, label):
    print(f'  {label} ({len(items)})')
    for it in sorted(items, key=lambda i: i.get('Slot', 0)):
        c = it.get('Count', 1)
        print(f'    {it.get("Slot", ""):>4}  ' + (f'{c}x ' if c > 1 else '') + item_name(it))
        inner = it.get('tag', {}).get('BlockEntityTag', {}).get('Items', []) if 'shulker_box' in it['id'] else []
        if inner:
            tot = collections.Counter()
            for x in inner:
                tot[item_name(x)] += x.get('Count', 1)
            print('          ' + ', '.join(f'{c}x {n}' if c > 1 else n for n, c in tot.most_common()))


def cmd_inventory(a, names):
    want = a.player.lower()
    for w in a.worlds:
        hits = [(u, d) for u, d in player_files(w) if want in (u.lower(), who(names, u).lower()) or u.startswith(want)]
        if not hits:
            print(f'== {w}: no player matching {a.player}')
            continue
        for u, d in hits:
            inv = d.get('Inventory', [])
            print(f'== {w}  {who(names, u)}  XP {d.get("XpLevel")}  last {[int(v) for v in d["Pos"]]} {d.get("Dimension")}')
            show_items([i for i in inv if 100 <= i.get('Slot', 0) <= 103], 'armor')
            show_items([i for i in inv if i.get('Slot') == -106], 'offhand')
            show_items([i for i in inv if 0 <= i.get('Slot', 0) <= 35], 'inventory (0-8 hotbar)')
            show_items(d.get('EnderItems', []), 'ender chest')


# ---------- villagers

# Villager.canBreed(): food level + inventory food points >= 12, adult (Age 0), plus a free bed nearby.
FOOD_POINTS = {'bread': 4, 'carrot': 1, 'potato': 1, 'beetroot': 1}
DIMS = {'overworld': '', 'nether': 'DIM-1', 'end': 'DIM1'}


def chunk_nbt(path, needle):
    """Parse only the chunks in a region file whose raw bytes contain needle."""
    data = open(path, 'rb').read()
    if len(data) < 8192:
        return
    for k in range(1024):
        off = int.from_bytes(data[k * 4:k * 4 + 3], 'big')
        if not off:
            continue
        p = off * 4096
        ln = struct.unpack_from('>i', data, p)[0]
        try:
            raw = zlib.decompress(data[p + 5:p + 4 + ln])
            if needle in raw:
                yield parse(raw)
        except Exception:
            continue


def villagers(w, dim):
    d = os.path.join(w, DIMS[dim])
    # 1.17+ keeps entities in entities/; 1.16 keeps them in the terrain chunk
    for f in glob.glob(os.path.join(d, 'entities', '*.mca')):
        for c in chunk_nbt(f, b'minecraft:villager'):
            yield from (e for e in c.get('Entities', []) if e.get('id') == 'minecraft:villager')
    if not glob.glob(os.path.join(d, 'entities', '*.mca')):
        for f in glob.glob(os.path.join(d, 'region', '*.mca')):
            for c in chunk_nbt(f, b'minecraft:villager'):
                yield from (e for e in c.get('Level', {}).get('Entities', []) if e.get('id') == 'minecraft:villager')


def beds(w, dim):
    out = []
    for f in glob.glob(os.path.join(w, DIMS[dim], 'poi', '*.mca')):
        for c in chunk_nbt(f, b'minecraft:home'):
            for sec in c.get('Sections', {}).values():
                out += [(r['pos'], r.get('free_tickets', 0)) for r in sec.get('Records', []) if r['type'] == 'minecraft:home']
    return out


def cmd_villagers(a, names):
    cx, cz = (int(v) for v in a.near.split(',')) if a.near else (None, None)
    near = lambda x, z: cx is None or (x - cx) ** 2 + (z - cz) ** 2 <= a.radius ** 2
    for w in a.worlds:
        vs = [v for v in villagers(w, a.dim) if near(v['Pos'][0], v['Pos'][2])]
        print(f'== {w} ({a.dim}): {len(vs)} villagers' + (f' within {a.radius} of {cx},{cz}' if a.near else ''))
        willing = 0
        for v in sorted(vs, key=lambda v: (v['Pos'][0], v['Pos'][2])):
            x, y, z = (int(p) for p in v['Pos'])
            vd = v.get('VillagerData', {})
            prof = vd.get('profession', '?').split(':')[-1]
            age = v.get('Age', 0)
            food = v.get('FoodLevel', 0)
            inv = v.get('Inventory', [])
            pts = food + sum(FOOD_POINTS.get(i['id'].split(':')[-1], 0) * i.get('Count', 1) for i in inv)
            if age < 0:
                state = f'baby ({-age // 20}s to grow)'
            elif age > 0:
                state = f'cooldown {age // 20}s'
            elif pts >= 12:
                state, willing = 'WILLING', willing + 1
            else:
                state = f'hungry, needs {12 - pts} more food points'
            home = v.get('Brain', {}).get('memories', {}).get('minecraft:home')
            name = v.get('CustomName', '')
            if name:
                try:
                    name = json.loads(name).get('text', name)
                except Exception:
                    pass
            print(f'  {x:6d} {y:4d} {z:6d}  {prof:13} L{vd.get("level", 1)}  food {food:2d}+inv = {pts:3d}  '
                  f'{"bed" if home else "NO BED"}  {state}' + (f'  "{name}"' if name else ''))
            if inv:
                print('          ' + ', '.join(f'{i.get("Count", 1)}x {item_name(i)}' for i in inv))
        print(f'  {willing} willing')
        if a.near:
            bs = [(p, t) for p, t in beds(w, a.dim) if near(p[0], p[2])]
            print(f'  beds within {a.radius}: {len(bs)}, unclaimed: {sum(1 for _, t in bs if t > 0)}')
    print('\nWilling = adult, off cooldown, food + inventory >= 12 (bread 4; carrot/potato/beetroot 1). '
          'Two willing villagers also need an unclaimed bed nearby to make a baby.')


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--usercache', help='usercache.json for UUID -> name')
    sub = ap.add_subparsers(dest='cmd', required=True)
    for name in ('halls', 'portals', 'builds', 'mapconfig', 'players', 'inventory', 'villagers'):
        p = sub.add_parser(name)
        p.add_argument('worlds', nargs='+')
        p.add_argument('--top', type=int, default=8 if name == 'mapconfig' else 15)
        if name == 'mapconfig':
            p.add_argument('--zoomin', type=int, default=2)
            p.add_argument('--min-radius', type=int, default=256)
            p.add_argument('--pad', type=int, default=64, help='blocks added around each detected area')
        if name == 'portals':
            p.add_argument('--min-beds', type=int, default=5)
        if name == 'builds':
            p.add_argument('--json', metavar='DIR', help='also dump every block entity as [kind,x,y,z,sign_text]')
        if name == 'inventory':
            p.add_argument('--player', required=True)
        if name == 'villagers':
            p.add_argument('--near', metavar='X,Z', help='only villagers around this block x,z (write --near=X,Z when X is negative)')
            p.add_argument('--radius', type=int, default=32)
            p.add_argument('--dim', choices=DIMS, default='overworld')
    a = ap.parse_args()
    for w in a.worlds:
        if not os.path.exists(os.path.join(w, 'level.dat')):
            sys.exit(f'{w}: no level.dat (pass the save directory)')
    globals()['cmd_' + a.cmd](a, load_names(a.worlds, a.usercache))


if __name__ == '__main__':
    main()
