// Block data produced by scripts/block-palette/extract.py, plus the color
// math for building palettes from it. All distances are in OKLab.

export type Lab = [number, number, number];
export type Face = 'top' | 'side' | 'bottom';

export interface ColorStats {
	hex: string;
	oklab: Lab;
	/** RMS OKLab distance of pixels from the mean: ~0 for concrete, high for ores */
	noise: number;
	/** Mean alpha: 1 for solid blocks, lower for glass, leaves, grates */
	coverage: number;
}

export interface FaceStats extends ColorStats {
	texture: string;
}

export interface Block {
	id: string;
	name: string;
	faces: Record<Face, FaceStats>;
	overall: ColorStats;
	tinted: boolean;
	animated: boolean;
	translucent: boolean;
	same_as?: string;
	/** Extra-state records (e.g. lit copper bulbs): the block they're a state of */
	variant_of?: string;
	/** Representative id of the block's material set (spruce planks, logs, leaves...) */
	family: string;
	material: Material;
	/** Shaped blocks this crafts into */
	shapes?: Shape[];
	dye?: string;
	/** Ores, machines and workstations: distinctive, used sparingly */
	feature?: boolean;
}

export type Material =
	'wood' | 'stone' | 'earth' | 'dyed' | 'plant' | 'mineral' | 'glass' | 'light' | 'other';
export type Shape = 'stairs' | 'slab' | 'wall' | 'fence' | 'carpet' | 'pane';

export interface BlockData {
	version: string;
	blocks: Block[];
}

export interface PoolOptions {
	/** Include glass, leaves, grates and other blocks you can see through */
	seeThrough: boolean;
	/** Include blocks with animated textures (magma, prismarine, ...) */
	animated: boolean;
}

export const SOLID_COVERAGE = 0.99;

export function isSeeThrough(block: Block): boolean {
	return block.translucent || block.overall.coverage < SOLID_COVERAGE;
}

/** Blocks eligible to appear in a palette. Visual duplicates (waxed, infested) are dropped. */
export function blockPool(blocks: Block[], options: PoolOptions): Block[] {
	return blocks.filter(
		(b) =>
			!b.same_as && (options.seeThrough || !isSeeThrough(b)) && (options.animated || !b.animated)
	);
}

export function labDistance(a: Lab, b: Lab): number {
	return Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2]);
}

export function lerpLab(a: Lab, b: Lab, t: number): Lab {
	return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t];
}

/** Below this OKLCh chroma a color is effectively gray and its hue is meaningless */
const ACHROMATIC = 0.02;

/**
 * Interpolate in OKLCh: lightness and chroma linearly, hue the short way
 * around the wheel. Keeps saturation through the middle instead of passing
 * through gray like a straight OKLab line does between distant hues.
 */
export function lerpLch(a: Lab, b: Lab, t: number): Lab {
	const ca = Math.hypot(a[1], a[2]);
	const cb = Math.hypot(b[1], b[2]);
	let ha = Math.atan2(a[2], a[1]);
	let hb = Math.atan2(b[2], b[1]);
	// Blending to/from gray: hold the colored end's hue and just fade chroma
	if (ca < ACHROMATIC) ha = hb;
	if (cb < ACHROMATIC) hb = ha;
	let dh = hb - ha;
	if (dh > Math.PI) dh -= 2 * Math.PI;
	else if (dh < -Math.PI) dh += 2 * Math.PI;
	const l = a[0] + (b[0] - a[0]) * t;
	const c = Math.max(0, ca + (cb - ca) * t); // t outside 0-1 extrapolates
	const h = ha + dh * t;
	return [l, c * Math.cos(h), c * Math.sin(h)];
}

export function labToCss([l, a, b]: Lab): string {
	return `oklab(${l} ${a} ${b})`;
}

/** Hue-ordered sort key: grays first (by lightness), then by hue angle, then lightness. */
export function hueSortKey(lab: Lab): [number, number, number] {
	const [l, a, b] = lab;
	const chroma = Math.hypot(a, b);
	if (chroma < 0.03) return [0, 0, l];
	const hue = (Math.atan2(b, a) * 180) / Math.PI;
	return [1, Math.round(((hue + 360 + 30) % 360) / 15), l];
}

export function sortByHue(blocks: Block[], face: Face): Block[] {
	const keyed = blocks.map((b) => ({ b, k: hueSortKey(b.faces[face].oklab) }));
	keyed.sort((x, y) => x.k[0] - y.k[0] || x.k[1] - y.k[1] || x.k[2] - y.k[2]);
	return keyed.map((x) => x.b);
}

/**
 * Cost of using `block` where `target` color is wanted.
 * `textureWeight` adds a penalty for differing busyness from `targetNoise`,
 * so smooth blocks pair with smooth blocks.
 */
