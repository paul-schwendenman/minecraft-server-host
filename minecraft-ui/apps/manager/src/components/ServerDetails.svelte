<script>
  import { details } from "../stores.js";
  import {
    ActivePlayerList,
    ActivePlayerMessage,
    ServerVersion,
  } from "@minecraft/ui";
</script>

{#await $details}
  <p>Loading details...</p>
{:then details}
  <ServerVersion version={details?.version?.name} />
  <ActivePlayerMessage count={details?.players?.online} />
  <ActivePlayerList players={details?.players?.sample} />
{:catch _}
  <p class="text-red-700">Failed to load details.</p>
{/await}

<style>
</style>
