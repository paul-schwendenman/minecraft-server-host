// Compare generated scheme palettes against a sample of community palettes.
//
//   npx vite-node scripts/evaluate-schemes.ts <sample-dir> [anchor ...]
//
// <sample-dir> holds JSON pages from blockpalettes.com's palette listing API
// (`{"palettes": [{"blockOne": ..., ..., "blockSix": ...}]}`). The sample isn't
// checked in. Each sampled palette's first block is used as an anchor for
// every preset, and the structural stats of the output are compared with the
// sample's palettes of the matching kind, so presets are judged against the
// whole distribution rather than tuned to reproduce particular palettes.

import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { labDistance, type Block, type BlockData } from '../src/lib/palette';
import { PRESETS, scheme, type Preset } from '../src/lib/scheme';

const [sampleDir, ...anchors] = process.argv.slice(2);
if (!sampleDir) {
	console.error('usage: evaluate-schemes.ts <sample-dir> [anchor ...]');
	process.exit(1);
}

const data: BlockData = JSON.parse(
	readFileSync(new URL('../static/data/26.3/blocks.json', import.meta.url), 'utf8')
);
const byId = new Map(data.blocks.map((b) => [b.id, b]));
const pool = data.blocks.filter((b) => !b.same_as);

const KEYS = ['blockOne', 'blockTwo', 'blockThree', 'blockFour', 'blockFive', 'blockSix'];
const sample: Block[][] = readdirSync(sampleDir)
	.filter((f) => /^(pop|rec)\d+\.json$/.test(f))
	.flatMap((f) => JSON.parse(readFileSync(join(sampleDir, f), 'utf8')).palettes)
	.map((p: Record<string, string>) =>
		KEYS.map((k) => byId.get(p[k])).filter((b): b is Block => !!b && !b.same_as)
	)
	.filter((p: Block[]) => p.length >= 5);

// --- stats (mirrors the research analysis) ---

const lch = (b: Block) => {
	const [l, a, bb] = b.faces.side.oklab;
	return { l, c: Math.hypot(a, bb), h: ((Math.atan2(bb, a) * 180) / Math.PI + 360) % 360 };
};

function hueGroups(blocks: Block[]): number {
	const hues = blocks
		.map(lch)
		.filter((x) => x.c >= 0.04)
		.map((x) => x.h)
		.sort((a, b) => a - b);
	if (!hues.length) return 0;
	const gaps = hues.map((h, i) => (i + 1 < hues.length ? hues[i + 1] - h : hues[0] + 360 - h));
	return Math.max(1, gaps.filter((g) => g > 35).length);
}

interface Stats {
	lightRange: number;
	maxChroma: number;
	strong: number;
	familyPair: boolean;
	nearPair: boolean;
	features: number;
	materials: number;
	seeThrough: number;
	hueGroups: number;
}

function stats(blocks: Block[]): Stats {
	const ls = blocks.map((b) => lch(b).l);
	const cs = blocks.map((b) => lch(b).c);
	const fams = new Map<string, number>();
	for (const b of blocks) fams.set(b.family, (fams.get(b.family) ?? 0) + 1);
	let near = Infinity;
	for (let i = 0; i < blocks.length; i++)
		for (let j = i + 1; j < blocks.length; j++)
			near = Math.min(near, labDistance(blocks[i].faces.side.oklab, blocks[j].faces.side.oklab));
	return {
		lightRange: Math.max(...ls) - Math.min(...ls),
		maxChroma: Math.max(...cs),
		strong: cs.filter((c) => c >= 0.1).length,
		familyPair: [...fams.values()].some((n) => n >= 2),
		nearPair: near < 0.04,
		features: blocks.filter((b) => b.feature).length,
		materials: new Set(blocks.map((b) => b.material)).size,
		seeThrough: blocks.filter((b) => b.translucent || b.overall.coverage < 0.99).length,
		hueGroups: hueGroups(blocks)
	};
}

/** Same rough archetypes as the research write-up */
function kind(s: Stats): Preset {
	if (s.lightRange < 0.22) return 'tonal';
	if (s.maxChroma < 0.1) return 'earthy';
	if (s.hueGroups >= 2) return 'twotone';
	return 'accent';
}

// --- report ---

const median = (xs: number[]) => {
	const s = [...xs].sort((a, b) => a - b);
	return s[Math.floor(s.length / 2)];
};
const pct = (xs: boolean[]) => `${Math.round((100 * xs.filter(Boolean).length) / xs.length)}%`;

function summarize(all: Stats[]) {
	return {
		n: all.length,
		'light range': median(all.map((s) => s.lightRange)).toFixed(2),
		'max chroma': median(all.map((s) => s.maxChroma)).toFixed(3),
		'strong (med)': median(all.map((s) => s.strong)),
		'family pair': pct(all.map((s) => s.familyPair)),
		'near pair': pct(all.map((s) => s.nearPair)),
		'feature>1': pct(all.map((s) => s.features > 1)),
		materials: median(all.map((s) => s.materials)),
		'see-through': pct(all.map((s) => s.seeThrough > 0)),
		'hue groups': median(all.map((s) => s.hueGroups))
	};
}

const sampleStats = sample.map(stats);
const presets = Object.keys(PRESETS) as Preset[];
const rows: Record<string, ReturnType<typeof summarize>> = {};
rows['sample (all)'] = summarize(sampleStats);
for (const preset of presets) {
	const ofKind = sampleStats.filter((s) => kind(s) === preset);
	rows[`sample: ${preset}`] = summarize(ofKind);
	const generated = sample.map((p) =>
		scheme(p[0], pool, { face: 'side', preset }).flatMap((g) => g.blocks)
	);
	const genStats = generated.map(stats);
	rows[`generated: ${preset}`] = summarize(genStats);
	rows[`  …classified as ${preset}`] = {
		...summarize(genStats),
		n: genStats.filter((s) => kind(s) === preset).length
	};
}
console.table(rows);

for (const id of anchors.length
	? anchors
	: [
			'spruce_planks',
			'deepslate_bricks',
			'poplar_planks',
			'cherry_planks',
			'quartz_block',
			'mud_bricks'
		]) {
	const anchor = byId.get(id);
	if (!anchor) continue;
	console.log(`\n${anchor.name}`);
	for (const preset of presets) {
		const groups = scheme(anchor, pool, { face: 'side', preset });
		console.log(
			`  ${preset.padEnd(8)}`,
			groups.map((g) => `${g.role}: ${g.blocks.map((b) => b.name).join(', ')}`).join(' | ')
		);
	}
}
