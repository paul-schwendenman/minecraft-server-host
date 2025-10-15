<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<main class="mx-auto flex min-h-screen max-w-5xl flex-col">
	<nav class="flex justify-center space-x-6 p-4 text-sm">
		<a href={resolve('/')} class="hover:text-gray-200">Server</a>
		<a href={resolve('/worlds')} class="hover:text-gray-200">Maps</a>
	</nav>
	{@render children?.()}
</main>
