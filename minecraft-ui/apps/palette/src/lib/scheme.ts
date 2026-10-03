// Scheme palettes: a handful of blocks in roles (main material, secondary
// material, light trim, accent, see-through detail...) rather than a line
// through color space. Role mixes per preset follow what popular community
// palettes do; see scripts/evaluate-schemes.ts for how output is compared
// against a sample of them.

import { labDistance, type Block, type Face, type Lab } from './palette';

export type Preset = 'earthy' | 'accent' | 'tonal' | 'twotone';
export type Role = 'main' | 'secondary' | 'trim' | 'accent' | 'detail' | 'tone';

export const ROLE_LABELS: Record<Role, string> = {
	main: 'Main',
	secondary: 'Secondary',
	trim: 'Trim',
	accent: 'Accent',
	detail: 'Detail',
	tone: 'Tone'
};

export const PRESET_INFO: Record<Preset, { label: string; title: string }> = {
	earthy: { label: 'Earthy', title: 'Main material, a contrasting second material and light trim' },
	accent: {
		label: 'Accent',
		title: 'Neutral base with a touch of color and a glass or leaf detail'
	},
	tonal: { label: 'Tonal', title: 'One color, with the variety coming from texture' },
	twotone: { label: 'Two-tone', title: 'Main material plus a second hue group' }
};

export interface SchemeGroup {
	role: Role;
	blocks: Block[];
}

export const PRESETS: Record<Preset, { role: Role; count: number }[]> = {
	// Wood + stone + light trim, no strong color (#123: spruce set + stone bricks/cobble/andesite)
	earthy: [
		{ role: 'main', count: 3 },
		{ role: 'secondary', count: 2 },
		{ role: 'trim', count: 1 }
	],
	// Neutral base with a little color and a window/leaf detail
	accent: [
		{ role: 'main', count: 2 },
		{ role: 'secondary', count: 1 },
		{ role: 'trim', count: 1 },
		{ role: 'accent', count: 1 },
		{ role: 'detail', count: 1 }
	],
	// One color, varied texture (deepslate tiles/bricks/basalt + gray glass)
	tonal: [
		{ role: 'main', count: 2 },
		{ role: 'tone', count: 3 },
		{ role: 'detail', count: 1 }
	],
	// Main material plus a second hue group
	twotone: [
		{ role: 'main', count: 2 },
		{ role: 'trim', count: 1 },
		{ role: 'accent', count: 2 },
		{ role: 'detail', count: 1 }
	]
};

export interface SchemeOptions {
	face: Face;
	preset: Preset;
	/** Pins the accent color; otherwise it's derived from the main block and seed */
	accent?: Block;
	/** 0 = best pick for every slot; other values choose among close alternatives */
	seed?: number;
}

// --- color helpers ---

interface Lch {
	l: number;
	c: number;
	h: number; // degrees
}

function toLch([l, a, b]: Lab): Lch {
	return { l, c: Math.hypot(a, b), h: ((Math.atan2(b, a) * 180) / Math.PI + 360) % 360 };
}

function fromLch({ l, c, h }: Lch): Lab {
	const r = (h * Math.PI) / 180;
	return [l, c * Math.cos(r), c * Math.sin(r)];
}

const lab = (b: Block, face: Face) => b.faces[face].oklab;
const noise = (b: Block, face: Face) => b.faces[face].noise;

function meanLab(blocks: Block[], face: Face): Lab {
	const sum = blocks.reduce<Lab>(
		(acc, b) => {
			const [l, a, bb] = lab(b, face);
			return [acc[0] + l, acc[1] + a, acc[2] + bb];
		},
		[0, 0, 0]
	);
	return [sum[0] / blocks.length, sum[1] / blocks.length, sum[2] / blocks.length];
}

// Below this chroma a block reads as neutral
const NEUTRAL = 0.05;
// Typical hue of oak/spruce/dark oak planks
const WOOD_HUE = 65;
// Textures busier than this (glazed terracotta, ores) overwhelm large surfaces
const BUSY = 0.12;

