"""Extract full-cube block colors and face textures from a Minecraft client jar.

Usage:
    uv run extract.py --version 26.3 [--out DIR]

Writes <out>/blocks.json and <out>/textures/*.png (one PNG per distinct
rendered face: tinted, overlays composited, first animation frame).
"""

import argparse
import hashlib
import io
import json
import sys
import urllib.request
import zipfile
from pathlib import Path

import numpy as np
from PIL import Image

MANIFEST_URL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"
CACHE_DIR = Path.home() / ".cache" / "block-palette"
REPO_ROOT = Path(__file__).resolve().parents[2]

DIRECTIONS = ("down", "up", "north", "south", "west", "east")
SIDES = ("north", "south", "west", "east")

# Unit vectors in model space (+y up, -z north)
DIR_VECTORS = {
    "down": (0, -1, 0),
    "up": (0, 1, 0),
    "north": (0, 0, -1),
    "south": (0, 0, 1),
    "west": (-1, 0, 0),
    "east": (1, 0, 0),
}
VECTOR_DIRS = {v: k for k, v in DIR_VECTORS.items()}

# Tints are hardcoded in the game's BlockColors rather than in the assets, so a
# model's tintindex alone doesn't say what color (or whether, e.g. cherry and
# pale oak leaves reuse the tinted leaves model but render untinted).
# Values are the plains-biome defaults.
GRASS_TINT = 0x91BD59
FOLIAGE_TINT = 0x77AB2F
BLOCK_TINTS = {
    "grass_block": GRASS_TINT,
    "oak_leaves": FOLIAGE_TINT,
    "jungle_leaves": FOLIAGE_TINT,
    "acacia_leaves": FOLIAGE_TINT,
    "dark_oak_leaves": FOLIAGE_TINT,
    "mangrove_leaves": 0x92C648,
    "birch_leaves": 0x80A755,
    "spruce_leaves": 0x619961,
    "cherry_leaves": None,
    "pale_oak_leaves": None,
}

# Full cubes that aren't obtainable building blocks
TECHNICAL_BLOCKS = {
    "barrier",
    "command_block",
    "chain_command_block",
    "repeating_command_block",
    "structure_block",
    "jigsaw",
    "light",
    "test_block",
    "test_instance_block",
}


# --- jar download ---------------------------------------------------------


def _fetch_json(url):
    with urllib.request.urlopen(url) as resp:
        return json.load(resp)


def fetch_client_jar(version, cache_dir=CACHE_DIR):
    """Download (or reuse a cached) client jar for a version, verifying sha1."""
    jar_path = cache_dir / version / "client.jar"
    manifest = _fetch_json(MANIFEST_URL)
    entry = next((v for v in manifest["versions"] if v["id"] == version), None)
    if entry is None:
        raise SystemExit(f"unknown version {version!r}")
    client = _fetch_json(entry["url"])["downloads"]["client"]

    if jar_path.exists() and _sha1(jar_path) == client["sha1"]:
        return jar_path

    print(f"downloading client jar for {version}...", file=sys.stderr)
    jar_path.parent.mkdir(parents=True, exist_ok=True)
    tmp = jar_path.with_suffix(".tmp")
    urllib.request.urlretrieve(client["url"], tmp)
    if _sha1(tmp) != client["sha1"]:
        tmp.unlink()
        raise SystemExit("client jar sha1 mismatch")
    tmp.rename(jar_path)
    return jar_path


def _sha1(path):
    h = hashlib.sha1()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


# --- asset access ---------------------------------------------------------


def _strip_ns(name):
    return name.removeprefix("minecraft:")


class Assets:
    """Read-only view of assets/minecraft inside a client jar (or a dict, for tests)."""

    def __init__(self, files):
        self._files = files
        self._models = {}

    @classmethod
    def from_jar(cls, path):
        zf = zipfile.ZipFile(path)
        prefix = "assets/minecraft/"
        files = _LazyZip(zf, prefix)
        return cls(files)

    def read_json(self, path):
        return json.loads(self._files[path])

    def exists(self, path):
        return path in self._files

    def blockstate_ids(self):
        return sorted(
            p.removeprefix("blockstates/").removesuffix(".json")
            for p in self._files
            if p.startswith("blockstates/") and p.endswith(".json")
        )

    def model(self, name):
        """Resolve a model's parent chain into (textures, elements)."""
        name = _strip_ns(name)
        if name in self._models:
            return self._models[name]
        data = self.read_json(f"models/{name}.json")
        if "parent" in data:
            parent_textures, parent_elements = self.model(data["parent"])
        else:
            parent_textures, parent_elements = {}, None
        textures = {**parent_textures, **data.get("textures", {})}
        elements = data.get("elements", parent_elements)
        self._models[name] = (textures, elements)
        return textures, elements

    def sprite(self, name):
        """Load a block texture as float RGBA in [0, 1], first animation frame only."""
        raw = self._files[f"textures/{_strip_ns(name)}.png"]
        img = Image.open(io.BytesIO(raw)).convert("RGBA")
        w, h = img.size
        if h > w:  # animated strip: frames stacked vertically
            img = img.crop((0, 0, w, w))
        return np.asarray(img, dtype=np.float64) / 255.0

    def is_animated(self, name):
        meta = f"textures/{_strip_ns(name)}.png.mcmeta"
        return self.exists(meta) and "animation" in self.read_json(meta)


