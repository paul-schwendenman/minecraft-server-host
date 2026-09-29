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
				? '1 player is online. You can switch once the server is empty.'
				: $playersOnline > 1
					? `${$playersOnline} players are online. You can switch once the server is empty.`
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
