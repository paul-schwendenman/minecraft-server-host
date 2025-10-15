<script lang="ts">
	import { onMount } from 'svelte';
	import { listWorlds, type World } from '@minecraft/data';
	import Header from '$lib/Header.svelte';

	let worlds: World[] = [];
	let error: string | null = null;
	let loading = true;

	onMount(async () => {
		try {
			worlds = await listWorlds();
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	});
</script>

<Header title="Worlds" subtitle="Available Minecraft worlds" />

{#if loading}
	<p>Loading worlds...</p>
{:else if error}
	<p class="text-red-600">Error: {error}</p>
{:else}
	<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
		{#each worlds as w}
			<a
				href={`/worlds/${w.world}`}
				class="block overflow-hidden rounded-xl bg-white shadow hover:shadow-lg"
			>
				<img src={w.previewUrl} alt={w.world} class="h-56 w-full object-cover" />
				<div class="p-3">
					<h2 class="text-lg font-bold">{w.world}</h2>
					<p class="text-sm text-gray-600">{w.dimensions.length} dimensions</p>
				</div>
			</a>
		{/each}
	</div>
{/if}
