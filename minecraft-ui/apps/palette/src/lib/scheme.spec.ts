import { describe, expect, it } from 'vitest';
import type { Block, Material } from './palette';
import { extendGroup, PRESETS, scheme, seededRandom, type Preset } from './scheme';

interface Spec {
	l: number;
	c?: number;
	h?: number;
	noise?: number;
	material?: Material;
	family?: string;
	texture?: string;
	coverage?: number;
	translucent?: boolean;
	feature?: boolean;
	shapes?: Block['shapes'];
}

function block(id: string, spec: Spec): Block {
	const { l, c = 0, h = 0, noise = 0.05, coverage = 1 } = spec;
	const r = (h * Math.PI) / 180;
	const stats = {
		texture: spec.texture ?? `${id}.png`,
		hex: '#000000',
		oklab: [l, c * Math.cos(r), c * Math.sin(r)] as [number, number, number],
		noise,
		coverage
	};
	return {
		id,
		name: id,
		faces: { top: stats, side: stats, bottom: stats },
		overall: stats,
		tinted: false,
		animated: false,
		translucent: spec.translucent ?? false,
		family: spec.family ?? id,
		material: spec.material ?? 'stone',
		shapes: spec.shapes ?? ['stairs', 'slab'],
		feature: spec.feature
	};
}

const spruce = { family: 'spruce_planks', material: 'wood' as const, h: 65 };
const pool: Block[] = [
	block('spruce_planks', { ...spruce, l: 0.48, c: 0.07, noise: 0.05 }),
	block('stripped_spruce_log', { ...spruce, l: 0.5, c: 0.07, noise: 0.03 }),
	block('spruce_log', { ...spruce, l: 0.3, c: 0.04, noise: 0.08, texture: 'spruce_bark.png' }),
	block('spruce_wood', { ...spruce, l: 0.3, c: 0.04, noise: 0.08, texture: 'spruce_bark.png' }),
	block('spruce_leaves', { ...spruce, l: 0.35, c: 0.06, h: 130, material: 'plant', coverage: 0.6 }),
	block('stone', { family: 'stone', l: 0.6, noise: 0.03 }),
	block('cobblestone', { family: 'stone', l: 0.55, noise: 0.09 }),
	block('stone_bricks', { family: 'stone', l: 0.58, noise: 0.05 }),
	block('deepslate', { family: 'deepslate', l: 0.35, noise: 0.06 }),
	block('deepslate_tiles', { family: 'deepslate', l: 0.33, noise: 0.07 }),
	block('basalt', { l: 0.36, noise: 0.08 }),
	block('calcite', { l: 0.9, c: 0.01, noise: 0.04, shapes: [] }),
	block('white_concrete', { l: 0.92, noise: 0.002, material: 'dyed' }),
	block('birch_planks', { family: 'birch_planks', material: 'wood', l: 0.78, c: 0.07, h: 90 }),
	block('orange_terracotta', { l: 0.55, c: 0.12, h: 50, noise: 0.01, material: 'dyed' }),
	block('red_terracotta', { l: 0.45, c: 0.12, h: 30, noise: 0.01, material: 'dyed' }),
	block('green_wool', { l: 0.5, c: 0.12, h: 135, noise: 0.03, material: 'dyed' }),
	block('lime_terracotta', { l: 0.6, c: 0.1, h: 130, noise: 0.01, material: 'dyed' }),
	block('orange_stained_glass', { l: 0.6, c: 0.14, h: 50, material: 'glass', translucent: true }),
	block('glass', { l: 0.85, material: 'glass', translucent: true, coverage: 0.3 }),
	block('iron_ore', { l: 0.55, c: 0.02, material: 'mineral', feature: true }),
	block('loom', { l: 0.6, c: 0.06, h: 65, material: 'wood', feature: true })
];
const byId = new Map(pool.map((b) => [b.id, b]));
const anchor = byId.get('spruce_planks')!;
const presets = Object.keys(PRESETS) as Preset[];
const flat = (groups: ReturnType<typeof scheme>) => groups.flatMap((g) => g.blocks);

