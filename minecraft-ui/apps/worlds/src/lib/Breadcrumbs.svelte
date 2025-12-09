<script lang="ts">
	export let items: { label: string; href?: string }[] = [];
</script>

<nav class="breadcrumbs text-sm">
	<ul>
		{#each items as crumb (crumb.label)}
			<li>
				{#if crumb.href}
					<!-- eslint-disable-next-line -->
					<a href={crumb.href} data-sveltekit-prefetch>{crumb.label}</a>
				{:else}
					<span class="opacity-70">{crumb.label}</span>
				{/if}
			</li>
		{/each}
	</ul>
</nav>
