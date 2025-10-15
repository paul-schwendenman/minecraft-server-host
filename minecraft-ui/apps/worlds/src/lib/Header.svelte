<script lang="ts">
	export let breadcrumbs: { label: string; href?: string }[] = [];
	export let title: string;
	export let subtitle: string | null = null;
	export let preview: string | null = null;
	export let actionLabel: string | null = null;
	export let actionHref: string | null = null;
</script>

<div class="bg-base-200/80 border-base-300 mb-6 rounded-xl border p-4 shadow backdrop-blur-sm">
	<!-- Breadcrumbs -->
	{#if breadcrumbs.length}
		<nav class="text-base-content/70 mb-1 flex flex-wrap gap-1 text-sm">
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

	<div class="flex items-center justify-between gap-4">
		<div class="flex items-center gap-3">
			{#if preview}
				<img
					src={preview}
					alt="Preview"
					class="border-base-300 h-16 w-16 rounded-lg border object-cover"
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
