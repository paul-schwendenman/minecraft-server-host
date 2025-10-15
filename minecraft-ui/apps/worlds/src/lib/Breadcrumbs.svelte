<script lang="ts">
	export let items: { label: string; href?: string }[] = [];
</script>

<nav class="breadcrumbs text-sm">
	<ul>
		{#each items as crumb, i}
			<li>
				{#if crumb.href}
					<a href={crumb.href}>{crumb.label}</a>
				{:else}
					<span class="opacity-70">{crumb.label}</span>
				{/if}
			</li>
		{/each}
	</ul>
</nav>
