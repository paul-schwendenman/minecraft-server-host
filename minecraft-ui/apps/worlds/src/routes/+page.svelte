<script>
	import { ServerStatus } from '@minecraft/ui';
	import { status } from '@minecraft/data';

	let serverStatus = status.refresh();

	function handleRefresh() {
		serverStatus = status.refresh();
	}
</script>

<svelte:head>
	<title>Minecraft Server Manager</title>
</svelte:head>

<section
	class="min-w-sm flex h-full max-w-full flex-1 flex-col justify-between p-8 sm:mx-auto sm:max-w-sm sm:pt-16"
>
	{#await serverStatus}
		<p class="my-2">Loading...</p>
	{:then}
		<ServerStatus />
	{:catch error}
		<p class="my-2 text-red-700">{error.message}</p>
		<button on:click={handleRefresh} class="btn w-full sm:w-auto"> Retry </button>
	{/await}
</section>