function cost(
	block: Block,
	face: Face,
	target: Lab,
	targetNoise: number,
	textureWeight: number
): number {
	const stats = block.faces[face];
	return labDistance(stats.oklab, target) + textureWeight * Math.abs(stats.noise - targetNoise);
}

export interface SimilarOptions {
	face: Face;
	count: number;
	textureWeight?: number;
}

/** The `count` blocks closest in color to `seed`, nearest first (seed excluded). */
export function similar(seed: Block, pool: Block[], options: SimilarOptions): Block[] {
	const { face, count, textureWeight = 0 } = options;
	const target = seed.faces[face];
	return pool
		.filter((b) => b.id !== seed.id)
		.map((b) => ({ b, c: cost(b, face, target.oklab, target.noise, textureWeight) }))
		.sort((x, y) => x.c - y.c)
		.slice(0, count)
		.map((x) => x.b);
}

export interface GradientStep {
	block: Block;
	/**
	 * The ideal color for this step, for comparison against the block's actual
	 * color. Absent for blocks restored from a shared list.
	 */
	target?: Lab;
}

/** `straight`: OKLab line (can pass through gray); `hue`: around the OKLCh hue wheel */
export type Blend = 'straight' | 'hue';

export interface BlendOptions {
	face: Face;
	textureWeight?: number;
	blend?: Blend;
}

export interface GradientOptions extends BlendOptions {
	steps: number;
}

/** Cheapest block in `pool` for a target color, skipping ids in `used`. */
function nearestUnused(
	pool: Block[],
	used: Set<string>,
	face: Face,
	target: Lab,
	targetNoise: number,
	textureWeight: number
): Block | undefined {
	let best: Block | undefined;
	let bestCost = Infinity;
	for (const block of pool) {
		if (used.has(block.id)) continue;
		const c = cost(block, face, target, targetNoise, textureWeight);
		if (c < bestCost) {
			best = block;
			bestCost = c;
		}
	}
	return best;
}

/**
 * Blend from `from` to `to` in `steps` blocks (endpoints included), picking the
 * closest unused block to each evenly spaced color between them (see `Blend`).
 */
export function gradient(
	from: Block,
	to: Block,
	pool: Block[],
	options: GradientOptions
): GradientStep[] {
	const { face, textureWeight = 0, blend = 'straight' } = options;
	const lerp = blend === 'hue' ? lerpLch : lerpLab;
	const steps = Math.max(2, Math.round(options.steps));
	const a = from.faces[face];
	const b = to.faces[face];
	const used = new Set([from.id, to.id]);

	const result: GradientStep[] = [{ block: from, target: a.oklab }];
	for (let i = 1; i < steps - 1; i++) {
		const t = i / (steps - 1);
		const target = lerp(a.oklab, b.oklab, t);
		const targetNoise = a.noise + (b.noise - a.noise) * t;
		const best = nearestUnused(pool, used, face, target, targetNoise, textureWeight);
		if (!best) break; // pool exhausted
		used.add(best.id);
		result.push({ block: best, target });
	}
	result.push({ block: to, target: b.oklab });
	return result;
}

/**
 * Insert a block at `position` (0 = before the first, `steps.length` = after
 * the last) and return the new list, or null if every pool block is in use.
 *
 * Between two blocks it targets their blend midpoint. At either end it
 * continues the gradient one step further, extrapolating from the two
 * outermost blocks.
 */
export function insertAt(
	steps: GradientStep[],
	position: number,
	pool: Block[],
	options: BlendOptions
): GradientStep[] | null {
	const { face, textureWeight = 0, blend = 'straight' } = options;
	if (steps.length < 2) return null;
	const lerp = blend === 'hue' ? lerpLch : lerpLab;
	const color = (i: number) => steps[i].block.faces[face];
	const n = steps.length;

	// Blend from `a` to `b` by `t`: 0.5 for a midpoint, 2 to step past `b`
	const [a, b, t] =
		position <= 0
			? [color(1), color(0), 2]
			: position >= n
				? [color(n - 2), color(n - 1), 2]
				: [color(position - 1), color(position), 0.5];
	const [l, la, lb] = lerp(a.oklab, b.oklab, t);
	const target: Lab = [Math.min(1, Math.max(0, l)), la, lb];
	const noise = t === 0.5 ? (a.noise + b.noise) / 2 : b.noise;

	const used = new Set(steps.map((s) => s.block.id));
	const block = nearestUnused(pool, used, face, target, noise, textureWeight);
	if (!block) return null;
	const at = Math.min(n, Math.max(0, position));
	return [...steps.slice(0, at), { block, target }, ...steps.slice(at)];
}
