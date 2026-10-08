import json

import metadata
from extract import Assets


def make_assets(recipes=(), tags=None, blockstates=None):
    """recipes: list of recipe dicts; tags: {"block/logs": [...], "item/x": [...]}"""
    data = {}
    for i, recipe in enumerate(recipes):
        data[f"recipe/r{i}.json"] = json.dumps(recipe).encode()
    for name, values in (tags or {}).items():
        data[f"tags/{name}.json"] = json.dumps({"values": values}).encode()
    files = {
        f"blockstates/{name}.json": json.dumps(state).encode()
        for name, state in (blockstates or {}).items()
    }
    return Assets(files, data)


def shaped(key, result):
    return {"type": "minecraft:crafting_shaped", "key": key, "pattern": ["##"], "result": {"id": result}}


def stonecut(ingredient, result):
    return {"type": "minecraft:stonecutting", "ingredient": ingredient, "result": {"id": result}}


def smelt(ingredient, result):
    return {"type": "minecraft:smelting", "ingredient": ingredient, "result": {"id": result}}


class TestRecipeLinks:
    def test_single_material_recipes_link(self):
        assets = make_assets(
            [
                shaped({"#": "minecraft:stone"}, "minecraft:stone_bricks"),
                stonecut("minecraft:stone", "minecraft:chiseled_stone_bricks"),
                smelt("minecraft:stone_bricks", "minecraft:cracked_stone_bricks"),
            ]
        )
        ids = {"stone", "stone_bricks", "chiseled_stone_bricks", "cracked_stone_bricks"}
        links, _ = metadata.recipe_links(assets, ids)
        assert sorted(links) == [
            ("stone", "chiseled_stone_bricks"),
            ("stone", "stone_bricks"),
            ("stone_bricks", "cracked_stone_bricks"),
        ]

    def test_wood_log_tag_links_but_category_tags_dont(self):
        assets = make_assets(
            [
                {
                    "type": "minecraft:crafting_shapeless",
                    "ingredients": ["#minecraft:spruce_logs"],
                    "result": {"id": "minecraft:spruce_planks"},
                },
                # Any planks -> crafting table must not merge every wood
                shaped({"#": "#minecraft:planks"}, "minecraft:crafting_table"),
            ],
            tags={
                "item/spruce_logs": ["minecraft:spruce_log", "minecraft:stripped_spruce_log"],
                "item/planks": ["minecraft:spruce_planks", "minecraft:oak_planks"],
            },
        )
        ids = {"spruce_log", "stripped_spruce_log", "spruce_planks", "oak_planks", "crafting_table"}
        links, _ = metadata.recipe_links(assets, ids)
        assert sorted(links) == [
            ("spruce_log", "spruce_planks"),
            ("stripped_spruce_log", "spruce_planks"),
        ]

    def test_multi_ingredient_and_glass_dont_link(self):
        assets = make_assets(
            [
                shaped({"#": "minecraft:cobblestone", "V": "minecraft:vine"}, "minecraft:mossy_cobblestone"),
                smelt("minecraft:sand", "minecraft:glass"),
            ]
        )
        links, _ = metadata.recipe_links(assets, {"cobblestone", "mossy_cobblestone", "sand", "glass"})
        assert links == []

    def test_shapes(self):
        assets = make_assets(
            [
                shaped({"#": "minecraft:stone_bricks"}, "minecraft:stone_brick_stairs"),
                stonecut("minecraft:stone_bricks", "minecraft:stone_brick_slab"),
                shaped({"W": "minecraft:oak_planks", "#": "minecraft:stick"}, "minecraft:oak_fence"),
                shaped({"#": "minecraft:white_wool"}, "minecraft:white_carpet"),
            ]
        )
        _, shapes = metadata.recipe_links(assets, {"stone_bricks", "oak_planks", "white_wool"})
        assert shapes == {
            "stone_bricks": {"stairs", "slab"},
            "oak_planks": {"fence"},
            "white_wool": {"carpet"},
        }


