<script lang="ts">
	import { onMount } from 'svelte';
	import AsyncButton from './AsyncButton.svelte';
	import { status } from '@minecraft/data';

	let { world }: { world: string } = $props();

	let error = $state('');

	const serverState = $derived($status.instance?.state);
	const activeWorld = $derived($status.instance?.active_world);
	const isUp = $derived(serverState === 'pending' || serverState === 'running');

	const run = async (action: () => Promise<void>) => {
		error = '';
		try {
			await action();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	};

	const handlePlay = () => run(() => status.startWorld(world));
	const handleRefresh = () => run(() => status.refresh());

	onMount(() => {
		if (!$status.instance) {
			handleRefresh();
		}
	});
</script>

<div class="card border border-base-300 bg-base-200">
	<div class="card-body flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between">
		<p>
			{#if !serverState}
				Checking the server…
			{:else if serverState === 'stopped'}
				The server is off.
			{:else if isUp && activeWorld === world}
				{#if serverState === 'pending'}
					Starting <strong>{world}</strong>…
				{:else}
					<strong>{world}</strong> is running.
				{/if}
			{:else if isUp}
				The server is running <strong>{activeWorld}</strong>. Stop it to play this world.
			{:else if serverState === 'stopping'}
				The server is stopping. Try again once it's off.
			{:else}
				Server is {serverState}.
			{/if}
		</p>

		<div class="flex gap-1">
			{#if serverState === 'stopped'}
				<AsyncButton action={handlePlay}>Play {world}</AsyncButton>
			{:else if isUp && activeWorld !== world}
				<button class="btn btn-lg min-h-12 btn-neutral sm:btn-md" disabled>Play {world}</button>
			{/if}
			{#if serverState && serverState !== 'stopped' && serverState !== 'running'}
				<AsyncButton action={handleRefresh}>Refresh</AsyncButton>
			{/if}
		</div>
	</div>

	{#if error}
		<p class="px-4 pb-4 text-error">{error}</p>
	{/if}
</div>
