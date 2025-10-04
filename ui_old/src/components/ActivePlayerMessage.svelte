<script>
	export let count;
</script>

{#if count == 0 }
	<p>The server has no active players.</p>
{:else if count == 1 }
	<p>The server has 1 active player:</p>
{:else}
	<p>The server has {count} active players:</p>
{/if}
