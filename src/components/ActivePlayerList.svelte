<script>
	import ActivePlayerMessage from './ActivePlayerMessage.svelte';

	export let players = [];
</script>

<style>
</style>

<ul>
    {#each players as player}
        <li>{player.name}</li>
    {/each}
</ul>
