#!/usr/bin/env python3
"""Find things in a Minecraft Java save without opening the game.

Read-only. Works on 1.16-1.20.4 saves (pre-1.20.5 item format; chunk formats from 1.16 on).

  scan.py hotspots  <world>...   where players spent time (chunk InhabitedTime peaks)
  scan.py halls     <world>...   villager job-site clusters (trading halls, crop farms, barrel rooms)
  scan.py portals   <world>...   nether portals (overworld + nether), bed clusters, player spawn/last position
  scan.py builds    <world>...   player-built areas scored by block entities, with sign/banner/book/head labels
  scan.py entities  <world>...   named mobs, pets by owner, map walls, item frames, armor stands, master villagers
  scan.py blocks    <world>...   chunks holding redstone, decorative, treasure blocks; nether ice roads
  scan.py maps      <world>...   map items: explored areas, locked copies, where maps hang
  scan.py mapconfig <world>...   suggested map-config.yml ranges around builds, halls, hotspots, portal hubs
  scan.py players   <world>...   play time, distance, blocks mined, deaths, last death location
  scan.py inventory <world>... --player NAME|UUID   inventory, armor, ender chest, shulker contents

<world> is a save directory (the one holding level.dat). Player names come from a
usercache.json next to the save (server layout), the local client's, or --usercache.
"""
import argparse, collections, glob, gzip, json, os, re, struct, sys, zlib
from multiprocessing import Pool

sys.dont_write_bytecode = True  # no __pycache__ in the skill dir
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from mcnbt import parse, region_chunks  # noqa: E402

CLIENT_USERCACHE = os.path.expanduser('~/Library/Application Support/minecraft/usercache.json')
DIMS = (('', 'overworld'), ('DIM-1', 'nether'), ('DIM1', 'end'))


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
    return names.get(uuid, uuid[:8]) if uuid else '?'


def uuid_str(v):
    if isinstance(v, list) and len(v) == 4:
        h = ''.join(f'{i & 0xffffffff:08x}' for i in v)
        return f'{h[:8]}-{h[8:12]}-{h[12:16]}-{h[16:20]}-{h[20:]}'
    return v if isinstance(v, str) and v else None


def text_of(s):
    """Plain text from a JSON text component (sign lines, custom names)."""
    if not s:
        return ''
    try:
        j = json.loads(s)
    except Exception:
        return s
    if isinstance(j, str):
        return j
    if isinstance(j, list):
        return ''.join(text_of(json.dumps(x)) for x in j)
    if isinstance(j, dict):
        return j.get('text', '') + ''.join(text_of(json.dumps(e)) for e in j.get('extra', []))
    return str(j)


def chunk_blobs(path):
    """Yield (chunk_x, chunk_z, raw NBT bytes) for each chunk in a region file, without parsing."""
    data = open(path, 'rb').read()
    if len(data) < 8192:
        return
    rx, rz = map(int, os.path.basename(path).split('.')[1:3])
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
        yield rx * 32 + k % 32, rz * 32 + k // 32, raw


def over_regions(region_dir, fn):
    """Run fn(path) -> list over every .mca in a dir, in parallel."""
    files = glob.glob(os.path.join(region_dir, '*.mca'))
    if not files:
        return []
    with Pool() as pool:
        return [e for r in pool.map(fn, files, chunksize=4) for e in r]


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


# ---------- hotspots

INHABITED = b'\x04\x00\x0dInhabitedTime'


def scan_inhabited(path):
    out = []
    for cx, cz, raw in chunk_blobs(path):
        i = raw.find(INHABITED)
        if i >= 0:
            h = struct.unpack_from('>q', raw, i + len(INHABITED))[0] / 72000
            if h >= 0.5:
                out.append((cx, cz, h))
    return out


