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

<button
	on:click={handleClick}
	class=" m-px min-h-12 cursor-pointer rounded border border-gray-500 bg-gray-200 p-4 text-lg text-gray-700 outline-none sm:px-4 sm:py-2 sm:text-base dark:border-none dark:bg-gray-700 dark:text-gray-100 {$$props.class}"
	disabled={pending}
>
	{#if !pending}
		<slot></slot>
	{:else}
		<Spinner />
	{/if}
</button>
