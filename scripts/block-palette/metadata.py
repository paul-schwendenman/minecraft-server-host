"""Builder-facing metadata for extracted blocks, derived from recipes and tags.

Adds to each block record:
    family    id of a representative block in the same set (spruce planks,
              logs, stripped logs and leaves share one); own id if alone
    material  wood | stone | earth | dyed | plant | mineral | glass | light | other
    shapes    stairs/slab/wall/fence/carpet/pane the block can be made into
    dye       dye color of dyed blocks and stained glass
    feature   true for ores, machines and workstations: distinctive blocks a
              palette uses sparingly, if at all
"""

import re

SHAPES = ("stairs", "slab", "wall", "fence", "carpet", "pane")

DYES = (
    "white light_gray gray black brown red orange yellow "
    "lime green cyan light_blue blue purple magenta pink"
).split()

# Recipe types whose ingredient -> result means "made from the same material"
FAMILY_RECIPES = {"crafting_shaped", "crafting_shapeless", "stonecutting", "smelting"}

# Families the recipes don't connect
EXTRA_FAMILY_LINKS = [
    ("nether_wart_block", "crimson_stem"),
    ("warped_wart_block", "warped_stem"),
    ("flowering_azalea_leaves", "azalea_leaves"),
    # Crafted from shards (items), not from each other
    ("prismarine_bricks", "prismarine"),
    ("dark_prismarine", "prismarine"),
    ("purpur_pillar", "purpur_block"),
]

LIGHT_BLOCKS = {
    "glowstone",
    "sea_lantern",
    "shroomlight",
    "ochre_froglight",
    "verdant_froglight",
    "pearlescent_froglight",
    "jack_o_lantern",
    "redstone_lamp",
}

# Mined with an axe but not wood in the building sense
PLANT_BLOCKS = {
    "pumpkin",
    "carved_pumpkin",
    "melon",
    "brown_mushroom_block",
    "red_mushroom_block",
    "mushroom_stem",
    "mangrove_roots",
    "muddy_mangrove_roots",
}

MINERAL_BLOCKS = {
    "coal_block",
    "redstone_block",
    "lapis_block",
    "raw_iron_block",
    "raw_gold_block",
    "raw_copper_block",
}

# Ores and blocks with machine/workstation behavior
FEATURE_BLOCKS = {
    "ancient_debris",
    "barrel",
    "beacon",
    "bee_nest",
    "beehive",
    "budding_amethyst",
    "cartography_table",
    "chiseled_bookshelf",
    "crafting_table",
    "creaking_heart",
    "dried_kelp_block",
    "fletching_table",
    "hay_block",
    "honey_block",
    "jukebox",
    "lodestone",
    "loom",
    "note_block",
    "reinforced_deepslate",
    "respawn_anchor",
    "sculk_catalyst",
    "slime_block",
    "smithing_table",
    "spawner",
    "sponge",
    "suspicious_gravel",
    "suspicious_sand",
    "target",
    "tnt",
    "trial_spawner",
    "vault",
    "wet_sponge",
}
# Blockstate properties that mean the block does something
FEATURE_PROPERTIES = {
    "crafting",
    "enabled",
    "has_record",
    "honey_level",
    "lit",
    "open",
    "orientation",
    "powered",
    "triggered",
}


def _ns(value):
    return value.removeprefix("minecraft:")


def dye_of(block_id):
    for color in sorted(DYES, key=len, reverse=True):
        if block_id.startswith(color + "_"):
            return color
    return None


# --- recipes --------------------------------------------------------------


def _ingredient(assets, value):
    """Resolve one ingredient to (ids, tag name or None)."""
    if isinstance(value, list):
        ids = set()
        for v in value:
            ids |= _ingredient(assets, v)[0]
        return ids, None
    if isinstance(value, dict):  # older {"item": ...} / {"tag": ...} format
        value = value.get("item") or "#" + value.get("tag", "")
    if value.startswith("#"):
        name = _ns(value[1:])
        return assets.tag("item", name), name
    return {_ns(value)}, None


def _ingredients(assets, recipe):
    kind = _ns(recipe["type"])
    if kind == "crafting_shaped":
        values = recipe["key"].values()
    elif kind == "crafting_shapeless":
        values = recipe["ingredients"]
    elif "ingredient" in recipe:
        values = [recipe["ingredient"]]
    else:
        return []
    return [_ingredient(assets, v) for v in values]


def _is_material_tag(name):
    """Tags naming one material's blocks (spruce_logs) rather than a category
    (planks, stone_crafting_materials), which would merge every family."""
    return name is None or name.endswith(("_logs", "_stems")) or name == "bamboo_blocks"