def hotspots(w, min_hours=None, radius=6):
    """Local peaks of InhabitedTime. It accrues on every chunk near a player, so a base shows
    up as a plateau; a chunk that's the max within `radius` chunks marks one spot."""
    hours = {(cx, cz): h for cx, cz, h in over_regions(os.path.join(w, 'region'), scan_inhabited)}
    if not hours:
        return [], 0
    top = max(hours.values())
    thr = min_hours if min_hours is not None else max(1.0, top * 0.05)
    peaks = []
    for (cx, cz), h in hours.items():
        if h < thr:
            continue
        if all(hours.get((cx + dx, cz + dz), 0) <= h
               for dx in range(-radius, radius + 1) for dz in range(-radius, radius + 1)):
            peaks.append((cx * 16 + 8, 0, cz * 16 + 8, h))
    spots = []
    for m in cluster(peaks, 16 * 3, pos=lambda p: p):  # plateau ties -> one spot
        h = max(p[3] for p in m)
        plateau = sum(1 for (cx, cz), v in hours.items() if v >= h * 0.9
                      and any(abs(cx * 16 + 8 - p[0]) <= radius * 16 and abs(cz * 16 + 8 - p[2]) <= radius * 16 for p in m))
        x, _, z = centre([(p[0], 0, p[2]) for p in m])
        spots.append({'x': x, 'z': z, 'hours': h, 'plateau': plateau})
    return sorted(spots, key=lambda s: -s['hours']), len(hours)


def cmd_hotspots(a, names):
    for w in a.worlds:
        spots, n = hotspots(w, a.min_hours)
        print(f'== {w}: {n} chunks visited, {len(spots)} hotspots')
        for s in spots[:a.top]:
            print(f'  {s["hours"]:7.1f} h  @ {s["x"]}, {s["z"]}   ({s["plateau"]} chunks within 90%)')
    print('\nHours a player was nearby (any player, not summed). Peaks mark where people stood most:'
          ' cross-check with builds/entities for what is there.')


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
        for sub, label in DIMS[:2]:
            ps = [p for p, _ in pois(os.path.join(w, sub), lambda t: t.endswith('nether_portal'))]
            cl = cluster(ps, 8, pos=lambda p: p)
            print(f'  {label} portals ({len(cl)}): ' + '  '.join('%d,%d,%d' % centre(m) for m in cl))
        lode = [p for p, _ in pois(w, lambda t: t.endswith('lodestone'))]
        if lode:
            print('  lodestones: ' + '  '.join('%d,%d,%d' % p for p in lode))
        beds = cluster([p for p, _ in pois(w, lambda t: t.endswith(':home'))], 16, pos=lambda p: p)
        big = [m for m in beds if len(m) >= a.min_beds]
        print(f'  bed clusters >= {a.min_beds} (villages or breeders): '
              + '  '.join('%d@%d,%d,%d' % ((len(m),) + centre(m)) for m in big[:a.top]))
        for uuid, d in player_files(w):
            sp = (d.get('SpawnX'), d.get('SpawnY'), d.get('SpawnZ'))
            pos = [int(v) for v in d.get('Pos', [0, 0, 0])]
            print(f'    {who(names, uuid):16} bed {sp}  last {pos} {d.get("Dimension")}')
    print('\nNether coords x8 = overworld. Portals at round nether distances suggest a highway.')


# ---------- builds (block entities)

# weight = how strongly a block entity says "a player built here"
WEIGHT = {'beacon': 10, 'conduit': 8, 'ender_chest': 6, 'shulker_box': 4, 'brewing_stand': 4, 'enchanting_table': 4,
          'lectern_book': 3, 'jukebox': 3, 'skull': 3, 'hopper': 3, 'smoker': 3, 'blast_furnace': 3,
          'comparator': 2, 'daylight_detector': 2, 'banner': 2, 'sign': 2, 'furnace': 2,
          'barrel': 1, 'chest': 1, 'spawner': 0, 'plain_banner': 0, 'loot_chest': 0}
