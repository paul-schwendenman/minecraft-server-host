<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<svelte:head>
	<title>Block Palettes</title>
</svelte:head>

<main class="flex min-h-screen flex-col bg-base-100 text-base-content">
	<nav class="navbar border-b border-base-300 bg-base-200 shadow-sm">
		<div class="mx-auto flex w-full max-w-5xl items-center px-4">
			<a href={resolve('/')} class="btn text-xl normal-case btn-ghost">Block Palettes</a>
		</div>
	</nav>

	<section class="mx-auto flex w-full max-w-5xl flex-1 flex-col px-4 py-6">
		{@render children?.()}
	</section>
</main>
