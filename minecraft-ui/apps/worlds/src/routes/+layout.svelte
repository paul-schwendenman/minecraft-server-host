<script lang="ts">
	import '../app.css';
	import { resolve } from '$app/paths';
	// import { page } from '$app/stores';
	import { page } from '$app/state';

	let { children } = $props();
</script>

<main class="flex min-h-screen flex-col bg-base-100 text-base-content">
	<nav class="navbar border-b border-base-300 bg-base-200 shadow-sm">
		<div class="mx-auto flex w-full max-w-5xl items-center justify-between px-4">
			<a href={resolve('/')} class="btn text-xl normal-case btn-ghost">
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

	<section class="mx-auto flex w-full max-w-5xl flex-1 flex-col">
		{@render children?.()}
	</section>
</main>
