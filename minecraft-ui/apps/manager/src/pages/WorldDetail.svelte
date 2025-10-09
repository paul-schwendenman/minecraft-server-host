<script>
  import { onMount } from "svelte";
  import { wrap } from "svelte-spa-router/wrap";

  export let params;
  let world = null;
  let loading = true;

  onMount(async () => {
    const res = await fetch(`/api/worlds/${params.world}/maps`);
    const data = await res.json();
    world = data;
    loading = false;
  });
</script>

<section class="max-w-3xl mx-auto p-6">
  {#if loading}
    <p class="text-gray-400">Loading {params.world}...</p>
  {:else}
    <h1 class="text-3xl font-bold mb-4 capitalize">{world.world}</h1>
    <ul class="space-y-2">
      {#each world.dimensions as dim}
        <li>
          <a href={dim.map_url} class="text-blue-400 hover:underline">
            {dim.id}
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</section>