// Accent hues for neutral main blocks, weighted toward the warm hues community
// palettes favor (wood, terracotta, brick, cherry/pink)
const NEUTRAL_ACCENT_HUES = [30, 60, 45, 350, 140, 75, 20, 320];

// --- ranking ---

interface Context {
	face: Face;
	anchor: Block;
	accentPin?: Block;
	groups: SchemeGroup[];
	rand: () => number;
	/** Sticky per-scheme choices, drawn once from the seed */
	hueOffset: number;
	fallbackHue: number;
	accentChroma: number;
	familySize: Map<string, number>;
}

function chosen(ctx: Context): Block[] {
	return ctx.groups.flatMap((g) => g.blocks);
}

function groupOf(ctx: Context, role: Role): Block[] {
	return ctx.groups.find((g) => g.role === role)?.blocks ?? [];
}

const isSeeThrough = (b: Block) => b.translucent || b.overall.coverage < 0.99;

/**
 * How well a block works as structure (walls, floors, roofs): refined
 * materials that come in shapes or as part of a set beat raw terrain, ores
 * and storage blocks.
 */
function structurePenalty(b: Block, ctx: Context): number {
	let p = 0;
	if (!b.shapes?.length && (ctx.familySize.get(b.family) ?? 1) < 2) p += 0.05;
	if (b.material === 'earth' || b.material === 'other') p += 0.08;
	if (b.material === 'mineral') p += 0.12;
	return p;
}

/** Penalties every role shares: duplicates, a second feature block, glass/leaves/lights */
function basePenalty(b: Block, ctx: Context, group: Block[], allowDetail = false): number {
	let p = 0;
	const all = chosen(ctx);
	// The same face texture is a duplicate (log vs wood). Same color with a
	// different texture (planks vs stripped log) is a pairing palettes seek out.
	const texture = b.faces[ctx.face].texture;
	if (all.some((c) => c.faces[ctx.face].texture === texture)) p += 1;
	if (b.feature) p += all.some((c) => c.feature) ? 1 : 0.4;
	const detail = isSeeThrough(b) || b.material === 'glass' || b.material === 'light';
	if (detail && !allowDetail) p += 1;
	// Within a group, prefer textures unlike what's already there
	if (group.length) {
		const minNoiseGap = Math.min(
			...group.map((c) => Math.abs(noise(c, ctx.face) - noise(b, ctx.face)))
		);
		p += Math.max(0, 0.04 - minNoiseGap) * 2;
	}
	return p;
}

/** Hue of the most colorful block chosen so far, if any reads as colored */
function paletteHue(ctx: Context): number | null {
	const colored = chosen(ctx)
		.map((b) => toLch(lab(b, ctx.face)))
		.filter((x) => x.c >= NEUTRAL)
		.sort((x, y) => y.c - x.c);
	return colored.length ? colored[0].h : null;
}

/** The palette's accent hue: its own dominant color (saturated), shifted for two-tone */
function accentHue(ctx: Context): number {
	return ((paletteHue(ctx) ?? ctx.fallbackHue) + ctx.hueOffset + 360) % 360;
}

function mainTarget(ctx: Context): Lab {
	const main = groupOf(ctx, 'main');
	return meanLab(main.length ? main : [ctx.anchor], ctx.face);
}

