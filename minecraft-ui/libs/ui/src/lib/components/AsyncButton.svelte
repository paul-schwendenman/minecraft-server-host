<script lang="ts">
	import Spinner from './Spinner.svelte';

	export let action: () => Promise<void>;

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

<button on:click={handleClick} class="btn btn-neutral {$$props.class}" disabled={pending}>
	{#if !pending}
		<slot></slot>
	{:else}
		<Spinner />
	{/if}
</button>
