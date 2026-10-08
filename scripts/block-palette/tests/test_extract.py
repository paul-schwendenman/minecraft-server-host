import io
import json

import numpy as np
import pytest
from PIL import Image

import extract
from extract import Assets


def png(color, size=(16, 16)):
    buf = io.BytesIO()
    Image.new("RGBA", size, color).save(buf, format="PNG")
    return buf.getvalue()


CUBE_ALL = {
    "textures": {"particle": "#all"},
    "elements": [
        {
            "from": [0, 0, 0],
            "to": [16, 16, 16],
            "faces": {
                d: {"texture": "#all", "cullface": d} for d in extract.DIRECTIONS
            },
        }
    ],
}

SINGLE_FACE = {
    "elements": [
        {
            "from": [0, 0, 0],
            "to": [16, 16, 0],
            "faces": {"north": {"texture": "#texture"}},
        }
    ]
}


def make_assets(blockstates, models, textures):
    files = {}
    for name, data in blockstates.items():
        files[f"blockstates/{name}.json"] = json.dumps(data).encode()
    for name, data in models.items():
        files[f"models/block/{name}.json"] = json.dumps(data).encode()
    for name, data in textures.items():
        files[f"textures/block/{name}.png"] = data
    return Assets(files)


class TestRotateDirection:
    @pytest.mark.parametrize(
        "x,y,expected",
        [
            (0, 0, "north"),
            (0, 90, "east"),
            (0, 180, "south"),
            (0, 270, "west"),
            (90, 0, "down"),
            (270, 0, "up"),
        ],
    )
    def test_north_face(self, x, y, expected):
        # Matches how mushroom blocks rotate one north-face model onto all sides
        assert extract.rotate_direction("north", x, y) == expected

    def test_up_unchanged_by_y(self):
        assert extract.rotate_direction("up", 0, 90) == "up"


class TestResolveTexture:
    def test_follows_references(self):
        textures = {"all": "minecraft:block/stone", "side": "#all"}
        assert extract.resolve_texture(textures, "#side") == ("block/stone", False)

    def test_sprite_object(self):
        textures = {"all": {"sprite": "minecraft:block/glass", "force_translucent": True}}
        assert extract.resolve_texture(textures, "#all") == ("block/glass", True)

    def test_missing_or_cyclic(self):
        assert extract.resolve_texture({}, "#nope") == (None, False)
        assert extract.resolve_texture({"a": "#b", "b": "#a"}, "#a") == (None, False)


class TestDefaultParts:
    def test_prefers_upright_axis(self):
        state = {
            "variants": {
                "axis=x": {"model": "log_horizontal", "x": 90},
                "axis=y": {"model": "log"},
            }
        }
        assert extract.default_parts(state) == [{"model": "log"}]

    def test_prefers_north_facing(self):
        state = {
            "variants": {
                "facing=east,lit=false": {"model": "furnace", "y": 90},
                "facing=north,lit=false": {"model": "furnace"},
            }
        }
        assert extract.default_parts(state) == [{"model": "furnace"}]

    def test_prefer_selects_state(self):
        state = {
            "variants": {
                "lit=false,powered=false": {"model": "bulb"},
                "lit=true,powered=false": {"model": "bulb_lit"},
            }
        }
        assert extract.default_parts(state) == [{"model": "bulb"}]
        assert extract.default_parts(state, frozenset({"lit=true"})) == [{"model": "bulb_lit"}]
        # Unknown preference falls back to the default pick
        assert extract.default_parts(state, frozenset({"nope=1"})) == [{"model": "bulb"}]

    def test_weighted_variant_takes_first(self):
        state = {"variants": {"": [{"model": "a"}, {"model": "b", "y": 90}]}}
        assert extract.default_parts(state) == [{"model": "a"}]

    def test_multipart_picks_true_booleans(self):
        state = {
            "multipart": [
                {"apply": {"model": "outside"}, "when": {"north": "true"}},
                {"apply": {"model": "inside"}, "when": {"north": "false"}},
                {"apply": {"model": "base"}},
            ]
        }
        assert extract.default_parts(state) == [{"model": "outside"}, {"model": "base"}]


