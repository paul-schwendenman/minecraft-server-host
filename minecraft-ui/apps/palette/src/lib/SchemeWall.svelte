<script lang="ts">
	import BlockTile from '$lib/BlockTile.svelte';
	import Icon from '$lib/Icon.svelte';
	import type { Face } from '$lib/palette';
	import { ROLE_LABELS, type SchemeGroup } from '$lib/scheme';

	interface Props {
		groups: SchemeGroup[];
		face: Face;
		/** Whether another block may be added (palette size cap) */
		canAdd: boolean;
		onadd: (groupIndex: number) => void;
		onremove: (groupIndex: number, blockIndex: number) => void;
	}

	let { groups, face, canAdd, onadd, onremove }: Props = $props();

	const total = $derived(groups.reduce((n, g) => n + g.blocks.length, 0));
	// Controls only appear on hover/focus, like coolors.co; always on touch screens
	const reveal =
		'opacity-0 transition-opacity focus-visible:opacity-100 pointer-coarse:opacity-100';
</script>

<!-- Seamless within a group, with a gap between groups so roles read as units -->
<div class="flex gap-1.5 overflow-x-auto">
	{#each groups as group, gi (group.role)}
		<!-- Zero basis so every tile in the wall gets the same width -->
		<section class="flex min-w-0 flex-col gap-1" style:flex="{group.blocks.length} 1 0%">
			<h3 class="truncate text-xs font-medium text-base-content/60">
				{ROLE_LABELS[group.role]}
			</h3>
			<div class="relative flex overflow-hidden rounded-md border border-base-300">
				{#each group.blocks as block, bi (block.id)}
					<div class="group relative min-w-0 flex-1">
						<BlockTile {block} {face} size={128} class="h-auto w-full" />
						<BlockTile {block} {face} size={128} class="h-auto w-full" />
						{#if total > 2}
							<button
								class="{reveal} btn absolute top-1.5 left-1/2 btn-circle -translate-x-1/2 shadow btn-sm group-hover:opacity-100"
								aria-label="Remove {block.name}"
								title="Remove {block.name}"
								onclick={() => onremove(gi, bi)}
							>
								<Icon name="trash" />
							</button>
						{/if}
					</div>
				{/each}
				{#if canAdd}
					<!-- Hover strip along the group's right edge -->
					<div
						class="group/add absolute inset-y-0 right-0 z-10 flex w-10 items-center justify-center"
					>
						<button
							class="{reveal} btn btn-circle shadow-md btn-md group-hover/add:opacity-100"
							aria-label="Add another {ROLE_LABELS[group.role].toLowerCase()} block"
							title="Add another {ROLE_LABELS[group.role].toLowerCase()} block"
							onclick={() => onadd(gi)}
						>
							<Icon name="plus" class="size-5" />
						</button>
					</div>
				{/if}
			</div>
		</section>
	{/each}
</div>