# worldgen makes spawners, plain banners (villages) and loot chests, so they don't count toward "built"
PLAYERISH = set(WEIGHT) - {'barrel', 'chest', 'spawner', 'plain_banner', 'loot_chest'}
# substring of the block entity id -> kind; order matters (ender_chest before chest, blast_furnace before furnace)
KINDS = [('ender_chest', 'ender_chest'), ('blast_furnace', 'blast_furnace'), ('shulker_box', 'shulker_box'),
         ('brewing_stand', 'brewing_stand'), ('enchanting_table', 'enchanting_table'), ('daylight_detector', 'daylight_detector'),
         ('mob_spawner', 'spawner'), ('beacon', 'beacon'), ('conduit', 'conduit'), ('jukebox', 'jukebox'),
         ('comparator', 'comparator'), ('skull', 'skull'), ('banner', 'banner'), ('lectern', 'lectern'),
         ('hopper', 'hopper'), ('smoker', 'smoker'), ('furnace', 'furnace'), ('sign', 'sign'),
         ('barrel', 'barrel'), ('chest', 'chest')]
BE_FILTER = (b'hopper', b'furnace', b'shulker', b'sign', b'brewing', b'enchanting', b'beacon', b'conduit',
             b'ender_chest', b'jukebox', b'comparator', b'daylight', b'skull', b'banner', b'lectern', b'spawner')


def kind(be_id):
    i = be_id.split(':')[-1]
    return next((k for sub, k in KINDS if sub in i), None)


def sign_text(b):
    lines = b['front_text'].get('messages', []) if 'front_text' in b else [b.get(f'Text{n}', '') for n in range(1, 5)]
    return ' | '.join(x for x in (text_of(l).strip() for l in lines) if x)


def be_detail(k, b):
    """(kind, label) for a block entity; label is sign text, banner name, head owner, disc, book title..."""
    if k == 'sign':
        return k, sign_text(b)
    if k == 'banner':
        name = text_of(b.get('CustomName'))
        return ('banner', name) if name or b.get('Patterns') else ('plain_banner', None)
    if k in ('chest', 'barrel') and 'LootTable' in b:
        return 'loot_chest', None  # never opened: a structure's chest, not a player's
    if k == 'skull':
        o = b.get('SkullOwner')
        return k, o.get('Name', '') if isinstance(o, dict) else (o or '')
    if k == 'jukebox':
        return k, b.get('RecordItem', {}).get('id', '').split(':')[-1]
    if k == 'lectern':
        book = b.get('Book')
        if not book:
            return None, None  # empty lecterns are job sites; halls covers those
        t = book.get('tag', {})
        return 'lectern_book', t.get('title') or text_of((t.get('pages') or [''])[0])[:40]
    if k == 'spawner':
        sd = b.get('SpawnData', {})
        return k, (sd.get('entity', sd).get('id') or '').split(':')[-1]
    if k == 'beacon':
        return k, f'level {b.get("Levels", 0)}'
    return k, None


def scan_region(path):
    out = []
    for _, _, raw in chunk_blobs(path):
        if not any(w in raw for w in BE_FILTER):  # skip chunks with nothing player-ish before a full parse
            continue
        try:
            c = parse(raw)
        except Exception:
            continue
        for b in c.get('block_entities') or c.get('Level', {}).get('TileEntities', []):
            k = kind(b.get('id', ''))
            if k:
                k, label = be_detail(k, b)
                if k:
                    out.append([k, b['x'], b['y'], b['z'], label or None])
    return out


def built_areas(w, json_dir=None):
    ents = over_regions(os.path.join(w, 'region'), scan_region)
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


