<script>
  import { push } from "svelte-spa-router";
</script>

<section class="h-screen flex flex-col justify-center items-center text-center">
  <h1 class="text-2xl font-semibold mb-4">404 – Page not found</h1>
  <button class="btn" on:click={() => push("/")}>Go Home</button>
</section>
