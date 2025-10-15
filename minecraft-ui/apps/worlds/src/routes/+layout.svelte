<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';
	// import { page } from '$app/stores';
	import { page } from '$app/state';

	let { children } = $props();
</script>

<main class="bg-base-100 text-base-content flex min-h-screen flex-col">
	<nav class="navbar bg-base-200 border-base-300 border-b shadow-sm">
		<div class="mx-auto flex w-full max-w-5xl items-center justify-between px-4">
			<a href={resolve('/')} class="btn btn-ghost text-xl normal-case">
				<span class="hidden sm:inline">Minecraft Server Manager</span>
				<span class="inline sm:hidden">MC Server Manager</span>
			</a>

			<div class="flex gap-2">
				<a
					href={resolve('/')}
					class="btn btn-ghost btn-sm"
					class:btn-active={page.url.pathname === '/'}
				>
					Controls
				</a>
				<a
					href={resolve('/worlds')}
					class="btn btn-ghost btn-sm"
					class:btn-active={page.url.pathname.startsWith('/worlds')}
				>
					Maps
				</a>
			</div>
		</div>
	</nav>

	<section class="mx-auto flex w-full max-w-5xl flex-1 flex-col p-4">
		{@render children?.()}
	</section>
</main>
