<script>
  import Spinner from "./Spinner.svelte";

  export let action;

  let pending = false;

  const handleClick = () => {
    if (!pending) {
      pending = true;

      action().then(() => {
        pending = false;
      });
    }
  };
</script>

<button on:click={handleClick} class="btn {$$props.class}" disabled={pending}>
  {#if !pending}
    <slot></slot>
  {:else}
    <Spinner />
  {/if}
</button>
