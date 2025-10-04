<script>
	export let players = [];
</script>

<style>
</style>

<ul class="list-disc pl-8 mb-4 space-y-2">
    {#each players as player}
        <li>{player?.name}</li>
    {/each}
</ul>