const ROLE_COST: Record<Role, (b: Block, ctx: Context, group: Block[]) => number> = {
	// The anchor's own material set first (planks, log, stripped log), then
	// same-material blocks of a similar color
	main(b, ctx, group) {
		const sameFamily = b.family === ctx.anchor.family;
		const d = labDistance(lab(b, ctx.face), lab(ctx.anchor, ctx.face));
		return (
			(sameFamily ? 0 : 0.12 + d) +
			(b.material === ctx.anchor.material ? 0 : 0.1) +
			(b.material === 'plant' && sameFamily ? 1 : 0) + // leaves belong to detail
			structurePenalty(b, ctx) +
			basePenalty(b, ctx, group)
		);
	},

	// A second material a step lighter or darker than main, complementing its
	// color: warm wood against gray stone, neutral stone against colored wood.
	// Further picks stay in the first pick's family.
	secondary(b, ctx, group) {
		const main = toLch(mainTarget(ctx));
		const towardMiddle = main.l < 0.55 ? 1 : -1;
		const l = main.l + 0.18 * towardMiddle;
		const target =
			main.c < NEUTRAL ? fromLch({ l, c: 0.06, h: WOOD_HUE }) : fromLch({ l, c: 0.02, h: main.h });
		const d = labDistance(lab(b, ctx.face), target);
		const mainMaterial = ctx.anchor.material;
		// The classic pairing: stone with wood, wood with everything else
		const pairs = mainMaterial === 'wood' ? b.material === 'stone' : b.material === 'wood';
		return (
			d +
			(b.material === mainMaterial ? 0.15 : 0) -
			(pairs ? 0.05 : 0) +
			(group.length && b.family !== group[0].family ? 0.12 : 0) +
			(noise(b, ctx.face) > BUSY ? 0.05 : 0) +
			structurePenalty(b, ctx) +
			basePenalty(b, ctx, group)
		);
	},

	// The light block nearly every palette has (calcite, quartz, birch...),
	// tinted toward the palette's color and in a contrasting material
	trim(b, ctx, group) {
		const hue = paletteHue(ctx);
		const target = fromLch({ l: 0.86, c: hue === null ? 0.01 : 0.045, h: hue ?? 0 });
		const { l } = toLch(lab(b, ctx.face));
		return (
			labDistance(lab(b, ctx.face), target) -
			(b.material !== ctx.anchor.material ? 0.04 : 0) +
			(l < 0.75 ? 1 : 0) +
			(noise(b, ctx.face) > BUSY ? 0.05 : 0) +
			(noise(b, ctx.face) < 0.01 ? 0.02 : 0) + // flat concrete is the fallback, not the default
			structurePenalty(b, ctx) +
			basePenalty(b, ctx, group)
		);
	},

	// Restrained color: moderate chroma at the scheme's accent hue
	accent(b, ctx, group) {
		const pin = ctx.accentPin && toLch(lab(ctx.accentPin, ctx.face));
		// Later accent blocks stay in the first one's hue and step lightness, so
		// the group reads as one color rather than two of the same
		const first = group.length ? toLch(lab(group[0], ctx.face)) : null;
		const base = first ?? pin ?? { l: 0.52, c: ctx.accentChroma, h: accentHue(ctx) };
		const step = group.length ? (base.l < 0.55 ? 0.12 : -0.12) * group.length : 0;
		const target = fromLch({ ...base, l: base.l + step });
		const { c } = toLch(lab(b, ctx.face));
		return (
			labDistance(lab(b, ctx.face), target) +
			(c < NEUTRAL && base.c >= NEUTRAL ? 0.3 : 0) +
			(b.material === 'mineral' || b.material === 'earth' || b.material === 'other' ? 0.06 : 0) +
			basePenalty(b, ctx, group)
		);
	},

	// Glass, leaves or a light source tinted toward the accent (or main) color;
	// the main material's own leaves are a natural fit
	detail(b, ctx, group) {
		const detail = isSeeThrough(b) || b.material === 'glass' || b.material === 'light';
		const accent = groupOf(ctx, 'accent');
		const target = accent.length ? meanLab(accent, ctx.face) : mainTarget(ctx);
		const ownLeaves = b.family === ctx.anchor.family;
		return (
			(detail ? 0 : 1) +
			(b.material === 'light' ? 0.05 : 0) + // glass and leaves first
			(b.material === 'plant' && !ownLeaves ? 0.03 : 0) +
			0.7 * labDistance(lab(b, ctx.face), target) -
			(b.family === ctx.anchor.family ? 0.1 : 0) +
			basePenalty(b, ctx, group, true)
		);
	},

	// Same color as the anchor, so the variety comes from texture: the rest of
	// the anchor's set first, then similar blocks of the same material
	tone(b, ctx, group) {
		const d = labDistance(lab(b, ctx.face), lab(ctx.anchor, ctx.face));
		const sameFamily = b.family === ctx.anchor.family;
		const repeatsOther = !sameFamily && chosen(ctx).some((c) => c.family === b.family);
		return (
			d * 1.5 -
			(sameFamily ? 0.04 : 0) +
			(repeatsOther ? 0.05 : 0) +
			(b.material === ctx.anchor.material ? 0 : 0.1) +
			structurePenalty(b, ctx) +
			basePenalty(b, ctx, group)
		);
	}
};

