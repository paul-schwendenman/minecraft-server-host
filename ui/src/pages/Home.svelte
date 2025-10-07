<script>
  import ServerStatus from "../components/ServerStatus.svelte";
  import { status } from "../stores.js";

  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/service-worker.js");
  }

  let serverStatus = status.refresh();

  function handleRefresh() {
    serverStatus = status.refresh();
  }
</script>

<section class="h-full max-w-full p-8 sm:pt-16 sm:max-w-sm sm:mx-auto">
  {#await serverStatus}
    <p>Loading...</p>
  {:then _}
    <ServerStatus />
  {:catch error}
    <p class="text-red-700">{error.message}</p>
    <button on:click={handleRefresh} class="btn w-full sm:w-auto">
      Retry
    </button>
  {/await}
</section>

<style global>
</style>