class TestNameLinks:
    def test_leaves_join_their_wood(self):
        ids = {"oak_leaves", "oak_log", "orange_poplar_leaves", "poplar_log", "crimson_stem", "azalea_leaves"}
        links = metadata.name_links(ids)
        assert ("oak_leaves", "oak_log") in links
        assert ("orange_poplar_leaves", "poplar_log") in links
        assert not any(a == "azalea_leaves" for a, _ in links)  # no azalea log

    def test_powder_and_mossy(self):
        ids = {"red_concrete", "red_concrete_powder", "cobblestone", "mossy_cobblestone"}
        links = metadata.name_links(ids)
        assert ("red_concrete_powder", "red_concrete") in links
        assert ("mossy_cobblestone", "cobblestone") in links


class TestFamilies:
    def test_representative_prefers_planks_then_shortest(self):
        ids = {"spruce_log", "spruce_planks", "stripped_spruce_log", "stone", "stone_bricks", "calcite"}
        links = [
            ("spruce_log", "spruce_planks"),
            ("stripped_spruce_log", "spruce_planks"),
            ("stone", "stone_bricks"),
        ]
        rep = metadata.families(ids, links)
        assert rep["stripped_spruce_log"] == "spruce_planks"
        assert rep["stone_bricks"] == "stone"
        assert rep["calcite"] == "calcite"


class TestClassification:
    def test_material(self):
        assets = make_assets(
            tags={
                "block/leaves": ["minecraft:oak_leaves"],
                "block/logs": ["minecraft:oak_log"],
                "block/mineable/pickaxe": ["minecraft:stone", "minecraft:red_terracotta"],
                "block/mineable/shovel": ["minecraft:dirt"],
            }
        )
        expected = {
            "white_stained_glass": "glass",
            "ochre_froglight": "light",
            "copper_bulb": "light",
            "red_terracotta": "dyed",
            "red_wool": "dyed",
            "oak_leaves": "plant",
            "oak_log": "wood",
            "iron_ore": "mineral",
            "dirt": "earth",
            "stone": "stone",
            "bedrock": "other",
        }
        assert {b: metadata.material(assets, b) for b in expected} == expected

    def test_feature(self):
        assets = make_assets(
            tags={"block/ores": ["minecraft:iron_ore"]},
            blockstates={
                "furnace": {"variants": {"facing=north,lit=false": {}, "facing=north,lit=true": {}}},
                "redstone_lamp": {"variants": {"lit=false": {}, "lit=true": {}}},
                "stone": {"variants": {"": {}}},
            },
        )
        assert metadata.is_feature(assets, "iron_ore", "iron_ore", "mineral")
        assert metadata.is_feature(assets, "loom", "loom", "wood")
        assert metadata.is_feature(assets, "furnace", "furnace", "stone")
        assert not metadata.is_feature(assets, "redstone_lamp", "redstone_lamp", "light")
        assert not metadata.is_feature(assets, "stone", "stone", "stone")


class TestAnnotate:
    def test_fields(self):
        assets = make_assets(
            [stonecut("minecraft:copper_block", "minecraft:cut_copper")],
            tags={"block/mineable/hoe": ["minecraft:orange_poplar_leaves"]},
        )
        blocks = [
            {"id": "copper_block"},
            {"id": "cut_copper"},
            {"id": "copper_bulb"},
            {"id": "copper_bulb_lit", "variant_of": "copper_bulb"},
            {"id": "red_wool"},
            {"id": "orange_poplar_leaves"},
        ]
        metadata.annotate(assets, blocks)
        by_id = {b["id"]: b for b in blocks}
        assert by_id["copper_block"]["family"] == "cut_copper"
        assert by_id["copper_bulb_lit"]["family"] == by_id["copper_bulb"]["family"]
        assert by_id["copper_bulb_lit"]["material"] == "light"
        assert by_id["red_wool"]["dye"] == "red"
        assert "dye" not in by_id["orange_poplar_leaves"]  # leaves aren't dyed
        assert "shapes" not in by_id["red_wool"]