def labels(m, cnt):
    """Short descriptions of the notable block entities in a cluster."""
    out = []
    for k, prefix in (('banner', 'banner'), ('skull', 'head'), ('jukebox', 'disc'), ('lectern_book', 'book'),
                      ('beacon', 'beacon')):
        vals = collections.Counter(e[4] for e in m if e[0] == k and e[4])
        if vals:
            out.append(f'{prefix}: ' + ', '.join(f'{v} x{n}' if n > 1 else v for v, n in vals.most_common(5)))
    spawners = collections.Counter(e[4] or '?' for e in m if e[0] == 'spawner')
    if spawners:
        out.append('spawner: ' + ', '.join(spawners) + (' (grinder?)' if cnt['hopper'] >= 3 else ''))
    return out


def cmd_builds(a, names):
    for w in a.worlds:
        ents, ranked = built_areas(w, a.json)
        print(f'== {w}: {len(ents)} block entities, {len(ranked)} built areas')
        for _, m, cnt in ranked[:a.top]:
            print('  %d, %d, %d  ' % centre([(e[1], e[2], e[3]) for e in m])
                  + ' '.join(f'{k}:{v}' for k, v in cnt.most_common()))
            for l in labels(m, cnt):
                print(f'      {l}')
            signs = collections.Counter(e[4] for e in m if e[0] == 'sign' and e[4]).most_common(6)
            if signs:
                print('      signs: ' + ' // '.join(s for s, _ in signs))


# ---------- entities

ENT_FILTER = (b'CustomName', b'Owner', b'item_frame', b'armor_stand', b'villager', b'hopper_minecart')
TAMEABLE = {'wolf', 'cat', 'parrot', 'ocelot'}  # horse family, llamas: use their Tame flag


def scan_entities(path):
    out = []
    for _, _, raw in chunk_blobs(path):
        if not any(f in raw for f in ENT_FILTER):
            continue
        try:
            c = parse(raw)
        except Exception:
            continue
        for e in c.get('Entities') or c.get('Level', {}).get('Entities', []):
            eid = e.get('id', '').split(':')[-1]
            try:
                x, y, z = (int(v) for v in e['Pos'])
            except Exception:
                continue
            name = text_of(e.get('CustomName'))
            if name:
                out.append(['named', eid, x, y, z, name])
            owner = uuid_str(e.get('Owner')) or uuid_str(e.get('OwnerUUID'))
            # projectiles also have an Owner (whoever shot them); only count tamed animals
            if owner and (eid in TAMEABLE or e.get('Tame')):
                out.append(['pet', eid, x, y, z, owner])
            if eid in ('item_frame', 'glow_item_frame'):
                item = e.get('Item', {}).get('id', '').split(':')[-1]
                if item:
                    out.append(['map_frame' if item == 'filled_map' else 'frame', item, x, y, z,
                                e.get('Item', {}).get('tag', {}).get('map')])
            elif eid == 'armor_stand':
                worn = [i.get('id', '').split(':')[-1] for i in e.get('ArmorItems', []) if i.get('id')]
                out.append(['armor_stand', eid, x, y, z, ', '.join(worn)])
            elif eid == 'villager' and e.get('VillagerData', {}).get('level', 0) >= 5:
                out.append(['master_villager', e['VillagerData'].get('profession', '').split(':')[-1], x, y, z, None])
            elif eid == 'hopper_minecart':  # chest minecarts are mostly mineshaft loot
                out.append(['cart', eid, x, y, z, None])
    return out


def entity_dirs(w, sub):
    d = os.path.join(w, sub)
    ents = os.path.join(d, 'entities')  # 1.17+ keeps entities apart from terrain
    return ents if glob.glob(os.path.join(ents, '*.mca')) else os.path.join(d, 'region')


