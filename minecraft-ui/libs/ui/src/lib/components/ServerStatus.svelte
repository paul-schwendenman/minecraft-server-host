<script lang="ts">
	import type { Snippet } from 'svelte';
	import ServerDetails from './ServerDetails.svelte';
	import AsyncButton from './AsyncButton.svelte';
	import StartButton from './StartButton.svelte';
	import { status } from '@minecraft/data';

	// startButton and stopButton replace the plain Start (server stopped) and
	// Stop (running). The maps app passes WorldStartButton and
	// WorldSwitchButton, which add a menu of worlds to start or switch to.
	let { startButton, stopButton }: { startButton?: Snippet; stopButton?: Snippet } = $props();

	const handleRefresh = () => {
		return status.refresh();
	};

	const handleStop = () => {
		return status.dispatch('stopInstance');
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
		{#if $status.instance?.active_world}
			<p>World: <strong>{$status.instance.active_world}</strong></p>
		{/if}
		{#if $status.instance?.state == 'running'}
			<p>IP address: <code>{$status.instance?.ip_address}</code></p>

			{#if $status.instance?.ip_address == $status.dns_record?.value}
				<ServerDetails />
			{/if}
		{/if}
	</div>
	<div class="flex flex-col flex-wrap gap-1 sm:flex-row">
		{#if $status.instance?.state == 'stopped'}
			{#if startButton}
				{@render startButton()}
			{:else}
				<StartButton class="flex-2" />
			{/if}
		{:else if $status.instance?.state == 'running'}
			{#if $status.instance?.ip_address != $status.dns_record?.value}
				<!-- Plain Stop until DNS is synced: three buttons with a split one is
				     crowded, and nobody can connect by name to switch worlds yet -->
				<AsyncButton class="flex-2" action={handleSyncDNS}>Update DNS</AsyncButton>
				<AsyncButton class="flex-1" action={handleStop}>Stop</AsyncButton>
			{:else if stopButton}
				{@render stopButton()}
			{:else}
				<AsyncButton class="flex-1" action={handleStop}>Stop</AsyncButton>
			{/if}
		{/if}
		<AsyncButton class="flex-1" action={handleRefresh}>Refresh</AsyncButton>
	</div>
</div>
