<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { getWorld, type WorldDetail } from '@minecraft/data';
	import Header from '$lib/Header.svelte';

	let world: WorldDetail | null = null;
	let error: string | null = null;
	let loading = true;

	$: name = $page.params.world_name ?? '';

	onMount(async () => {
		try {
			world = await getWorld(name);
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	});
</script>

<Header
	breadcrumbs={[{ label: 'Worlds', href: '/worlds' }, { label: world?.world ?? 'Unknown' }]}
	title={world?.world ?? 'Unknown'}
	subtitle="Select a dimension to explore"
/>

{#if loading}
	<p>Loading world...</p>
{:else if error}
	<p class="text-red-600">Error: {error}</p>
{:else if world}
	<section class="p-4">
		<h1 class="mb-4 text-2xl font-bold">{world.world}</h1>
		<img src={world.previewUrl} alt={world.world} class="mb-6 w-full max-w-3xl rounded-xl shadow" />

		<h2 class="mb-2 text-xl font-semibold">Dimensions</h2>
		<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
			{#each world.dimensions as d}
				<div class="rounded-lg bg-white p-3 shadow hover:shadow-lg">
					<!-- <a href={d.mapUrl} target="_blank"> -->
					<a href={`/worlds/${name}/${d.name}`}>
						<img src={d.previewUrl} alt={d.name} class="mb-2 h-40 w-full rounded-md object-cover" />
						<h3 class="font-semibold capitalize text-gray-800">{d.name}</h3>
					</a>
				</div>
			{/each}
		</div>
	</section>
{/if}
