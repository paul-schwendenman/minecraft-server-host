<script>
  import Router from "svelte-spa-router";
  export let routes;
</script>

<main class="min-h-screen bg-gray-900 text-gray-100">
  <Router {routes} />
</main>
