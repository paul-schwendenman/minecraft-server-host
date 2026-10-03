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
}

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
	/** The ideal color for this step, for comparison against the block's actual color */
	target: Lab;
}

export interface GradientOptions {
	face: Face;
	steps: number;
	textureWeight?: number;
}

/**
 * Blend from `from` to `to` in `steps` blocks (endpoints included), picking the
 * closest unused block to each evenly spaced point on the OKLab line between them.
 */
export function gradient(
	from: Block,
	to: Block,
	pool: Block[],
	options: GradientOptions
): GradientStep[] {
	const { face, textureWeight = 0 } = options;
	const steps = Math.max(2, Math.round(options.steps));
	const a = from.faces[face];
	const b = to.faces[face];
	const used = new Set([from.id, to.id]);

	const result: GradientStep[] = [{ block: from, target: a.oklab }];
	for (let i = 1; i < steps - 1; i++) {
		const t = i / (steps - 1);
		const target = lerpLab(a.oklab, b.oklab, t);
		const targetNoise = a.noise + (b.noise - a.noise) * t;
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
		if (!best) break; // pool exhausted
		used.add(best.id);
		result.push({ block: best, target });
	}
	result.push({ block: to, target: b.oklab });
	return result;
}
