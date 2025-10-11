<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<main class="flex min-h-screen flex-col bg-gray-900 text-gray-100">
	<nav class="flex justify-center space-x-6 p-4 text-sm text-gray-400">
		<a href="/" class="hover:text-gray-200">Server</a>
		<a href="/worlds" class="hover:text-gray-200">Maps</a>
	</nav>
	{@render children?.()}
</main>
