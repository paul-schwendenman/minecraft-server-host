# block-palette

Extracts full-cube block textures and colors from a Minecraft client jar, as
data for a block palette generator.

```bash
cd scripts/block-palette
uv run extract.py --version 26.3      # -> <repo>/dist/block-palette/26.3
uv run extract.py --version 26.3 --out /tmp/palette
uv run pytest tests/ -v
```

The client jar is downloaded from Mojang's version manifest and cached in
`~/.cache/block-palette/<version>/`. (Server jars don't contain textures.)

## Output

`blocks.json`:

```jsonc
{
  "version": "26.3",
  "blocks": [
    {
      "id": "oak_log",
      "name": "Oak Log",
      "faces": {
        "top":    { "texture": "oak_log-top.png", "hex": "#9d7e4c", "oklab": [L, a, b], "noise": 0.08, "coverage": 1.0 },
        "side":   { ... },   // stats averaged over all four sides; texture is the north (front) face
        "bottom": { ... }
      },
      "overall": { ... },    // all six faces
      "tinted": false,       // biome tint applied (plains defaults)
      "animated": false,     // first frame used
      "translucent": false,  // force_translucent (glass); see coverage for cutouts like leaves
      "same_as": "...",      // present when visually identical to another block (waxed, infested)
      "variant_of": "...",   // extra state records, e.g. copper_bulb_lit -> copper_bulb
      "family": "oak_planks",// representative of its material set (see below)
      "material": "wood",    // wood|stone|earth|dyed|plant|mineral|glass|light|other
      "shapes": ["stairs", "slab", "fence"],  // shaped blocks it crafts into, if any
      "dye": "red",          // dyed blocks and stained glass only
      "feature": true        // ores, machines, workstations: use sparingly
    }
  ]
}
```

- `hex`/`oklab`: mean color, averaged in linear light.
- `noise`: RMS OKLab distance of pixels from the mean — ~0 for concrete, high for ores.
- `coverage`: mean alpha; < 1 for glass, leaves, grates.

`textures/*.png`: rendered faces (tint + overlays composited, first animation
frame), deduplicated by content.

## Builder metadata (`metadata.py`)

Derived from the jar's recipes and tags rather than hand-maintained lists
where possible:

- **family**: blocks linked by single-material recipes (crafting,
  stonecutting, smelting), e.g. stone -> stone bricks -> cracked stone bricks,
  or `#spruce_logs` -> spruce planks. Category tags like `#planks` are ignored
  so a crafting table doesn't merge every wood. Name rules add leaves to their
  wood, concrete powder to concrete, and mossy variants to their base.
- **shapes**: recipes producing `*_stairs`/`_slab`/`_wall`/`_fence`/`_carpet`/`_pane`.
- **material** / **feature**: tags (`logs`, `leaves`, `mineable/*`, `ores`)
  plus short lists for what tags don't cover (light sources, workstations).

Extra records cover states builders treat as their own blocks (lit copper
bulbs and redstone lamp), and the chiseled bookshelf front, which is built
from per-slot models, is rendered from its full texture.

## Limitations

- Only full cubes. Slabs, stairs, walls, etc. reuse their base block's texture.
- Blocks are rendered in one default state (upright, facing north).
