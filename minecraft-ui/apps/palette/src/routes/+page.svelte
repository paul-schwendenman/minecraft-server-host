<script lang="ts">
	import { page } from '$app/state';
	import { afterNavigate, replaceState } from '$app/navigation';
	import { resolve } from '$app/paths';
	import BlockPicker from '$lib/BlockPicker.svelte';
	import BlockTile from '$lib/BlockTile.svelte';
	import {
		blockPool,
		gradient,
		labToCss,
		similar,
		type Blend,
		type Block,
		type Face,
		type GradientStep
	} from '$lib/palette';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	type Mode = 'gradient' | 'similar';
	type PickerTarget = 'from' | 'to' | 'seed';

	const FACES: { value: Face; label: string }[] = [
		{ value: 'side', label: 'Side' },
		{ value: 'top', label: 'Top' }
	];
	const BLENDS: { value: Blend; label: string; title: string }[] = [
		{
			value: 'straight',
			label: 'Straight',
			title: 'Direct color blend; distant hues meet in gray'
		},
		{ value: 'hue', label: 'Hue', title: 'Travel around the color wheel, staying saturated' }
	];
	const SIMILAR_COUNT = 15;

	const blocks = $derived(data.blockData.blocks);
	const byId = $derived(new Map(blocks.map((b) => [b.id, b])));
	const pickable = $derived(blocks.filter((b) => !b.same_as));

	// --- state, initialized from (and mirrored to) the query string ---
	const params = page.url.searchParams;
	const intParam = (key: string, fallback: number, min: number, max: number) => {
		const n = Number.parseInt(params.get(key) ?? '', 10);
		return Number.isNaN(n) ? fallback : Math.min(max, Math.max(min, n));
	};

	let mode = $state<Mode>(params.get('mode') === 'similar' ? 'similar' : 'gradient');
	let face = $state<Face>(params.get('face') === 'top' ? 'top' : 'side');
	let fromId = $state(params.get('from') ?? 'deepslate_tiles');
	let toId = $state(params.get('to') ?? 'cut_sandstone');
	let seedId = $state(params.get('seed') ?? 'oak_planks');
	let steps = $state(intParam('steps', 7, 3, 12));
	let blend = $state<Blend>(params.get('blend') === 'hue' ? 'hue' : 'straight');
	let texture = $state(intParam('texture', 30, 0, 100));
	let seeThrough = $state(params.get('see') === '1');
	let animated = $state(params.get('anim') !== '0');

	const query = $derived(
		Object.entries({
			mode,
			face,
			...(mode === 'gradient' ? { from: fromId, to: toId, steps, blend } : { seed: seedId }),
			texture,
			...(seeThrough ? { see: 1 } : {}),
			...(animated ? {} : { anim: 0 })
		})
			.map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
			.join('&')
	);

	// replaceState throws if called before the router has started
	let routerReady = $state(false);
	afterNavigate(() => (routerReady = true));

	$effect(() => {
		if (routerReady) replaceState(resolve(`/?${query}`), {});
	});

	// --- palette ---
	const pool = $derived(blockPool(blocks, { seeThrough, animated }));
	// 0-100 slider -> penalty weight; noise values sit around 0-0.2, colors 0-1
	const textureWeight = $derived(texture / 50);

	const from = $derived(byId.get(fromId) ?? pickable[0]);
	const to = $derived(byId.get(toId) ?? pickable[1]);
	const seed = $derived(byId.get(seedId) ?? pickable[0]);

	const gradientSteps: GradientStep[] = $derived(
		gradient(from, to, pool, { face, steps, textureWeight, blend })
	);
	const similarBlocks: Block[] = $derived(
		similar(seed, pool, { face, count: SIMILAR_COUNT, textureWeight })
	);
	const palette = $derived(
		mode === 'gradient' ? gradientSteps.map((s) => s.block) : [seed, ...similarBlocks]
	);

	// --- picker ---
	let pickerOpen = $state(false);
	let pickerTarget = $state<PickerTarget>('from');
	const pickerTitles: Record<PickerTarget, string> = {
		from: 'Gradient start',
		to: 'Gradient end',
		seed: 'Find blocks similar to…'
	};

	function openPicker(target: PickerTarget) {
		pickerTarget = target;
		pickerOpen = true;
	}

	function onpick(block: Block) {
		if (pickerTarget === 'from') fromId = block.id;
		else if (pickerTarget === 'to') toId = block.id;
		else seedId = block.id;
	}

	function swap() {
		[fromId, toId] = [toId, fromId];
	}

	let copied = $state(false);
	async function copyList() {
		await navigator.clipboard.writeText(palette.map((b) => b.name).join('\n'));
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

{#snippet blockButton(label: string, block: Block, target: PickerTarget)}
	<button
		class="btn h-auto justify-start gap-3 py-2 normal-case btn-ghost"
		onclick={() => openPicker(target)}
	>
		<BlockTile {block} {face} size={40} class="rounded-sm" />
		<span class="flex flex-col items-start text-left">
			<span class="text-xs text-base-content/60">{label}</span>
			<span>{block.name}</span>
		</span>
	</button>
{/snippet}

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-4 rounded-xl border border-base-300 bg-base-200/80 p-4 shadow">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div role="tablist" class="tabs-box tabs">
				<button
					role="tab"
					class="tab"
					class:tab-active={mode === 'gradient'}
					onclick={() => (mode = 'gradient')}>Gradient</button
				>
				<button
					role="tab"
					class="tab"
					class:tab-active={mode === 'similar'}
					onclick={() => (mode = 'similar')}>Similar</button
				>
			</div>

			<div class="join" aria-label="Face to match">
				{#each FACES as f (f.value)}
					<button
						class="btn join-item btn-sm"
						class:btn-active={face === f.value}
						onclick={() => (face = f.value)}>{f.label}</button
					>
				{/each}
			</div>
		</div>

		{#if mode === 'gradient'}
			<div class="flex flex-wrap items-center gap-2">
				{@render blockButton('From', from, 'from')}
				<button class="btn btn-circle btn-ghost btn-sm" onclick={swap} aria-label="Swap">⇄</button>
				{@render blockButton('To', to, 'to')}
				<div class="ml-auto flex flex-wrap items-center gap-x-4 gap-y-2 text-sm">
					<div class="join" aria-label="Blend">
						{#each BLENDS as b (b.value)}
							<button
								class="btn join-item btn-sm"
								class:btn-active={blend === b.value}
								title={b.title}
								onclick={() => (blend = b.value)}>{b.label}</button
							>
						{/each}
					</div>
					<label class="flex items-center gap-3">
						<span class="whitespace-nowrap">{steps} blocks</span>
						<input type="range" class="range w-32 range-sm" min="3" max="12" bind:value={steps} />
					</label>
				</div>
			</div>
		{:else}
			<div class="flex flex-wrap items-center gap-2">
				{@render blockButton('Similar to', seed, 'seed')}
			</div>
		{/if}

		<div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
			<label class="flex items-center gap-3">
				<span class="whitespace-nowrap" title="Prefer blocks with similarly busy or smooth textures"
					>Match texture</span
				>
				<input type="range" class="range w-32 range-sm" min="0" max="100" bind:value={texture} />
			</label>
			<label class="flex cursor-pointer items-center gap-2">
				<input type="checkbox" class="toggle toggle-sm" bind:checked={seeThrough} />
				See-through blocks
			</label>
			<label class="flex cursor-pointer items-center gap-2">
				<input type="checkbox" class="toggle toggle-sm" bind:checked={animated} />
				Animated blocks
			</label>
		</div>
	</div>

	<!-- Seamless wall so textures can be judged side by side -->
	<div
		class="grid overflow-hidden rounded-lg border border-base-300"
		style:grid-template-columns="repeat({mode === 'gradient' ? palette.length : 8}, minmax(0, 1fr))"
	>
		{#each mode === 'gradient' ? [0, 1] : [0] as row (row)}
			{#each palette as block, i (`${row}-${i}`)}
				<BlockTile {block} {face} size={128} class="h-auto w-full" />
			{/each}
		{/each}
	</div>

	<div class="flex items-center justify-between">
		<h2 class="text-lg font-bold">
			{mode === 'gradient' ? 'Gradient' : `Closest to ${seed.name}`}
		</h2>
		<button class="btn btn-sm" onclick={copyList}>{copied ? 'Copied!' : 'Copy list'}</button>
	</div>

	<ol class="grid grid-cols-[repeat(auto-fill,minmax(14rem,1fr))] gap-2">
		{#if mode === 'gradient'}
			{#each gradientSteps as step, i (i)}
				<li class="flex items-center gap-3 rounded-lg bg-base-200 p-2">
					<span class="w-5 text-right text-xs text-base-content/50">{i + 1}</span>
					<BlockTile block={step.block} {face} size={40} class="rounded-sm" />
					<span class="flex-1 text-sm">{step.block.name}</span>
					<span
						class="flex h-8 w-4 flex-col overflow-hidden rounded-sm"
						title="Ideal color (top) vs block color (bottom)"
					>
						<span class="flex-1" style:background-color={labToCss(step.target)}></span>
						<span class="flex-1" style:background-color={step.block.faces[face].hex}></span>
					</span>
				</li>
			{/each}
		{:else}
			{#each similarBlocks as block (block.id)}
				<li>
					<button
						class="flex w-full items-center gap-3 rounded-lg bg-base-200 p-2 text-left hover:bg-base-300"
						onclick={() => (seedId = block.id)}
						title="Find blocks similar to {block.name}"
					>
						<BlockTile {block} {face} size={40} class="rounded-sm" />
						<span class="flex-1 text-sm">{block.name}</span>
					</button>
				</li>
			{/each}
		{/if}
	</ol>

	<p class="text-xs text-base-content/50">
		Minecraft {data.blockData.version} · {pool.length} blocks in pool
	</p>
</div>

<BlockPicker
	bind:open={pickerOpen}
	blocks={pickable}
	{face}
	title={pickerTitles[pickerTarget]}
	{onpick}
/>
