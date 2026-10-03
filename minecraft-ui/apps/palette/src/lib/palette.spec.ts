import { describe, expect, it } from 'vitest';
import {
	blockPool,
	gradient,
	insertAt,
	hueSortKey,
	labDistance,
	lerpLab,
	lerpLch,
	similar,
	sortByHue,
	type Block,
	type GradientStep,
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
		expect(labDistance(steps[1].target!, [0.5, 0, 0])).toBeCloseTo(0);
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

const chroma = ([, a, b]: Lab) => Math.hypot(a, b);
const hue = ([, a, b]: Lab) => (Math.atan2(b, a) * 180) / Math.PI;
/** OKLCh (lightness, chroma, hue degrees) -> OKLab */
const lch = (l: number, c: number, h: number): Lab => [
	l,
	c * Math.cos((h * Math.PI) / 180),
	c * Math.sin((h * Math.PI) / 180)
];

describe('lerpLch', () => {
	const red = lch(0.5, 0.15, 30);
	const teal = lch(0.7, 0.1, 200);

	it('hits both endpoints', () => {
		expect(labDistance(lerpLch(red, teal, 0), red)).toBeCloseTo(0);
		expect(labDistance(lerpLch(red, teal, 1), teal)).toBeCloseTo(0);
	});

	it('keeps chroma through the middle where a straight line goes gray', () => {
		expect(chroma(lerpLab(red, teal, 0.5))).toBeLessThan(0.03);
		expect(chroma(lerpLch(red, teal, 0.5))).toBeCloseTo(0.125);
		expect(lerpLch(red, teal, 0.5)[0]).toBeCloseTo(0.6);
	});

	it('takes the short way around the hue wheel', () => {
		const mid = lerpLch(lch(0.5, 0.1, 170), lch(0.5, 0.1, -170), 0.5);
		expect(Math.abs(hue(mid))).toBeCloseTo(180);
	});

	it('holds the colored hue when blending to gray', () => {
		const gray = lch(0.9, 0.001, 270);
		const mid = lerpLch(red, gray, 0.5);
		expect(hue(mid)).toBeCloseTo(30);
		expect(chroma(mid)).toBeCloseTo(0.0755, 3);
	});
});

describe('gradient blend', () => {
	const from = block('red', lch(0.5, 0.15, 30));
	const to = block('teal', lch(0.7, 0.1, 200));
	const grayish = block('grayish', [0.6, 0, 0]);
	const yellowGreen = block('yellow-green', lch(0.6, 0.12, 115));
	const pool = [from, to, grayish, yellowGreen];

	it('straight blend picks the gray middle, hue blend the colorful one', () => {
		const middle = (blend: 'straight' | 'hue') =>
			gradient(from, to, pool, { face: 'side', steps: 3, blend })[1].block.id;
		expect(middle('straight')).toBe('grayish');
		expect(middle('hue')).toBe('yellow-green');
	});
});

describe('insertAt', () => {
	const pool = [black, white, ...grays];
	const steps = gradient(black, white, pool, { face: 'side', steps: 3 }); // black, gray5, white
	const ids = (list: GradientStep[] | null) => list?.map((s) => s.block.id);

	it('between two blocks, inserts the closest unused block to their midpoint', () => {
		const result = insertAt(steps, 1, pool, { face: 'side' });
		expect(ids(result)).toEqual(['black', 'gray2', 'gray5', 'white']);
		expect(labDistance(result![1].target!, [0.25, 0, 0])).toBeCloseTo(0);
	});

	it('at the ends, continues the gradient one more step', () => {
		const mids = [grays[1], grays[2]].map((block) => ({ block })); // gray4, gray5
		expect(ids(insertAt(mids, 2, pool, { face: 'side' }))).toEqual(['gray4', 'gray5', 'gray6']);
		expect(ids(insertAt(mids, 0, pool, { face: 'side' }))).toEqual(['gray2', 'gray4', 'gray5']);
	});

	it('clamps extrapolated lightness', () => {
		const result = insertAt(steps, 3, pool, { face: 'side' });
		expect(result![3].target![0]).toBe(1);
		expect(ids(result)).toEqual(['black', 'gray5', 'white', 'gray8']);
	});

	it('does not modify the original list', () => {
		insertAt(steps, 2, pool, { face: 'side' });
		expect(steps).toHaveLength(3);
	});

	it('skips blocks already in the palette', () => {
		const result = insertAt(steps, 2, [black, white, grays[2], grays[3]], { face: 'side' });
		expect(ids(result)).toEqual(['black', 'gray5', 'gray6', 'white']);
	});

	it('returns null when the pool is used up', () => {
		expect(insertAt(steps, 1, [black, white, grays[2]], { face: 'side' })).toBeNull();
	});

	it('uses the hue blend for the midpoint', () => {
		const red = block('red', lch(0.5, 0.15, 30));
		const teal = block('teal', lch(0.7, 0.1, 200));
		const grayish = block('grayish', [0.6, 0, 0]);
		const yellowGreen = block('yellow-green', lch(0.6, 0.12, 115));
		const pair = [{ block: red }, { block: teal }];
		const pool = [red, teal, grayish, yellowGreen];
		expect(insertAt(pair, 1, pool, { face: 'side' })?.[1].block.id).toBe('grayish');
		expect(insertAt(pair, 1, pool, { face: 'side', blend: 'hue' })?.[1].block.id).toBe(
			'yellow-green'
		);
	});
});
