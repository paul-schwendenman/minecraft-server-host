<script lang="ts">
	export let version: string;
</script>

{#if version}
	<p>Minecraft version: {version}</p>
{/if}
