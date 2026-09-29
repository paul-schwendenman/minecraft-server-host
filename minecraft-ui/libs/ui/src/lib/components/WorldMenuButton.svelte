<script lang="ts">
	import { onMount } from 'svelte';
	import AsyncButton from './AsyncButton.svelte';
	import Spinner from './Spinner.svelte';
	import { status, listWorlds } from '@minecraft/data';

	// A split button: a main action, plus a menu of the published worlds to act
	// on instead (WorldStartButton, WorldSwitchButton).
	let {
		label,
		action,
		menuLabel,
		menuTitle,
		onPick,
		disableCurrent = false,
		disabledReason = '',
		class: className = ''
	}: {
		/** Main button text */
		label: string;
		/** Main button action */
		action: () => Promise<unknown>;
		/** Accessible name of the menu toggle */
		menuLabel: string;
		/** Heading inside the menu */
		menuTitle: string;
		/** Acts on the chosen world; may return a message to show afterwards */
		onPick: (world: string) => Promise<string | void>;
		/** Disable the active world's entry (nothing to do for it) */
		disableCurrent?: boolean;
		/** When set, every entry is disabled and this says why */
		disabledReason?: string;
		class?: string;
	} = $props();

	// Empty until the world list loads, and stays empty if it can't: the main
	// button must never depend on the maps API.
	let worlds: string[] = $state([]);
	let pending = $state(false);
	let error = $state('');
	let notice = $state('');
	// The status the message was shown against: it's hidden once the status
	// updates (the next Refresh), so "Switching to…" doesn't linger
	let messageStatus: unknown = $state.raw(null);
	const showMessage = $derived($status === messageStatus);

	const activeWorld = $derived($status.instance?.active_world);

	onMount(async () => {
		try {
			worlds = (await listWorlds()).map((w) => w.world);
		} catch {
			worlds = [];
		}
	});

	// The daisyUI dropdown is open while it has focus
	const closeMenu = () => (document.activeElement as HTMLElement | null)?.blur();

	const handlePick = async (world: string) => {
		closeMenu();
		if (pending) return;

		pending = true;
		error = '';
		notice = '';
		try {
			notice = (await onPick(world)) ?? '';
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			messageStatus = $status;
			pending = false;
		}
	};
</script>

{#if worlds.length === 0}
	<AsyncButton class={className} {action}>{label}</AsyncButton>
{:else}
	<!-- The menu is positioned against this group, so it matches its width -->
	<div class="relative flex {className}">
		<AsyncButton class="flex-1 rounded-r-none" {action}>{label}</AsyncButton>
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<!-- Anchored to the group's left edge and at least 15rem wide, so the menu
		     stays readable when the button is narrow -->
		<div
			class="dropdown static dropdown-start dropdown-top md:dropdown-bottom"
			onkeydown={(e) => e.key === 'Escape' && closeMenu()}
		>
			<div
				tabindex="0"
				role="button"
				aria-label={menuLabel}
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
				class="dropdown-content menu z-10 mb-1 md:mt-1 md:mb-0 max-h-72 w-full min-w-60 flex-nowrap overflow-y-auto rounded-box border border-base-300 bg-base-100 p-2 shadow"
			>
				<li class="menu-title">{menuTitle}</li>
				{#if disabledReason}
					<li class="px-4 pb-2 text-sm text-base-content/70">{disabledReason}</li>
				{/if}
				{#each worlds as world (world)}
					<li class={{ 'menu-disabled': !!disabledReason }}>
						<button
							onclick={() => handlePick(world)}
							disabled={pending || !!disabledReason || (disableCurrent && world === activeWorld)}
						>
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

<!-- order-last: in a flex row of buttons, the message goes below all of them
     rather than splitting the row -->
{#if showMessage && error}
	<p class="order-last w-full text-error">{error}</p>
{:else if showMessage && notice}
	<p class="order-last w-full">{notice}</p>
{/if}
