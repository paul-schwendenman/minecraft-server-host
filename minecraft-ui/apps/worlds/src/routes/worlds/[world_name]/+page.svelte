<script>
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';

	export let params;
	let world = null;
	let loading = true;

	onMount(async () => {
		const res = await fetch(`/api/worlds/${params.world}/maps`);
		const data = await res.json();
		world = data;
		loading = false;
	});
</script>

<section class="mx-auto max-w-3xl p-6">
	{#if loading}
		<p class="text-gray-400">Loading {params.world}...</p>
	{:else}
		<h1 class="mb-4 text-3xl font-bold capitalize">{world.world}</h1>
		<ul class="space-y-2">
			{#each world.dimensions as dim (dim.id)}
				<li>
					<a href={resolve(dim.map_url)} class="text-blue-400 hover:underline">
						{dim.id}
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</section>
