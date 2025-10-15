<script lang="ts">
	export let breadcrumbs: { label: string; href?: string }[] = [];
	export let title: string;
	export let subtitle: string | null = null;
	export let preview: string | null = null;
	export let actionLabel: string | null = null;
	export let actionHref: string | null = null;
</script>

<div class="mb-4 flex flex-col gap-2">
	{#if breadcrumbs.length}
		<nav class="flex flex-wrap gap-1 text-sm text-gray-500">
			{#each breadcrumbs as crumb, i}
				{#if crumb.href}
					<a href={crumb.href} class="hover:underline">{crumb.label}</a>
				{:else}
					<span>{crumb.label}</span>
				{/if}
				{#if i < breadcrumbs.length - 1}
					<span class="text-gray-400">/</span>
				{/if}
			{/each}
		</nav>
	{/if}

	<!-- Main Header -->
	<div class="flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			{#if preview}
				<img
					src={preview}
					alt="Preview"
					class="h-16 w-16 rounded-xl border border-gray-200 object-cover"
				/>
			{/if}
			<div>
				<h1 class="text-2xl font-bold text-gray-900">{title}</h1>
				{#if subtitle}
					<p class="text-sm text-gray-500">{subtitle}</p>
				{/if}
			</div>
		</div>

		{#if actionLabel && actionHref}
			<a
				href={actionHref}
				target="_blank"
				rel="noopener"
				class="rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium hover:bg-gray-200"
			>
				{actionLabel} ↗
			</a>
		{/if}
	</div>
</div>
