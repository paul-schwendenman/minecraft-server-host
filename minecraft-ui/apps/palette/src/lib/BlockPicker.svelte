<script lang="ts">
	import BlockTile from '$lib/BlockTile.svelte';
	import { sortByHue, type Block, type Face } from '$lib/palette';

	interface Props {
		/** Blocks to choose from */
		blocks: Block[];
		face: Face;
		title: string;
		open: boolean;
		onpick: (block: Block) => void;
	}

	let { blocks, face, title, open = $bindable(), onpick }: Props = $props();

	let search = $state('');

	const sorted = $derived(sortByHue(blocks, face));
	const shown = $derived.by(() => {
		const q = search.trim().toLowerCase();
		return q ? sorted.filter((b) => b.name.toLowerCase().includes(q)) : sorted;
	});

	function syncOpen(dialog: HTMLDialogElement) {
		if (open && !dialog.open) dialog.showModal();
		else if (!open && dialog.open) dialog.close();
	}

	function onclose() {
		open = false;
		search = '';
	}

	function pick(block: Block) {
		onpick(block);
		open = false;
	}
</script>

<dialog {@attach syncOpen} class="modal" {onclose}>
	<div class="modal-box flex max-h-[85vh] max-w-3xl flex-col gap-3">
		<div class="flex items-center justify-between gap-2">
			<h2 class="text-lg font-bold">{title}</h2>
			<form method="dialog">
				<button class="btn btn-circle btn-ghost btn-sm" aria-label="Close">✕</button>
			</form>
		</div>
		<!-- svelte-ignore a11y_autofocus -->
		<input
			type="search"
			class="input w-full"
			placeholder="Search {blocks.length} blocks…"
			bind:value={search}
			autofocus
		/>
		<div class="grid grid-cols-[repeat(auto-fill,minmax(3rem,1fr))] gap-1 overflow-y-auto">
			{#each shown as block (block.id)}
				<button
					class="rounded p-0.5 hover:bg-base-300 focus:bg-base-300 focus:outline-2 focus:outline-primary"
					onclick={() => pick(block)}
					title={block.name}
				>
					<BlockTile {block} {face} size={44} class="h-auto w-full" />
				</button>
			{:else}
				<p class="col-span-full py-6 text-center text-base-content/60">No blocks match.</p>
			{/each}
		</div>
	</div>
	<form method="dialog" class="modal-backdrop">
		<button>close</button>
	</form>
</dialog>
