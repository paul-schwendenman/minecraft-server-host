<script lang="ts">
	import AsyncButton from './AsyncButton.svelte';
	import { status } from '@minecraft/data';

	// Starts whatever world the ActiveWorld tag names. WorldStartButton adds a
	// menu for choosing the world first.
	let { class: className = '' }: { class?: string } = $props();

	const handleStart = () => status.dispatch('startInstance');
</script>

<AsyncButton class={className} action={handleStart}>Start</AsyncButton>
