<script>
	import { getStatus, startInstance, stopInstance, syncDnsRecord, getDetails } from './data.service';
	import ServerDetails from './components/ServerDetails.svelte';

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
		<header>
			<h1>{data.dns_record.name}</h1>
		</header>
		<p>
			Server is {data.instance.state}.
		</p>
		{#if data.instance.state == "stopped"}
			<button on:click={handleStart} class="button">
				Start
			</button>
		{:else if data.instance.state == "running"}
			<p>IP address: <code>{data.instance.ip_address}</code></p>

			{#if data.instance.ip_address != data.dns_record.value}
				<button on:click={handleSyncDNS} class="button">
					Update DNS Record
				</button>
			{:else}
				<ServerDetails serverDetails={getDetails()} />
			{/if}
			<button on:click={handleStop} class="button">
				Stop
			</button>
		{/if}
		<button on:click={handleRefresh} class="button">
			Refresh
		</button>
	{:catch error}
		<p class="error">{error.message}</p>
		<button on:click={handleRefresh} class="button">
			Retry
		</button>
	{/await}
</section>
