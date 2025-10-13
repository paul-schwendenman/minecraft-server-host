<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { getWorldDimension, type Dimension } from '@minecraft/data';

	let dimension: Dimension | null = null;
	let loading = true;
	let error: string | null = null;
	let showInline = false;

	$: worldName = $page.params.world_name;
	$: dimName = $page.params.dim;

	onMount(async () => {
		try {
			dimension = await getWorldDimension(worldName, dimName);
		} catch (err) {
			error = err instanceof Error ? err.message : String(err);
		} finally {
			loading = false;
		}
	});

	function toggleInline() {
		showInline = !showInline;
	}
</script>

{#if loading}
	<p>Loading dimension...</p>
{:else if error}
	<p class="text-red-600">Error: {error}</p>
{:else if dimension}
	<section class="space-y-4 p-4">
		<div class="flex items-center justify-between">
			<h1 class="text-2xl font-bold capitalize">{dimension.name}</h1>
			<button
				on:click={toggleInline}
				class="rounded-lg bg-blue-600 px-3 py-1.5 text-white shadow hover:bg-blue-700"
			>
				{#if showInline}
					Close Inline View
				{:else}
					View Inline
				{/if}
			</button>
		</div>

		<img
			src={dimension.previewUrl}
			alt={`${dimension.name} preview`}
			class="w-full max-w-3xl rounded-xl shadow"
		/>

		{#if showInline}
			<div class="mt-4">
				<iframe src={dimension.mapUrl} class="h-[80vh] w-full rounded-xl border shadow-inner" />
			</div>
		{:else}
			<div class="mt-4">
				<a
					href={dimension.mapUrl}
					target="_blank"
					class="inline-block rounded-lg bg-green-600 px-4 py-2 text-white shadow hover:bg-green-700"
				>
					Open Full Map ↗
				</a>
			</div>
		{/if}
	</section>
{/if}
