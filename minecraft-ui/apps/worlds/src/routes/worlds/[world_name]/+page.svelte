<script lang="ts">
	import Header from '$lib/Header.svelte';
	export let data;
	const { world, dimensions } = data;
</script>

<svelte:head>
	<title>{world.world} | Map | Minecraft Server Manager</title>
</svelte:head>

<div class="space-y-6 p-4">
	<Header
		breadcrumbs={[{ label: 'Worlds', href: '/worlds' }, { label: world.world }]}
		title={world.world}
		preview={world.previewUrl}
		subtitle="Select a dimension to explore"
	/>

	<section class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3">
		{#each dimensions as dim}
			<a
				href={`/worlds/${world.world}/${dim.name}`}
				class="card bg-base-200 border-base-300 hover:bg-base-300 border transition"
			>
				<figure class="aspect-video overflow-hidden">
					<img src={dim.previewUrl} alt={dim.name} class="h-full w-full object-cover" />
				</figure>
				<div class="card-body p-4">
					<h2 class="card-title text-base-content">{dim.name}</h2>
					<p class="text-base-content/70 text-sm capitalize">{dim.name}</p>
				</div>
			</a>
		{/each}
	</section>
</div>
