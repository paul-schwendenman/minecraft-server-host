<script lang="ts">
	import Card from '$lib/Card.svelte';
	import Header from '$lib/Header.svelte';
	export let data;
	const { world, dimensions } = data;
</script>

<svelte:head>
	<title>{world.world} | Maps | Minecraft Server Manager</title>
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
			<Card title={dim.name} image={dim.previewUrl} href={`/worlds/${world.world}/${dim.name}`} />
		{/each}
	</section>
</div>
