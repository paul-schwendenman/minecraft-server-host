<script>
	import ServerDetails from './ServerDetails.svelte';
	import { AsyncButton } from '@minecraft/ui';
	import { status } from '@minecraft/data';

	const handleRefresh = () => {
		return status.refresh();
	};

	const handleStop = () => {
		return status.dispatch('stopInstance');
	};

	const handleStart = () => {
		return status.dispatch('startInstance');
	};

	const handleSyncDNS = () => {
		return status.dispatch('syncDnsRecord');
	};
</script>

<div class="flex h-full flex-1 flex-col justify-between md:justify-start">
	<div class="mb-4">
		<header>
			<h1 class="my-4 text-2xl font-semibold">{$status.dns_record?.name}</h1>
		</header>
		<p>
			Server is {$status.instance?.state}.
		</p>
		{#if $status.instance?.state == 'running'}
			<p>IP address: <code>{$status.instance?.ip_address}</code></p>

			{#if $status.instance?.ip_address == $status.dns_record?.value}
				<ServerDetails />
			{/if}
		{/if}
	</div>
	<div class="flex flex-col flex-wrap gap-1 sm:flex-row">
		{#if $status.instance?.state == 'stopped'}
			<AsyncButton class="flex-1" action={handleStart}>Start</AsyncButton>
		{:else if $status.instance?.state == 'running'}
			{#if $status.instance?.ip_address != $status.dns_record?.value}
				<AsyncButton class="flex-2" action={handleSyncDNS}>Update DNS</AsyncButton>
			{/if}
			<AsyncButton class="flex-1" action={handleStop}>Stop</AsyncButton>
		{/if}
		<AsyncButton class="flex-1" action={handleRefresh}>Refresh</AsyncButton>
	</div>
</div>