describe('scheme', () => {
	it.each(presets)('%s: produces its roles in order, anchor first', (preset) => {
		const groups = scheme(anchor, pool, { face: 'side', preset });
		expect(groups.map((g) => g.role)).toEqual(PRESETS[preset].map((r) => r.role));
		groups.forEach((g, i) => expect(g.blocks).toHaveLength(PRESETS[preset][i].count));
		expect(groups[0].blocks[0]).toBe(anchor);
	});

	it.each(presets)('%s: no repeated block or face texture, at most one feature', (preset) => {
		for (const seed of [0, 1, 2, 3, 4, 5]) {
			const blocks = flat(scheme(anchor, pool, { face: 'side', preset, seed }));
			expect(new Set(blocks.map((b) => b.id)).size).toBe(blocks.length);
			expect(new Set(blocks.map((b) => b.faces.side.texture)).size).toBe(blocks.length);
			expect(blocks.filter((b) => b.feature).length).toBeLessThanOrEqual(1);
		}
	});

	it("fills main with the anchor's own set, skipping same-texture duplicates", () => {
		const [main] = scheme(anchor, pool, { face: 'side', preset: 'earthy' });
		expect(main.blocks.map((b) => b.id).sort()).toEqual(
			['spruce_log', 'spruce_planks', 'stripped_spruce_log'].sort()
		);
	});

	it('pairs a wood main with a stone secondary from one family', () => {
		const groups = scheme(anchor, pool, { face: 'side', preset: 'earthy' });
		const secondary = groups.find((g) => g.role === 'secondary')!.blocks;
		expect(secondary.every((b) => b.material === 'stone')).toBe(true);
		expect(new Set(secondary.map((b) => b.family)).size).toBe(1);
	});

	it('picks a light trim and see-through detail', () => {
		const groups = scheme(anchor, pool, { face: 'side', preset: 'accent' });
		const trim = groups.find((g) => g.role === 'trim')!.blocks[0];
		const detail = groups.find((g) => g.role === 'detail')!.blocks[0];
		expect(trim.faces.side.oklab[0]).toBeGreaterThan(0.75);
		expect(detail.translucent || detail.overall.coverage < 0.99).toBe(true);
	});

	it('accent follows the palette hue, or a pinned block', () => {
		const accentOf = (accent?: Block) =>
			scheme(anchor, pool, { face: 'side', preset: 'accent', accent }).find(
				(g) => g.role === 'accent'
			)!.blocks[0].id;
		expect(accentOf()).toBe('orange_terracotta'); // spruce is orange-brown
		expect(accentOf(byId.get('green_wool'))).toBe('green_wool');
		// Even a near-neutral pin is used as-is
		expect(accentOf(byId.get('calcite'))).toBe('calcite');
	});

	it('tonal stays close to the anchor color', () => {
		const deepslate = byId.get('deepslate')!;
		const tone = scheme(deepslate, pool, { face: 'side', preset: 'tonal' }).find(
			(g) => g.role === 'tone'
		)!.blocks;
		for (const b of tone) expect(Math.abs(b.faces.side.oklab[0] - 0.35)).toBeLessThan(0.2);
	});

	it('is deterministic per seed and varies across seeds', () => {
		const ids = (seed: number) =>
			flat(scheme(anchor, pool, { face: 'side', preset: 'accent', seed }))
				.map((b) => b.id)
				.join();
		expect(ids(7)).toBe(ids(7));
		const variants = new Set([1, 2, 3, 4, 5, 6, 7, 8].map(ids));
		expect(variants.size).toBeGreaterThan(1);
	});
});

describe('extendGroup', () => {
	it("adds the role's next-best unused block to just that group", () => {
		const options = { face: 'side' as const, preset: 'earthy' as const };
		const groups = scheme(anchor, pool, options);
		const index = groups.findIndex((g) => g.role === 'secondary');
		const next = extendGroup(groups, index, anchor, pool, options)!;
		expect(next[index].blocks).toHaveLength(groups[index].blocks.length + 1);
		const added = next[index].blocks.at(-1)!;
		expect(flat(groups).map((b) => b.id)).not.toContain(added.id);
		expect(next.filter((_, i) => i !== index)).toEqual(groups.filter((_, i) => i !== index));
	});

	it('returns null when nothing is left', () => {
		const tiny = [anchor, byId.get('stone')!];
		const options = { face: 'side' as const, preset: 'earthy' as const };
		const groups = scheme(anchor, tiny, options);
		expect(extendGroup(groups, 0, anchor, tiny, options)).toBeNull();
	});
});

describe('seededRandom', () => {
	it('repeats for a seed and stays in [0, 1)', () => {
		const a = seededRandom(42);
		const b = seededRandom(42);
		const xs = Array.from({ length: 50 }, () => a());
		expect(xs).toEqual(Array.from({ length: 50 }, () => b()));
		expect(xs.every((x) => x >= 0 && x < 1)).toBe(true);
	});
});
