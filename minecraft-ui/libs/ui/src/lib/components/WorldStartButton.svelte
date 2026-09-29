<script lang="ts">
	import WorldMenuButton from './WorldMenuButton.svelte';
	import { status } from '@minecraft/data';

	// Start, plus a menu for starting a different world (server stopped)
	let { class: className = '' }: { class?: string } = $props();
</script>

<WorldMenuButton
	class={className}
	label="Start"
	action={() => status.dispatch('startInstance')}
	menuLabel="Start a different world"
	menuTitle="Start a world"
	onPick={(world) => status.startWorld(world)}
/>