class _LazyZip:
    """Mapping of path -> bytes over a zip subdirectory, read on demand."""

    def __init__(self, zf, prefix):
        self._zf = zf
        self._prefix = prefix
        self._names = {
            n.removeprefix(prefix) for n in zf.namelist() if n.startswith(prefix)
        }

    def __getitem__(self, path):
        return self._zf.read(self._prefix + path)

    def __contains__(self, path):
        return path in self._names

    def __iter__(self):
        return iter(self._names)


def resolve_texture(textures, ref):
    """Follow '#var' references to a sprite. Returns (sprite, force_translucent).

    Newer versions allow {"sprite": ..., "force_translucent": true} objects in
    place of plain strings.
    """
    seen = set()
    while True:
        if isinstance(ref, dict):
            return _strip_ns(ref["sprite"]), bool(ref.get("force_translucent"))
        if not ref.startswith("#"):
            return _strip_ns(ref), False
        var = ref[1:]
        if var in seen or var not in textures:
            return None, False
        seen.add(var)
        ref = textures[var]


# --- blockstate -> faces --------------------------------------------------


def rotate_direction(direction, x=0, y=0):
    """Apply a blockstate x then y rotation (degrees, clockwise) to a face direction."""
    vx, vy, vz = DIR_VECTORS[direction]
    for _ in range((x // 90) % 4):  # north -> down -> south -> up
        vy, vz = vz, -vy
    for _ in range((y // 90) % 4):  # north -> east -> south -> west
        vx, vz = -vz, vx
    return VECTOR_DIRS[(vx, vy, vz)]


def default_parts(blockstate):
    """Pick the models rendered for a block's default-looking state.

    Returns a list of {"model", "x", "y"} dicts.
    """
    if "variants" in blockstate:
        variants = blockstate["variants"]
        # Prefer upright pillars, and front-facing-north directional blocks so
        # the north face (used as the side preview) shows the front
        key = next((k for k in ("", "axis=y") if k in variants), None)
        if key is None:
            key = next(
                (k for k in variants if {"facing=north", "orientation=north_up"} & set(k.split(","))),
                None,
            )
        if key is None:
            key = next(iter(variants))
        choice = variants[key]
        return [choice[0] if isinstance(choice, list) else choice]

    parts = blockstate.get("multipart", [])
    state = _guess_multipart_state(parts)
    applied = []
    for part in parts:
        if "when" not in part or _when_matches(part["when"], state):
            apply = part["apply"]
            applied.append(apply[0] if isinstance(apply, list) else apply)
    return applied


def _guess_multipart_state(parts):
    """Choose one value per property: 'true' for booleans, else the first seen.

    For mushroom blocks this selects the all-faces-outside state.
    """
    values = {}

    def collect(cond):
        for key, val in cond.items():
            if key in ("OR", "AND"):
                for sub in val:
                    collect(sub)
            else:
                values.setdefault(key, [])
                values[key].extend(str(val).split("|"))

    for part in parts:
        if "when" in part:
            collect(part["when"])
    return {k: ("true" if "true" in v else v[0]) for k, v in values.items()}


def _when_matches(cond, state):
    if "OR" in cond:
        return any(_when_matches(c, state) for c in cond["OR"])
    if "AND" in cond:
        return all(_when_matches(c, state) for c in cond["AND"])
    return all(state.get(k) in str(v).split("|") for k, v in cond.items())


def _is_outer_face(element, direction):
    """True if this element face lies on the block boundary and spans it fully."""
    lo, hi = element["from"], element["to"]
    axis = {"west": 0, "east": 0, "down": 1, "up": 1, "north": 2, "south": 2}[
        direction
    ]
    on_boundary = (
        lo[axis] == 0 if direction in ("west", "down", "north") else hi[axis] == 16
    )
    others = [i for i in range(3) if i != axis]
    spans = all(lo[i] == 0 and hi[i] == 16 for i in others)
    return on_boundary and spans


def cube_faces(assets, parts):
    """Collect layered textures for each of the six outer faces.

    Returns {direction: [layer, ...]} where layer is
    {"sprite", "tinted", "force_translucent"}, or None if the block isn't a
    full cube.
    """
    faces = {d: [] for d in DIRECTIONS}
    for part in parts:
        textures, elements = assets.model(part["model"])
        if not elements:
            return None  # builtin/entity-rendered (chests, beds, ...)
        for element in elements:
            if element.get("rotation", {}).get("angle", 0):
                continue
            for local_dir, face in element.get("faces", {}).items():
                if not _is_outer_face(element, local_dir):
                    continue
                sprite, translucent = resolve_texture(textures, face["texture"])
                if sprite is None:
                    continue
                world_dir = rotate_direction(
                    local_dir, part.get("x", 0), part.get("y", 0)
                )
                faces[world_dir].append(
                    {
                        "sprite": sprite,
                        "tinted": "tintindex" in face,
                        "force_translucent": translucent,
                    }
                )
    if not all(faces.values()):
        return None
    return faces


# --- rendering & color ----------------------------------------------------


def render_face(assets, layers, tint=None):
    """Alpha-composite face layers (bottom first) into a float RGBA image."""
    out = None
    for layer in layers:
        img = assets.sprite(layer["sprite"]).copy()
        if layer["tinted"] and tint is not None:
            img[..., :3] *= _hex_to_rgb(tint)
        if out is None:
            out = img
            continue
        if img.shape != out.shape:
            continue  # mismatched overlay size; keep base
        a = img[..., 3:4]
        out_a = a + out[..., 3:4] * (1 - a)
        safe = np.where(out_a == 0, 1, out_a)
        out[..., :3] = (img[..., :3] * a + out[..., :3] * out[..., 3:4] * (1 - a)) / safe
        out[..., 3:4] = out_a
    return out


def _hex_to_rgb(value):
    return np.array([(value >> 16) & 255, (value >> 8) & 255, value & 255]) / 255.0


def srgb_to_linear(c):
    c = np.asarray(c, dtype=np.float64)
    return np.where(c <= 0.04045, c / 12.92, ((c + 0.055) / 1.055) ** 2.4)


def linear_to_srgb(c):
    c = np.clip(np.asarray(c, dtype=np.float64), 0, 1)
    return np.where(c <= 0.0031308, c * 12.92, 1.055 * c ** (1 / 2.4) - 0.055)


def linear_to_oklab(rgb):
    """Linear sRGB (..., 3) -> OKLab (..., 3)."""
    m1 = np.array(
        [
            [0.4122214708, 0.5363325363, 0.0514459929],
            [0.2119034982, 0.6806995451, 0.1073969566],
            [0.0883024619, 0.2817188376, 0.6299787005],
        ]
    )
    m2 = np.array(
        [
            [0.2104542553, 0.7936177850, -0.0040720468],
            [1.9779984951, -2.4285922050, 0.4505937099],
            [0.0259040371, 0.7827717662, -0.8086757660],
        ]
    )
    lms = np.asarray(rgb) @ m1.T
    return np.cbrt(lms) @ m2.T


def color_stats(images):
    """Summarize one or more face images.

    The mean is taken in linear light (how a surface reads from a distance)
    and reported as hex + OKLab. `noise` is the RMS OKLab distance of pixels
    from that mean: ~0 for concrete, high for busy textures like ores.
    `coverage` is mean alpha (1.0 = fully opaque).
    """
    px = np.concatenate([img.reshape(-1, 4) for img in images])
    alpha = px[:, 3]
    coverage = float(alpha.mean())
    weights = alpha if alpha.sum() > 0 else np.ones_like(alpha)
    linear = srgb_to_linear(px[:, :3])
    mean_linear = (linear * weights[:, None]).sum(0) / weights.sum()
    mean_lab = linear_to_oklab(mean_linear)
    pixel_lab = linear_to_oklab(linear)
    dist2 = ((pixel_lab - mean_lab) ** 2).sum(1)
    noise = float(np.sqrt((dist2 * weights).sum() / weights.sum()))
    srgb = np.round(linear_to_srgb(mean_linear) * 255).astype(int)
    return {
        "hex": "#{:02x}{:02x}{:02x}".format(*srgb),
        "oklab": [round(float(v), 4) for v in mean_lab],
        "noise": round(noise, 4),
        "coverage": round(coverage, 3),
    }


def to_png_bytes(img):
    data = np.round(np.clip(img, 0, 1) * 255).astype(np.uint8)
    buf = io.BytesIO()
    Image.fromarray(data, "RGBA").save(buf, format="PNG", optimize=True)
    return buf.getvalue()


# --- main -----------------------------------------------------------------


def extract(assets, lang):
    """Build block records. Returns (blocks, textures) where textures maps
    filename -> PNG bytes."""
    blocks = []
    textures = {}
    texture_by_hash = {}
    canonical_by_faces = {}

    def save_texture(block_id, face, img):
        png = to_png_bytes(img)
        digest = hashlib.sha1(png).hexdigest()
        if digest not in texture_by_hash:
            name = f"{block_id}-{face}.png"
            texture_by_hash[digest] = name
            textures[name] = png
        return texture_by_hash[digest]

    # Shortest id first so e.g. "stone" is canonical over "infested_stone"
    ids = sorted(assets.blockstate_ids(), key=lambda i: (len(i), i))
    for block_id in ids:
        if block_id in TECHNICAL_BLOCKS:
            continue
        parts = default_parts(assets.read_json(f"blockstates/{block_id}.json"))
        if not parts:
            continue
        faces = cube_faces(assets, parts)
        if faces is None:
            continue

        tinted = any(layer["tinted"] for layers in faces.values() for layer in layers)
        tint = BLOCK_TINTS.get(block_id)
        if tinted and block_id not in BLOCK_TINTS:
            print(f"warning: {block_id} has tintindex but no known tint", file=sys.stderr)

        rendered = {d: render_face(assets, faces[d], tint) for d in DIRECTIONS}
        side_imgs = [rendered[d] for d in SIDES]
        sprites = {
            layer["sprite"] for layers in faces.values() for layer in layers
        }

        record = {
            "id": block_id,
            "name": lang.get(f"block.minecraft.{block_id}", block_id),
            "faces": {
                "top": {
                    "texture": save_texture(block_id, "top", rendered["up"]),
                    **color_stats([rendered["up"]]),
                },
                "side": {
                    "texture": save_texture(block_id, "side", rendered["north"]),
                    **color_stats(side_imgs),
                },
                "bottom": {
                    "texture": save_texture(block_id, "bottom", rendered["down"]),
                    **color_stats([rendered["down"]]),
                },
            },
            "overall": color_stats(list(rendered.values())),
            "tinted": tint is not None,
            "animated": any(assets.is_animated(s) for s in sprites),
            "translucent": any(
                layer["force_translucent"]
                for layers in faces.values()
                for layer in layers
            ),
        }

        # Waxed copper, infested stone, etc. render identically to another block
        face_key = tuple(hashlib.sha1(to_png_bytes(rendered[d])).digest() for d in DIRECTIONS)
        if face_key in canonical_by_faces:
            record["same_as"] = canonical_by_faces[face_key]
        else:
            canonical_by_faces[face_key] = block_id

        blocks.append(record)

    blocks.sort(key=lambda b: b["id"])
    return blocks, textures


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--version", required=True, help="Minecraft version, e.g. 26.3")
    parser.add_argument("--jar", type=Path, help="Use a local client jar instead of downloading")
    parser.add_argument("--out", type=Path, help="Output dir (default: <repo>/dist/block-palette/<version>)")
    args = parser.parse_args(argv)

    jar = args.jar or fetch_client_jar(args.version)
    out = args.out or REPO_ROOT / "dist" / "block-palette" / args.version

    assets = Assets.from_jar(jar)
    lang = assets.read_json("lang/en_us.json")
    blocks, textures = extract(assets, lang)

    tex_dir = out / "textures"
    tex_dir.mkdir(parents=True, exist_ok=True)
    for old in tex_dir.glob("*.png"):
        old.unlink()
    for name, png in textures.items():
        (tex_dir / name).write_bytes(png)
    (out / "blocks.json").write_text(
        json.dumps({"version": args.version, "blocks": blocks}, indent=1) + "\n"
    )
    unique = sum(1 for b in blocks if "same_as" not in b)
    print(
        f"wrote {len(blocks)} blocks ({unique} visually unique), "
        f"{len(textures)} textures to {out}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
