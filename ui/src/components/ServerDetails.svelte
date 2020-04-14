<script>
	import ActivePlayerMessage from './ActivePlayerMessage.svelte';

	export let serverDetails;
</script>

<style>
</style>

{#await serverDetails}
	<p>Loading details...</p>
{:then details}
	<ActivePlayerMessage count={details.players.online} />
	{#if details.players.sample}
		<ul>
			{#each details.players.sample as player}
				<li>{player.name}</li>
			{/each}
		</ul>
	{/if}
{:catch error}
	<!-- <p>Failed to load details.</p> -->
{/await}
