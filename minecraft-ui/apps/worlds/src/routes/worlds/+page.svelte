<script>
	import { onMount } from 'svelte';
	let worlds = [];
	let error = null;
	let loading = true;

	onMount(async () => {
		try {
			const res = await fetch('/api/worlds');
			if (!res.ok) throw new Error('Failed to fetch worlds');
			const data = await res.json();
			worlds = data?.worlds ?? [];
		} catch (err) {
			console.error(err);
			error = err;
		} finally {
			loading = false;
		}
	});

	function formatDate(ts) {
		if (!ts) return 'unknown';
		try {
			return new Date(ts).toLocaleString();
		} catch {
			return ts;
		}
	}
</script>

<section class="mx-auto max-w-5xl p-6 text-center sm:p-10">
	<h1 class="mb-8 text-3xl font-bold">Minecraft World Maps</h1>

	{#if loading}
		<div class="animate-pulse text-gray-400">Loading worlds...</div>
	{:else if error}
		<p class="text-red-500">{error.message}</p>
	{:else if worlds.length === 0}
		<p class="text-gray-400">No worlds found.</p>
	{:else}
		<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
			{#each worlds as world}
				<a
					href={`/worlds/${world.name}`}
					class="group block transform overflow-hidden rounded-2xl bg-gray-800 shadow transition hover:-translate-y-1 hover:shadow-lg"
				>
					{#if world.preview_url}
						<img
							src={world.preview_url}
							alt="Map preview"
							class="h-40 w-full object-cover transition-opacity group-hover:opacity-90"
							on:error={(e) => (e.target.style.display = 'none')}
						/>
					{/if}

					<div class="p-4 text-left">
						<h2 class="text-lg font-semibold capitalize text-white group-hover:text-blue-400">
							{world.name}
						</h2>

						<p class="mt-1 text-sm text-gray-400">
							Updated {formatDate(world.last_updated)}
						</p>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</section>
