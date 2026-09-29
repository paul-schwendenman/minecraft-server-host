<script lang="ts">
	import WorldMenuButton from './WorldMenuButton.svelte';
	import { status, playersOnline } from '@minecraft/data';

	// Stop, plus a menu for switching to a different world (server running).
	// Switching is only offered with nobody online; the API refuses it too.
	let { class: className = '' }: { class?: string } = $props();

	const disabledReason = $derived(
		$playersOnline === null
			? "Checking who's online…"
			: $playersOnline === 1
				? 'Switch worlds once the server is empty (1 player online)'
				: $playersOnline > 1
					? `Switch worlds once the server is empty (${$playersOnline} players online)`
					: ''
	);

	const handleSwitch = async (world: string) => {
		await status.switchWorld(world);
		return `Switching to ${world}. It'll be ready in a minute or two.`;
	};
</script>

<WorldMenuButton
	class={className}
	label="Stop"
	action={() => status.dispatch('stopInstance')}
	menuLabel="Switch to a different world"
	menuTitle="Switch to a world"
	onPick={handleSwitch}
	disableCurrent
	{disabledReason}
/>
