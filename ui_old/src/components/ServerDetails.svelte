<script>
	import ActivePlayerMessage from './ActivePlayerMessage.svelte';
	import ActivePlayerList from './ActivePlayerList.svelte';

	export let serverDetails;
</script>

<style>
	.error {
		color: #d32f2f;
	}
</style>

{#await serverDetails}
	<p>Loading details...</p>
{:then details}
	<ActivePlayerMessage count={details.players.online} />
	<ActivePlayerList players={details.players.sample} />
{:catch error}
	<p class="error">Failed to load details.</p>
{/await}
