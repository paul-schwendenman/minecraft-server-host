<script>
  import { ServerStatus } from "@minecraft/ui";
  import { status } from "@minecraft/data";

  if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/service-worker.js");
  }

  let serverStatus = status.refresh();

  function handleRefresh() {
    serverStatus = status.refresh();
  }
</script>

<section
  class="flex flex-col justify-between flex-1 h-full max-w-full p-8 sm:pt-16 sm:max-w-sm sm:mx-auto"
>
  {#await serverStatus}
    <p class="my-2">Loading...</p>
  {:then _}
    <ServerStatus />
  {:catch error}
    <p class="my-2 text-red-700">{error.message}</p>
    <button on:click={handleRefresh} class="btn w-full sm:w-auto">
      Retry
    </button>
  {/await}
</section>

<style global>
</style>
