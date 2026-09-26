<script lang="ts">
	import { onMount } from 'svelte';
	import AsyncButton from './AsyncButton.svelte';
	import Spinner from './Spinner.svelte';
	import { status, listWorlds } from '@minecraft/data';

	let { class: className = '' }: { class?: string } = $props();

	// Empty until the world list loads, and stays empty if it can't: the plain
	// Start button must never depend on the maps API.
	let worlds: string[] = $state([]);
	let pending = $state(false);
	let error = $state('');

	const activeWorld = $derived($status.instance?.active_world);

	onMount(async () => {
		try {
			worlds = (await listWorlds()).map((w) => w.world);
		} catch {
			worlds = [];
		}
	});

	const handleStart = () => status.dispatch('startInstance');

	// The daisyUI dropdown is open while it has focus
	const closeMenu = () => (document.activeElement as HTMLElement | null)?.blur();

	const handleStartWorld = async (world: string) => {
		closeMenu();
		if (pending) return;

		pending = true;
		error = '';
		try {
			await status.startWorld(world);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			pending = false;
		}
	};
</script>

{#if worlds.length === 0}
	<AsyncButton class={className} action={handleStart}>Start</AsyncButton>
{:else}
	<!-- The menu is positioned against this group, so it matches its width -->
	<div class="relative flex {className}">
		<AsyncButton class="flex-1 rounded-r-none" action={handleStart}>Start</AsyncButton>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="dropdown static dropdown-end dropdown-top md:dropdown-bottom"
			onkeydown={(e) => e.key === 'Escape' && closeMenu()}
		>
			<div
				tabindex="0"
				role="button"
				aria-label="Start a different world"
				class="btn btn-lg min-h-12 rounded-l-none border-l-base-content/10 btn-neutral sm:btn-md"
			>
				{#if pending}
					<Spinner />
				{:else}
					<svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
						<path
							fill-rule="evenodd"
							d="M5.23 7.21a.75.75 0 011.06.02L10 11.17l3.71-3.94a.75.75 0 111.08 1.04l-4.25 4.5a.75.75 0 01-1.08 0l-4.25-4.5a.75.75 0 01.02-1.06z"
							clip-rule="evenodd"
						/>
					</svg>
				{/if}
			</div>
			<ul
				tabindex="-1"
				class="dropdown-content menu z-10 mb-1 md:mt-1 md:mb-0 max-h-72 w-full flex-nowrap overflow-y-auto rounded-box border border-base-300 bg-base-100 p-2 shadow"
			>
				<li class="menu-title">Start a world</li>
				{#each worlds as world (world)}
					<li>
						<button onclick={() => handleStartWorld(world)} disabled={pending}>
							<span class="flex-1 text-left">{world}</span>
							{#if world === activeWorld}
								<span class="text-xs text-base-content/60">current</span>
							{/if}
						</button>
					</li>
				{/each}
			</ul>
		</div>
	</div>
{/if}

{#if error}
	<p class="w-full text-error">{error}</p>
{/if}
