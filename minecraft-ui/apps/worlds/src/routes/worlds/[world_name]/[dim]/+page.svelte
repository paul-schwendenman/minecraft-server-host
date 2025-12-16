<script lang="ts">
	import Header from '$lib/Header.svelte';
	export let data;

	const { world, map, preview, mapUrl } = data;
</script>

<svelte:head>
	<title>{world.name} – {map.name} | Maps | Minecraft Server Manager</title>
</svelte:head>

<Header
	breadcrumbs={[
		{ label: 'Worlds', href: '/worlds' },
		{ label: world.name, href: `/worlds/${world.name}` },
		{ label: map.name }
	]}
	title={`${world.name} – ${map.name}`}
	{preview}
	actionLabel="Open Full Map"
	actionHref={mapUrl}
/>

<section class="h-[75vh] w-full overflow-hidden rounded-xl border shadow-inner">
	<iframe src={mapUrl} class="h-full w-full rounded-lg border-0" loading="lazy" title="Map">
	</iframe>
</section>
