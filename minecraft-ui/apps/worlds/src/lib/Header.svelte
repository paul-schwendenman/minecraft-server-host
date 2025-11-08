<script lang="ts">
	import Breadcrumbs from '$lib/Breadcrumbs.svelte';

	export let breadcrumbs: { label: string; href?: string }[] = [];
	export let title: string;
	export let subtitle: string | null = null;
	export let preview: string | null = null;
	export let actionLabel: string | null = null;
	export let actionHref: string | null = null;
</script>

<div class="mb-6 rounded-xl border border-base-300 bg-base-200/80 p-4 shadow backdrop-blur-sm">
	{#if breadcrumbs.length}
		<Breadcrumbs items={breadcrumbs} />
	{/if}

	<div class="mt-2 flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			{#if preview}
				<img
					src={preview}
					alt="Preview"
					class="h-14 w-14 rounded-lg border border-base-300 object-cover"
				/>
			{/if}
			<div>
				<h1 class="text-2xl font-bold text-base-content">{title}</h1>
				{#if subtitle}
					<p class="text-sm text-base-content/70">{subtitle}</p>
				{/if}
			</div>
		</div>

		{#if actionLabel && actionHref}
			<!-- eslint-disable-next-line -->
			<a href={actionHref} target="_blank" rel="noopener noreferrer" class="btn btn-sm btn-primary">
				{actionLabel} ↗
			</a>
		{/if}
	</div>
</div>
