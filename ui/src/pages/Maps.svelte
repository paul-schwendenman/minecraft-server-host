<script>
  import { onMount } from "svelte";
  let worlds = [];
  let error = null;
  let loading = true;

  onMount(async () => {
    try {
      const res = await fetch("/api/worlds");
      if (!res.ok) throw new Error("Failed to fetch worlds");
      const data = await res.json();
      worlds = data?.worlds ?? [];
    } catch (err) {
      console.error(err);
      error = err;
    } finally {
      loading = false;
    }
  });

  function formatDate(ts) {
    if (!ts) return "unknown";
    try {
      return new Date(ts).toLocaleString();
    } catch {
      return ts;
    }
  }
</script>

<section class="max-w-5xl mx-auto p-6 sm:p-10 text-center">
  <h1 class="text-3xl font-bold mb-8">Minecraft World Maps</h1>

  {#if loading}
    <div class="animate-pulse text-gray-400">Loading worlds...</div>
  {:else if error}
    <p class="text-red-500">{error.message}</p>
  {:else if worlds.length === 0}
    <p class="text-gray-400">No worlds found.</p>
  {:else}
    <div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
      {#each worlds as world}
        <a
          href={world.map_url}
          class="group block bg-gray-800 rounded-2xl shadow hover:shadow-lg overflow-hidden transition transform hover:-translate-y-1"
        >
          {#if world.preview_url}
            <img
              src={world.preview_url}
              alt="Map preview"
              class="w-full h-40 object-cover group-hover:opacity-90 transition-opacity"
              on:error={(e) => (e.target.style.display = "none")}
            />
          {/if}

          <div class="p-4 text-left">
            <h2
              class="text-lg font-semibold text-white capitalize group-hover:text-blue-400"
            >
              {world.name}
            </h2>

            <p class="text-sm text-gray-400 mt-1">
              Updated {formatDate(world.last_updated)}
            </p>
          </div>
        </a>
      {/each}
    </div>
  {/if}
</section>
