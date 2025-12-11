<script lang="ts">
	import { resolve } from '$app/paths';

	export let title: string;
	export let subtitle: string | null = null;
	export let image: string;
	export let href: string;

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const resolvedHref = resolve(href as any);
</script>

<a href={resolvedHref} class="card border border-base-300 bg-base-200 transition hover:bg-base-300">
	<figure class="aspect-video overflow-hidden">
		<img src={image} alt={title} class="h-full w-full object-cover" />
	</figure>
	<div class="card-body p-4">
		<h2 class="card-title text-base-content">{title}</h2>
		{#if subtitle}
			<p class="text-sm text-base-content/70">{subtitle}</p>
		{/if}
	</div>
</a>
