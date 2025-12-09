<script lang="ts">
	import type { Player } from '@minecraft/data';

	export let players: Player[] = [];
</script>

{#if players && players.length > 0}
	<ul class="mb-4 list-disc space-y-2 pl-8">
		{#each players as player (player.id)}
			<li>{player.name}</li>
		{/each}
	</ul>
{/if}
