<script lang="ts">
	import { onMount } from 'svelte';
	import AsyncButton from './AsyncButton.svelte';
	import Spinner from './Spinner.svelte';
	import { status } from '@minecraft/data';

	let { world }: { world: string } = $props();

	let error = $state('');
	let refreshing = $state(false);

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
	const handleRefresh = async () => {
		if (refreshing) return;
		refreshing = true;
		await run(() => status.refresh());
		refreshing = false;
	};

	// The stored status may be stale (another client or autoshutdown), so
	// always check again when a world page opens
	onMount(handleRefresh);
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

		<div class="flex items-center gap-1">
			{#if serverState === 'stopped'}
				<AsyncButton action={handlePlay}>Play {world}</AsyncButton>
			{:else if isUp && activeWorld !== world}
				<button class="btn btn-lg min-h-12 btn-neutral sm:btn-md" disabled>Play {world}</button>
			{/if}
			{#if serverState}
				<button
					class="btn btn-square btn-ghost btn-sm"
					aria-label="Refresh server status"
					title="Refresh"
					disabled={refreshing}
					onclick={handleRefresh}
				>
					{#if refreshing}
						<Spinner />
					{:else}
						<svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
							<path
								fill-rule="evenodd"
								d="M15.312 11.424a5.5 5.5 0 01-9.201 2.466l-.312-.311h2.433a.75.75 0 000-1.5H3.989a.75.75 0 00-.75.75v4.242a.75.75 0 001.5 0v-2.43l.31.31a7 7 0 0011.712-3.138.75.75 0 00-1.449-.39zm1.23-3.723a.75.75 0 00.219-.53V2.929a.75.75 0 00-1.5 0V5.36l-.31-.31A7 7 0 003.239 8.188a.75.75 0 101.448.389A5.5 5.5 0 0113.89 6.11l.311.31h-2.432a.75.75 0 000 1.5h4.243a.75.75 0 00.53-.219z"
								clip-rule="evenodd"
							/>
						</svg>
					{/if}
				</button>
			{/if}
		</div>
	</div>

	{#if error}
		<p class="px-4 pb-4 text-error">{error}</p>
	{/if}
</div>
