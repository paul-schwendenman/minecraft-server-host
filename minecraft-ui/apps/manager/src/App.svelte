<script>
  import Router from "svelte-spa-router";
  export let routes;
</script>

<main class="min-h-screen flex flex-col bg-gray-900 text-gray-100">
  <nav class="flex justify-center space-x-6 p-4 text-sm text-gray-400">
    <a href="/" class="hover:text-gray-200">Server</a>
    <a href="#/maps" class="hover:text-gray-200">Maps</a>
  </nav>
  <Router {routes} />
</main>
