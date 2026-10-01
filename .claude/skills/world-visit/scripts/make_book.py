#!/usr/bin/env python3
"""Build a datapack per world: a written book of clickable /tp links, plus the same links in chat.

  make_book.py <locations.json> --out DIR

locations.json:
  {"<world>": {"version": "1.19", "pages": [
      {"title": "Bases", "places": [
          {"name": "Trading hall", "pos": [x, y, z], "note": "hover text", "dim": "the_nether"}]}]}}

Writes DIR/<world>/oldworlds-book/. In game: /reload, then /function oldworlds:book
(book) or /function oldworlds:places (chat links, work in spectator where books don't open).
Pre-1.20.5 item NBT only (1.16 - 1.20.4).
"""
import argparse, json, os, sys

# data/<ns>/functions/ layout holds through 1.20.x; 1.20.5+ needs a different give syntax anyway
PACK_FORMAT = {'1.16': 6, '1.17': 7, '1.18': 8, '1.18.2': 9, '1.19': 10, '1.19.4': 12,
               '1.20': 15, '1.20.2': 18, '1.20.3': 26, '1.20.4': 26}


def pack_format(v):
    if v in PACK_FORMAT:
        return PACK_FORMAT[v]
    for k in sorted(PACK_FORMAT, key=lambda k: [int(p) for p in k.split('.')], reverse=True):
        if [int(p) for p in v.split('.')] >= [int(p) for p in k.split('.')]:
            return PACK_FORMAT[k]
    sys.exit(f'unsupported version {v}')


def snbt_str(s):
    return "'" + s.replace('\\', '\\\\').replace("'", "\\'") + "'"


def link(p, text, color):
    x, y, z = p['pos']
    dim = p.get('dim', 'overworld')
    hover = f'{x}, {y}, {z}' + (f' ({dim})' if dim != 'overworld' else '') + (f"\n{p['note']}" if p.get('note') else '')
    return {'text': text, 'color': color, 'underlined': True,
            'clickEvent': {'action': 'run_command', 'value': f'/execute in minecraft:{dim} run tp @s {x} {y} {z}'},
            'hoverEvent': {'action': 'show_text', 'contents': hover}}


def mode(label, gm):
    return {'text': f'[{label}]', 'color': 'dark_green',
            'clickEvent': {'action': 'run_command', 'value': f'/gamemode {gm}'},
            'hoverEvent': {'action': 'show_text', 'contents': f'/gamemode {gm}'}}


def build(world, spec, out_root):
    pages = [['', {'text': f'{world}\n', 'bold': True}, {'text': f"{spec['version']}\n\n", 'color': 'dark_gray'},
              'Click a name to teleport. Hover for notes.\n\n', mode('Creative', 'creative'),
              "\n\nF3+N flips to spectator and back (books don't open in spectator). ",
              '/function oldworlds:places lists these in chat.\n\nF1 hides HUD, F2 screenshots.']]
    for pg in spec['pages']:
        for i in range(0, len(pg['places']), 4):  # ~4 places + coords fit on a page
            body = ['', {'text': pg['title'] + '\n\n', 'bold': True}]
            for p in pg['places'][i:i + 4]:
                body += [link(p, p['name'] + '\n', 'dark_blue'),
                         {'text': '%d %d %d\n\n' % tuple(p['pos']), 'color': 'dark_gray', 'underlined': False}]
            pages.append(body)
    title = f'{world} places'[:32]
    give = ('give @s minecraft:written_book{title:%s,author:%s,pages:[%s]}' %
            (snbt_str(title), snbt_str(spec.get('author', 'map')),
             ','.join(snbt_str(json.dumps(p, separators=(',', ':'))) for p in pages)))

    chat = [['', {'text': f'== {world} places ==', 'bold': True, 'color': 'gold'}]]
    for pg in spec['pages']:
        line = ['', {'text': pg['title'] + ': ', 'color': 'gray'}]
        for i, p in enumerate(pg['places']):
            line += ([{'text': ' | ', 'color': 'dark_gray'}] if i else []) + [link(p, p['name'], 'aqua')]
        chat.append(line)

    pack = os.path.join(out_root, world, 'oldworlds-book')
    fn = os.path.join(pack, 'data', 'oldworlds', 'functions')
    os.makedirs(fn, exist_ok=True)
    json.dump({'pack': {'pack_format': pack_format(spec['version']), 'description': f'Places book for {world}'}},
              open(os.path.join(pack, 'pack.mcmeta'), 'w'), indent=2)
    open(os.path.join(fn, 'book.mcfunction'), 'w').write(give + '\n')
    open(os.path.join(fn, 'places.mcfunction'), 'w').write(
        ''.join(f"tellraw @s {json.dumps(l, separators=(',', ':'))}\n" for l in chat))
    print(f'{world}: {len(pages)} pages, {sum(len(p["places"]) for p in spec["pages"])} places -> {pack}')


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('locations')
    ap.add_argument('--out', required=True)
    a = ap.parse_args()
    for world, spec in json.load(open(a.locations)).items():
        build(world, spec, a.out)


if __name__ == '__main__':
    main()
