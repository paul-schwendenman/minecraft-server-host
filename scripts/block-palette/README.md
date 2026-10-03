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
      "same_as": "..."       // present when visually identical to another block (waxed, infested)
    }
  ]
}
```

- `hex`/`oklab`: mean color, averaged in linear light.
- `noise`: RMS OKLab distance of pixels from the mean — ~0 for concrete, high for ores.
- `coverage`: mean alpha; < 1 for glass, leaves, grates.

`textures/*.png`: rendered faces (tint + overlays composited, first animation
frame), deduplicated by content.

## Limitations

- Only full cubes. Slabs, stairs, walls, etc. reuse their base block's texture.
- Blocks are rendered in one default state (upright, facing north).
- Chiseled bookshelves are skipped (their front is built from per-slot models).