def cmd_entities(a, names):
    for w in a.worlds:
        for sub, dim in DIMS:
            es = over_regions(entity_dirs(w, sub), scan_entities)
            if not es:
                continue
            by = collections.defaultdict(list)
            for e in es:
                by[e[0]].append(e)
            print(f'== {w} {dim}: ' + ', '.join(f'{k}:{len(v)}' for k, v in sorted(by.items())))
            pos = lambda e: (e[2], e[3], e[4])
            for e in sorted(by['named'], key=lambda e: e[5])[:a.top * 3]:
                print(f'  named   {e[5]!r:28} {e[1]:14} @ {e[2]}, {e[3]}, {e[4]}')
            owners = collections.defaultdict(list)
            for e in by['pet']:
                owners[e[5]].append(e)
            for o, pets in sorted(owners.items(), key=lambda kv: -len(kv[1])):
                spots = cluster(pets, 32, pos=pos)
                print(f'  pets of {who(names, o)}: ' + '  '.join(
                    f'{len(m)} ' + '/'.join(sorted({p[1] for p in m})) + ' @%d,%d,%d' % centre([pos(p) for p in m])
                    for m in spots[:5]))
            for k, label, minimum in (('map_frame', 'map wall', 2), ('frame', 'framed items', 4),
                                      ('armor_stand', 'armor stands', 2), ('master_villager', 'master villagers', 3),
                                      ('cart', 'hopper carts', 2)):
                groups = [m for m in cluster(by[k], 16, pos=pos) if len(m) >= minimum]
                for m in groups[:a.top]:
                    what = collections.Counter(e[1] for e in m).most_common(4)
                    print(f'  {label:16} {len(m):4d} @ %d, %d, %d  ' % centre([pos(e) for e in m])
                          + ', '.join(f'{t}:{n}' for t, n in what))


# ---------- blocks (chunk palettes)

# Only block palette entries carry a "Name" string tag; items, entities and block entities use "id".
# So this regex sees placed blocks, not a chest full of concrete.
PALETTE_RE = re.compile(rb'\x08\x00\x04Name\x00[\x00-\xff]minecraft:([a-z0-9_]+)', re.S)
QUARTZ = {'quartz_block', 'quartz_pillar', 'quartz_bricks', 'smooth_quartz', 'chiseled_quartz_block',
          'quartz_stairs', 'quartz_slab', 'smooth_quartz_stairs', 'smooth_quartz_slab'}
REDSTONE = {'piston', 'sticky_piston', 'observer', 'slime_block', 'honey_block', 'redstone_lamp', 'powered_rail',
            'detector_rail', 'activator_rail', 'target', 'comparator', 'daylight_detector'}
TREASURE = {'netherite_block', 'diamond_block', 'emerald_block'}
# structures that generate the blocks above (ancient cities: lamps, sticky pistons, comparators, targets;
# trail ruins: glazed terracotta). Hits within NATURAL_REACH blocks of a chunk holding any of these are dropped.
NATURAL = {b'reinforced_deepslate', b'sculk_shrieker', b'sculk_catalyst', b'suspicious_gravel', b'suspicious_sand'}
NATURAL_REACH = 128


def block_category(name, dim):
    if name in TREASURE:
        return 'treasure'
    if name in REDSTONE:
        return 'redstone'
    if name.endswith('_concrete') or name.endswith('_glazed_terracotta') or name in QUARTZ or name == 'scaffolding':
        return 'decor'
    if dim == 'nether':
        if name in ('blue_ice', 'packed_ice'):
            return 'ice_road'
        if name in ('glass', 'glass_pane') or name.endswith('stained_glass') or name.endswith('_planks'):
            return 'nether_build'
    return None


def scan_palettes(args):
    path, dim = args
    out = []
    for cx, cz, raw in chunk_blobs(path):
        cats = collections.defaultdict(set)
        found = {m.group(1) for m in PALETTE_RE.finditer(raw)}
        if found & NATURAL:
            out.append(('_natural', cx * 16 + 8, 0, cz * 16 + 8, []))
            continue
        for n in found:
            n = n.decode()
            c = block_category(n, dim)
            if c:
                cats[c].add(n)
        for c, ns in cats.items():
            out.append((c, cx * 16 + 8, 0, cz * 16 + 8, sorted(ns)))
    return out


