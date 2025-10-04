<script>
	import { getStatus, startInstance, stopInstance, syncDnsRecord } from './data.service';
	import ServerStatus from './components/ServerStatus.svelte';

	let serverStatus = getStatus();

	function handleStart() {
		serverStatus = startInstance()
			.then(() => getStatus());
	}
	function handleStop() {
		serverStatus = stopInstance()
			.then(() => getStatus());
	}
	function handleSyncDNS() {
		serverStatus = syncDnsRecord()
			.then(() => getStatus());
	}
	function handleRefresh() {
		serverStatus = getStatus();
	}
</script>

<style>
	section {
		margin: 5rem auto;
		max-width: 360px;
	}

	.error {
		color: #d32f2f;
	}

</style>

<section>
	{#await serverStatus}
		<p>Loading...</p>
	{:then data}
		<ServerStatus
			serverStatus={data}
			handleStart={handleStart}
			handleStop={handleStop}
			handleRefresh={handleRefresh}
			handleSyncDNS={handleSyncDNS}
		/>
	{:catch error}
		<p class="error">{error.message}</p>
		<button on:click={handleRefresh} class="button">
			Retry
		</button>
	{/await}
</section>
