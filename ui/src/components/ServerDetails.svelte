<script>
	import ActivePlayerMessage from './ActivePlayerMessage.svelte';
	import ActivePlayerList from './ActivePlayerList.svelte';
	import { details } from '../stores.js';
	import ServerVersion from './ServerVersion.svelte';
</script>

<style>
</style>

{#await $details}
	<p>Loading details...</p>
{:then details}
	<ServerVersion version={details?.version?.name} />
	<ActivePlayerMessage count={details?.players?.online} />
	<ActivePlayerList players={details?.players?.sample} />
{:catch _}
	<p class="text-red-700">Failed to load details.</p>
{/await}
