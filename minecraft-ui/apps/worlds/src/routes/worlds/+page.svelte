<script lang="ts">
	import Header from '$lib/Header.svelte';
	import Card from '$lib/Card.svelte';
	export let data;
	const { worlds } = data;
</script>

<svelte:head>
	<title>Maps | Minecraft Server Manager</title>
</svelte:head>

<div class="space-y-6 p-4">
	<Header title="Worlds" subtitle="Explore all Minecraft worlds" />

	<section class="grid gap-6 sm:grid-cols-2 md:grid-cols-3">
		{#each worlds as world (world.world)}
			<Card
				title={world.world}
				subtitle={world?.version ? `Version ${world?.version}` : null}
				image={world.previewUrl}
				href={`/worlds/${world.world}`}
			/>
		{/each}
	</section>
</div>