def cmd_blocks(a, names):
    for w in a.worlds:
        for sub, dim in DIMS[:2]:
            files = [(f, dim) for f in glob.glob(os.path.join(w, sub, 'region', '*.mca'))]
            if not files:
                continue
            with Pool() as pool:
                hits = [e for r in pool.map(scan_palettes, files, chunksize=4) for e in r]
            near = {(h[1] // NATURAL_REACH, h[3] // NATURAL_REACH) for h in hits if h[0] == '_natural'}
            by = collections.defaultdict(list)
            dropped = 0
            for h in hits:
                if h[0] == '_natural':
                    continue
                gx, gz = h[1] // NATURAL_REACH, h[3] // NATURAL_REACH
                if any((gx + dx, gz + dz) in near for dx in (-1, 0, 1) for dz in (-1, 0, 1)):
                    dropped += 1
                    continue
                by[h[0]].append(h)
            print(f'== {w} {dim}: ' + ', '.join(f'{k}:{len(v)} chunks' for k, v in sorted(by.items()))
                  + (f' ({dropped} near ancient cities/trail ruins dropped)' if dropped else ''))
            for cat in sorted(by):
                for m in cluster(by[cat], 16, pos=lambda h: (h[1], h[2], h[3]))[:a.top]:
                    if len(m) < a.min_chunks:
                        break
                    xs, zs = [h[1] for h in m], [h[3] for h in m]
                    blocks = collections.Counter(b for h in m for b in h[4]).most_common(5)
                    print(f'  {cat:12} {len(m):4d} chunks @ %d, %d  x {min(xs)}..{max(xs)} z {min(zs)}..{max(zs)}  '
                          % (centre([(h[1], 0, h[3]) for h in m])[0], centre([(h[1], 0, h[3]) for h in m])[2])
                          + ', '.join(b for b, _ in blocks))
    print('\nLong thin ice_road clusters in the nether are highways (x/z extents show the length).')


# ---------- maps (map items)

def cmd_maps(a, names):
    for w in a.worlds:
        rows = []
        for f in glob.glob(os.path.join(w, 'data', 'map_*.dat')):
            try:
                d = parse(gzip.decompress(open(f, 'rb').read())).get('data', {})
            except Exception:
                continue
            dim = d.get('dimension')
            dim = {0: 'overworld', -1: 'the_nether', 1: 'the_end'}.get(dim, str(dim).split(':')[-1])
            rows.append({'id': os.path.basename(f)[4:-4], 'x': d.get('xCenter', 0), 'z': d.get('zCenter', 0),
                         'scale': d.get('scale', 0), 'dim': dim, 'locked': bool(d.get('locked')),
                         'frames': [(fr['Pos']['X'], fr['Pos']['Y'], fr['Pos']['Z']) for fr in d.get('frames', [])],
                         'banners': [(text_of(b.get('Name')), b['Pos']['X'], b['Pos']['Z']) for b in d.get('banners', [])
                                     if b.get('Name')]})
        print(f'== {w}: {len(rows)} maps, {sum(r["locked"] for r in rows)} locked (copies for walls or map art)')
        for scale in sorted({r['scale'] for r in rows}):
            rs = [r for r in rows if r['scale'] == scale and r['dim'] == 'overworld']
            size = 128 << scale
            groups = cluster(rs, size * 2, pos=lambda r: (r['x'], 0, r['z']))
            print(f'  scale {scale} ({size} blocks): {len(rs)} overworld maps, explored areas: ' + '  '.join(
                f'{len(m)}@{centre([(r["x"], 0, r["z"]) for r in m])[0]},{centre([(r["x"], 0, r["z"]) for r in m])[2]}'
                for m in groups[:a.top]))
        frames = [f for r in rows for f in r['frames']]
        if frames:
            print('  hung in item frames: ' + '  '.join(f'{len(m)}@%d,%d,%d' % centre(m)
                                                       for m in cluster(frames, 16, pos=lambda p: p)[:a.top]))
        for r in rows:
            for b in r['banners']:
                print(f'  banner marker {b[0]!r} @ {b[1]}, {b[2]} (map {r["id"]})')


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
            r = to_range(area_name([e[4] for e in m if e[0] == 'sign' and e[4]], f'base_{x}_{z}'), pts, a.min_radius, a.pad)
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
        spots, _ = hotspots(w)
        for s in spots[:a.top]:
            if s['hours'] < a.min_hours:
                break
            r = to_range(f'hotspot_{s["x"]}_{s["z"]}', [(s['x'], 0, s['z'])], a.min_radius, a.pad)
            r['n'] = 1  # never let a hotspot's name win a merge over a named build
            r['why'] = f'{s["hours"]:.0f} h inhabited'
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
        deaths = {}
        for uuid, d in player_files(w):
            ld = d.get('LastDeathLocation')  # 1.19+
            if ld:
                deaths[uuid] = f'{ld.get("dimension", "").split(":")[-1]} {",".join(str(v) for v in ld.get("pos", []))}'
        rows = []
        for f in glob.glob(os.path.join(w, 'stats', '*.json')):
            d = json.load(open(f))
            s = d.get('stats', {})
            c = s.get('minecraft:custom', {})
            uuid = os.path.basename(f)[:-5]
            ticks = c.get('minecraft:play_time', c.get('minecraft:play_one_minute', 0))
            km = sum(v for k, v in c.items() if k.endswith('_one_cm')) / 100000
            rows.append((ticks / 72000, who(names, uuid), km, sum(s.get('minecraft:mined', {}).values()),
                         c.get('minecraft:deaths', 0), deaths.get(uuid, '')))
        rows.sort(reverse=True)
        print(f'== {w}')
        for h, n, km, mined, dead, where in rows:
            print(f'  {n:16} {h:7.1f} h  {km:7.1f} km  {mined:7d} mined  {dead:4d} deaths'
                  + (f'  last died {where}' if where else ''))
        print(f'  total {sum(r[0] for r in rows):.1f} h, {len(rows)} players')
    print('\nPlay time is time connected, AFK included. Last death location needs a 1.19+ save.')


# ---------- inventory

def item_name(it):
    n = it['id'].split(':')[-1]
    tag = it.get('tag', {})
    disp = tag.get('display', {}).get('Name')
    if disp:
        n += f' "{text_of(disp)}"'
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


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument('--usercache', help='usercache.json for UUID -> name')
    sub = ap.add_subparsers(dest='cmd', required=True)
    for name in ('hotspots', 'halls', 'portals', 'builds', 'entities', 'blocks', 'maps', 'mapconfig', 'players',
                 'inventory'):
        p = sub.add_parser(name)
        p.add_argument('worlds', nargs='+')
        p.add_argument('--top', type=int, default=8 if name == 'mapconfig' else 15)
        if name == 'hotspots':
            p.add_argument('--min-hours', type=float, help='default: 5%% of the busiest chunk, at least 1')
        if name == 'mapconfig':
            p.add_argument('--zoomin', type=int, default=2)
            p.add_argument('--min-radius', type=int, default=256)
            p.add_argument('--pad', type=int, default=64, help='blocks added around each detected area')
            p.add_argument('--min-hours', type=float, default=24, help='hotspots below this are not ranges')
        if name == 'portals':
            p.add_argument('--min-beds', type=int, default=5)
        if name == 'builds':
            p.add_argument('--json', metavar='DIR', help='also dump every block entity as [kind,x,y,z,label]')
        if name == 'blocks':
            p.add_argument('--min-chunks', type=int, default=2)
        if name == 'inventory':
            p.add_argument('--player', required=True)
    a = ap.parse_args()
    for w in a.worlds:
        if not os.path.exists(os.path.join(w, 'level.dat')):
            sys.exit(f'{w}: no level.dat (pass the save directory)')
    globals()['cmd_' + a.cmd](a, load_names(a.worlds, a.usercache))


if __name__ == '__main__':
    main()