def recipe_links(assets, block_ids):
    """Scan recipes for family links and shapes.

    Returns (links, shapes): links is a list of (ingredient, result) block id
    pairs made from a single material; shapes maps block id -> set of shapes.
    """
    links = []
    shapes = {}
    for path in assets.data_paths("recipe/"):
        recipe = assets.data_json(path)
        result = recipe.get("result")
        if not isinstance(result, dict) or "id" not in result:
            continue
        out = _ns(result["id"])
        ingredients = _ingredients(assets, recipe)
        if not ingredients:
            continue

        for shape in SHAPES:
            if out.endswith("_" + shape):
                for ids, _ in ingredients:
                    for i in ids & block_ids:
                        shapes.setdefault(i, set()).add(shape)

        kind = _ns(recipe["type"])
        if out not in block_ids or kind not in FAMILY_RECIPES:
            continue
        if kind == "smelting" and "glass" in out:
            continue  # sand -> glass isn't a material family
        distinct = {frozenset(ids) for ids, _ in ingredients}
        if len(distinct) != 1 or not all(_is_material_tag(tag) for _, tag in ingredients):
            continue
        for i in next(iter(distinct)) & block_ids:
            if i != out:
                links.append((i, out))
    return links, shapes


def name_links(block_ids):
    """Family links the recipes miss: leaves to their wood, powder to
    concrete, mossy variants to their base."""
    links = []
    for block_id in block_ids:
        if block_id.startswith("mossy_") and block_id.removeprefix("mossy_") in block_ids:
            links.append((block_id, block_id.removeprefix("mossy_")))
        if block_id.endswith("_leaves"):
            # oak_leaves -> oak; orange_poplar_leaves -> poplar
            tokens = block_id.removesuffix("_leaves").split("_")
            for start in range(len(tokens)):
                wood = "_".join(tokens[start:])
                trunk = next(
                    (f"{wood}_{t}" for t in ("log", "stem") if f"{wood}_{t}" in block_ids),
                    None,
                )
                if trunk:
                    links.append((block_id, trunk))
                    break
        if block_id.endswith("_concrete_powder"):
            concrete = block_id.removesuffix("_powder")
            if concrete in block_ids:
                links.append((block_id, concrete))
    links += [(a, b) for a, b in EXTRA_FAMILY_LINKS if a in block_ids and b in block_ids]
    return links


def families(block_ids, links):
    """Union linked blocks into families. Returns block id -> representative:
    the family's planks if it has them, else its shortest id."""
    parent = {b: b for b in block_ids}

    def find(x):
        while parent[x] != x:
            parent[x] = parent[parent[x]]
            x = parent[x]
        return x

    for a, b in links:
        parent[find(a)] = find(b)

    members = {}
    for b in block_ids:
        members.setdefault(find(b), []).append(b)
    rep = {}
    for group in members.values():
        planks = [b for b in group if b.endswith("_planks")]
        choice = min(planks or group, key=lambda b: (len(b), b))
        for b in group:
            rep[b] = choice
    return rep


# --- classification -------------------------------------------------------


def material(assets, block_id):
    tag = lambda name: block_id in assets.tag("block", name)  # noqa: E731
    if "glass" in block_id:
        return "glass"
    if block_id in LIGHT_BLOCKS or "copper_bulb" in block_id:
        return "light"
    if "terracotta" in block_id or (dye_of(block_id) and re.search(r"wool|concrete", block_id)):
        return "dyed"
    if block_id in PLANT_BLOCKS or tag("leaves") or tag("mineable/hoe"):
        return "plant"
    if tag("logs") or tag("planks") or "bamboo" in block_id:
        return "wood"
    if (
        block_id in MINERAL_BLOCKS
        or tag("beacon_base_blocks")
        or tag("ores")
        or re.search(r"copper|amethyst|_ore$", block_id)
    ):
        return "mineral"
    if tag("mineable/shovel"):
        return "earth"
    if tag("mineable/axe"):
        return "wood"
    if tag("mineable/pickaxe"):
        return "stone"
    return "other"


def _state_properties(assets, blockstate_id):
    path = f"blockstates/{blockstate_id}.json"
    if not assets.exists(path):
        return set()
    props = set()
    for key in assets.read_json(path).get("variants", {}):
        props |= {kv.split("=")[0] for kv in key.split(",") if "=" in kv}
    return props


def is_feature(assets, block_id, blockstate_id, block_material):
    if block_id in FEATURE_BLOCKS or block_id in assets.tag("block", "ores"):
        return True
    if block_material == "light":
        return False  # lamps toggle "lit" but are decorative
    return bool(_state_properties(assets, blockstate_id) & FEATURE_PROPERTIES)


def annotate(assets, blocks):
    """Add family/material/shapes/dye/feature to block records in place."""
    ids = {b["id"] for b in blocks}
    links, shapes = recipe_links(assets, ids)
    links += name_links(ids)
    # Extra states (lit bulbs) belong with the block they're a state of
    links += [(b["id"], b["variant_of"]) for b in blocks if b.get("variant_of") in ids]
    family = families(ids, links)

    for b in blocks:
        state = b.get("variant_of", b["id"])
        b["family"] = family[b["id"]]
        b["material"] = material(assets, state)
        if b["id"] in shapes:
            b["shapes"] = sorted(shapes[b["id"]], key=SHAPES.index)
        if b["material"] in ("dyed", "glass") and (dye := dye_of(b["id"])):
            b["dye"] = dye
        if is_feature(assets, b["id"], state, b["material"]):
            b["feature"] = True
