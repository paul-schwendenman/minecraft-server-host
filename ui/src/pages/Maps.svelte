<script>
  import { onMount } from "svelte";
  let maps = [];
  let error = null;
  let loading = true;

  onMount(async () => {
    try {
      const res = await fetch("/api/maps");
      if (!res.ok) throw new Error("Failed to fetch maps");
      maps = await res.json();
    } catch (err) {
      error = err;
    } finally {
      loading = false;
    }
  });
</script>

<section
  class="h-full max-w-full p-8 sm:pt-16 sm:max-w-lg sm:mx-auto text-center"
>
  <h1 class="text-2xl font-semibold mb-6">World Maps</h1>

  {#if loading}
    <div class="animate-pulse text-gray-400">Loading maps...</div>
  {:else if error}
    <p class="text-red-600">{error.message}</p>
  {:else if maps.length === 0}
    <p class="text-gray-400">No maps found.</p>
  {:else}
    <ul class="space-y-4">
      {#each maps as world}
        <li
          class="p-4 bg-gray-800 rounded-xl shadow-md hover:bg-gray-700 transition-colors"
        >
          <a href={world.url} class="block text-left">
            <h2 class="text-lg font-semibold text-white capitalize">
              {world.name}
            </h2>
            <p class="text-sm text-gray-400 mt-1">
              Updated {new Date(world.lastUpdated).toLocaleString()}
            </p>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>