/** Unused pool blocks ranked best-first for a role. */
export function rankForRole(role: Role, pool: Block[], ctx: Context, group: Block[]): Block[] {
	const used = new Set(chosen(ctx).map((b) => b.id));
	return pool
		.filter((b) => !used.has(b.id) && !b.same_as)
		.map((b) => ({ b, cost: ROLE_COST[role](b, ctx, group) }))
		.sort((x, y) => x.cost - y.cost)
		.map((x) => x.b);
}

// --- generation ---

/** Small deterministic PRNG (mulberry32) so a seed reproduces a scheme */
export function seededRandom(seed: number): () => number {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

function makeContext(
	anchor: Block,
	pool: Block[],
	options: SchemeOptions,
	groups: SchemeGroup[]
): Context {
	const seed = options.seed ?? 0;
	const rand = seededRandom(seed);
	// Two-tone: a second hue 60-130° from the palette's own, the spread
	// community two-hue palettes use
	const hueOffset =
		options.preset === 'twotone'
			? (seed === 0 ? 100 : 60 + rand() * 70) * (rand() < 0.5 ? 1 : -1)
			: 0;
	// For all-neutral palettes, which have no hue of their own
	const fallbackHue =
		seed === 0
			? NEUTRAL_ACCENT_HUES[0]
			: NEUTRAL_ACCENT_HUES[Math.floor(rand() * NEUTRAL_ACCENT_HUES.length)];
	const familySize = new Map<string, number>();
	for (const b of pool) familySize.set(b.family, (familySize.get(b.family) ?? 0) + 1);
	return {
		face: options.face,
		anchor,
		accentPin: options.accent,
		groups,
		rand,
		hueOffset,
		fallbackHue,
		// Two hue groups stay a notch less saturated than a single accent
		accentChroma: options.preset === 'twotone' ? 0.085 : 0.11,
		familySize
	};
}

/** Pick for a slot: the best, or (seeded) one of the closest few. */
function pick(ranked: Block[], ctx: Context, seeded: boolean): Block | undefined {
	if (!seeded || ranked.length < 2) return ranked[0];
	const r = ctx.rand();
	return ranked[r < 0.55 ? 0 : r < 0.85 ? 1 : 2] ?? ranked[0];
}

export function scheme(anchor: Block, pool: Block[], options: SchemeOptions): SchemeGroup[] {
	const groups: SchemeGroup[] = [];
	const ctx = makeContext(anchor, pool, options, groups);
	const seeded = (options.seed ?? 0) !== 0;
	for (const { role, count } of PRESETS[options.preset]) {
		const group: SchemeGroup = { role, blocks: [] };
		groups.push(group);
		for (let i = 0; i < count; i++) {
			if (role === 'main' && i === 0) {
				group.blocks.push(anchor);
				continue;
			}
			// A pinned accent is the accent, unless it's already in the palette
			const pin = options.accent;
			if (role === 'accent' && i === 0 && pin && !chosen(ctx).some((b) => b.id === pin.id)) {
				group.blocks.push(pin);
				continue;
			}
			const block = pick(rankForRole(role, pool, ctx, group.blocks), ctx, seeded);
			if (block) group.blocks.push(block);
		}
	}
	return groups.filter((g) => g.blocks.length);
}

/** Add the next-best block for a group's role. Returns null if none is left. */
export function extendGroup(
	groups: SchemeGroup[],
	index: number,
	anchor: Block,
	pool: Block[],
	options: SchemeOptions
): SchemeGroup[] | null {
	const ctx = makeContext(anchor, pool, options, groups);
	const group = groups[index];
	const [block] = rankForRole(group.role, pool, ctx, group.blocks);
	if (!block) return null;
	return groups.map((g, i) => (i === index ? { ...g, blocks: [...g.blocks, block] } : g));
}
