<script lang="ts">
	import Breadcrumbs from '$lib/Breadcrumbs.svelte';

	export let breadcrumbs: { label: string; href?: string }[] = [];
	export let title: string;
	export let subtitle: string | null = null;
	export let preview: string | null = null;
	export let actionLabel: string | null = null;
	export let actionHref: string | null = null;
</script>

<div class="bg-base-200/80 border-base-300 mb-6 rounded-xl border p-4 shadow backdrop-blur-sm">
	{#if breadcrumbs.length}
		<Breadcrumbs items={breadcrumbs} />
	{/if}

	<div class="mt-2 flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			{#if preview}
				<img
					src={preview}
					alt="Preview"
					class="border-base-300 h-14 w-14 rounded-lg border object-cover"
				/>
			{/if}
			<div>
				<h1 class="text-base-content text-2xl font-bold">{title}</h1>
				{#if subtitle}
					<p class="text-base-content/70 text-sm">{subtitle}</p>
				{/if}
			</div>
		</div>

		{#if actionLabel && actionHref}
			<a href={actionHref} target="_blank" rel="noopener" class="btn btn-primary btn-sm">
				{actionLabel} ↗
			</a>
		{/if}
	</div>
</div>