class TestCubeFaces:
    def test_cube_all(self):
        assets = make_assets(
            {},
            {"cube_all": CUBE_ALL, "stone": {"parent": "block/cube_all", "textures": {"all": "block/stone"}}},
            {},
        )
        faces = extract.cube_faces(assets, [{"model": "block/stone"}])
        assert set(faces) == set(extract.DIRECTIONS)
        assert faces["up"] == [
            {"sprite": "block/stone", "tinted": False, "force_translucent": False}
        ]

    def test_rotated_single_faces_make_a_cube(self):
        models = {
            "single_face": SINGLE_FACE,
            "mushroom": {"parent": "block/single_face", "textures": {"texture": "block/cap"}},
        }
        rotations = [(0, 0), (0, 90), (0, 180), (0, 270), (270, 0), (90, 0)]
        parts = [{"model": "block/mushroom", "x": x, "y": y} for x, y in rotations]
        faces = extract.cube_faces(make_assets({}, models, {}), parts)
        assert faces is not None
        assert all(layers[0]["sprite"] == "block/cap" for layers in faces.values())

    def test_partial_block_rejected(self):
        slab = {
            "textures": {"all": "block/stone"},
            "elements": [
                {
                    "from": [0, 0, 0],
                    "to": [16, 8, 16],
                    "faces": {d: {"texture": "#all"} for d in extract.DIRECTIONS},
                }
            ],
        }
        assets = make_assets({}, {"slab": slab}, {})
        assert extract.cube_faces(assets, [{"model": "block/slab"}]) is None

    def test_builtin_model_rejected(self):
        assets = make_assets({}, {"chest": {"textures": {"particle": "block/oak_planks"}}}, {})
        assert extract.cube_faces(assets, [{"model": "block/chest"}]) is None


class TestColor:
    def test_oklab_white_and_black(self):
        np.testing.assert_allclose(extract.linear_to_oklab([1, 1, 1]), [1, 0, 0], atol=1e-4)
        np.testing.assert_allclose(extract.linear_to_oklab([0, 0, 0]), [0, 0, 0], atol=1e-4)

    def test_srgb_roundtrip(self):
        c = np.linspace(0, 1, 11)
        np.testing.assert_allclose(extract.linear_to_srgb(extract.srgb_to_linear(c)), c, atol=1e-9)

    def test_flat_texture_stats(self):
        img = np.zeros((16, 16, 4))
        img[..., :3] = np.array([0x12, 0x34, 0x56]) / 255
        img[..., 3] = 1
        stats = extract.color_stats([img])
        assert stats["hex"] == "#123456"
        assert stats["noise"] == 0
        assert stats["coverage"] == 1

    def test_mean_is_linear_light_and_ignores_transparent(self):
        img = np.zeros((2, 2, 4))
        img[0, :, :] = [1, 1, 1, 1]  # white, opaque
        img[1, 0, :] = [0, 0, 0, 1]  # black, opaque
        img[1, 1, :] = [1, 0, 0, 0]  # red, fully transparent
        stats = extract.color_stats([img])
        # 2/3 white in linear light -> sRGB ~0.836 (#d5), not the naive 0.667 (#aa)
        assert stats["hex"] == "#d5d5d5"
        assert stats["coverage"] == 0.75
        assert stats["noise"] > 0


class TestExtract:
    def test_tint_dedupe_and_names(self):
        leaves_model = {
            "textures": {"particle": "#all"},
            "elements": [
                {
                    "from": [0, 0, 0],
                    "to": [16, 16, 16],
                    "faces": {d: {"texture": "#all", "tintindex": 0} for d in extract.DIRECTIONS},
                }
            ],
        }
        assets = make_assets(
            blockstates={
                "stone": {"variants": {"": {"model": "block/stone"}}},
                "infested_stone": {"variants": {"": {"model": "block/stone"}}},
                "oak_leaves": {"variants": {"": {"model": "block/oak_leaves"}}},
                "cherry_leaves": {"variants": {"": {"model": "block/cherry_leaves"}}},
                "barrier": {"variants": {"": {"model": "block/stone"}}},
            },
            models={
                "cube_all": CUBE_ALL,
                "leaves": leaves_model,
                "stone": {"parent": "block/cube_all", "textures": {"all": "block/stone"}},
                "oak_leaves": {"parent": "block/leaves", "textures": {"all": "block/oak_leaves"}},
                "cherry_leaves": {"parent": "block/leaves", "textures": {"all": "block/cherry_leaves"}},
            },
            textures={
                "stone": png((126, 126, 126, 255)),
                "oak_leaves": png((255, 255, 255, 255)),
                "cherry_leaves": png((255, 255, 255, 255)),
            },
        )
        blocks, textures = extract.extract(assets, {"block.minecraft.stone": "Stone"})
        by_id = {b["id"]: b for b in blocks}

        assert "barrier" not in by_id
        assert by_id["stone"]["name"] == "Stone"
        assert by_id["infested_stone"]["same_as"] == "stone"
        assert "same_as" not in by_id["stone"]

        assert by_id["oak_leaves"]["tinted"]
        assert by_id["oak_leaves"]["faces"]["side"]["hex"] == "#77ab2f"
        assert not by_id["cherry_leaves"]["tinted"]
        assert by_id["cherry_leaves"]["faces"]["side"]["hex"] == "#ffffff"

        assert len(textures) == 3  # stone, tinted oak, untinted cherry
