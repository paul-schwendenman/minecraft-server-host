<script>
	import { details } from '@minecraft/data';
	import { ActivePlayerList, ActivePlayerMessage, ServerVersion } from '@minecraft/ui';
</script>

{#await $details}
	<p>Loading details...</p>
{:then details}
	<ServerVersion version={details?.version?.name ?? ''} />
	<ActivePlayerMessage count={details?.players?.online ?? 0} />
	<ActivePlayerList players={details?.players?.sample ?? []} />
{:catch}
	<p class="my-2 text-red-700">Failed to load details.</p>
{/await}

<style>
</style>
