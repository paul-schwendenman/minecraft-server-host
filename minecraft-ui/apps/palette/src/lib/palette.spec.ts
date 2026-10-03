import { describe, expect, it } from 'vitest';
import {
	blockPool,
	gradient,
	hueSortKey,
	labDistance,
	similar,
	sortByHue,
	type Block,
	type Lab
} from './palette';

function block(id: string, oklab: Lab, extra: Partial<Block> & { noise?: number } = {}): Block {
	const { noise = 0, ...rest } = extra;
	const stats = { texture: `${id}.png`, hex: '#000000', oklab, noise, coverage: 1 };
	return {
		id,
		name: id,
		faces: { top: stats, side: stats, bottom: stats },
		overall: stats,
		tinted: false,
		animated: false,
		translucent: false,
		...rest
	};
}

const black = block('black', [0, 0, 0]);
const white = block('white', [1, 0, 0]);
const grays = [0.2, 0.4, 0.5, 0.6, 0.8].map((l) => block(`gray${l * 10}`, [l, 0, 0]));

describe('blockPool', () => {
	const glass = block('glass', [0.9, 0, 0], { translucent: true });
	const leaves = { ...block('leaves', [0.5, -0.1, 0.1]) };
	leaves.overall = { ...leaves.overall, coverage: 0.67 };
	const magma = block('magma', [0.5, 0.1, 0.1], { animated: true });
	const waxed = block('waxed', [0.6, 0.1, 0.1], { same_as: 'copper' });
	const all = [black, glass, leaves, magma, waxed];

	it('drops see-through, animated and duplicate blocks by default', () => {
		const ids = blockPool(all, { seeThrough: false, animated: false }).map((b) => b.id);
		expect(ids).toEqual(['black']);
	});

	it('includes them when asked, but never duplicates', () => {
		const ids = blockPool(all, { seeThrough: true, animated: true }).map((b) => b.id);
		expect(ids).toEqual(['black', 'glass', 'leaves', 'magma']);
	});
});

describe('similar', () => {
	it('returns nearest blocks first, excluding the seed', () => {
		const pool = [black, white, ...grays];
		const ids = similar(grays[2], pool, { face: 'side', count: 3 }).map((b) => b.id);
		expect(ids.slice(0, 2).sort()).toEqual(['gray4', 'gray6']);
		expect(ids).not.toContain('gray5');
		expect(ids).toHaveLength(3);
	});

	it('texture weight prefers matching busyness', () => {
		const seed = block('seed', [0.5, 0, 0], { noise: 0.2 });
		const smooth = block('smooth', [0.51, 0, 0], { noise: 0 });
		const busy = block('busy', [0.53, 0, 0], { noise: 0.2 });
		const opts = { face: 'side' as const, count: 1 };
		expect(similar(seed, [smooth, busy], opts)[0].id).toBe('smooth');
		expect(similar(seed, [smooth, busy], { ...opts, textureWeight: 1 })[0].id).toBe('busy');
	});
});

describe('gradient', () => {
	const pool = [black, white, ...grays];

	it('keeps endpoints and fills evenly spaced steps', () => {
		const ids = gradient(black, white, pool, { face: 'side', steps: 5 }).map((s) => s.block.id);
		expect(ids).toEqual(['black', 'gray2', 'gray5', 'gray8', 'white']);
	});

	it('never repeats a block', () => {
		const steps = gradient(black, white, pool, { face: 'side', steps: 7 });
		const ids = steps.map((s) => s.block.id);
		expect(new Set(ids).size).toBe(ids.length);
		expect(ids).toHaveLength(7);
	});

	it('stops early when the pool runs out', () => {
		const ids = gradient(black, white, [black, white, grays[0]], { face: 'side', steps: 6 }).map(
			(s) => s.block.id
		);
		expect(ids).toEqual(['black', 'gray2', 'white']);
	});

	it('reports ideal target colors along the line', () => {
		const steps = gradient(black, white, pool, { face: 'side', steps: 3 });
		expect(labDistance(steps[1].target, [0.5, 0, 0])).toBeCloseTo(0);
	});
});

describe('sortByHue', () => {
	it('puts grays first by lightness, then colors by hue', () => {
		const red = block('red', [0.6, 0.2, 0.1]);
		const blue = block('blue', [0.5, -0.05, -0.2]);
		const green = block('green', [0.6, -0.15, 0.12]);
		const ids = sortByHue([blue, white, green, red, black], 'side').map((b) => b.id);
		expect(ids).toEqual(['black', 'white', 'red', 'green', 'blue']);
	});

	it('treats low chroma as gray', () => {
		expect(hueSortKey([0.5, 0.01, 0.01])[0]).toBe(0);
	});
});
