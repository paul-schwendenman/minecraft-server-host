<script lang="ts">
	import { textureUrl } from '$lib/data';
	import type { Block, Face } from '$lib/palette';

	interface Props {
		block: Block;
		face: Face;
		size?: number;
		class?: string;
	}

	let { block, face, size = 64, class: className = '' }: Props = $props();
</script>

<img
	src={textureUrl(block, face)}
	alt={block.name}
	title={block.name}
	width={size}
	height={size}
	loading="lazy"
	class="block-texture {className}"
/>

<style>
	.block-texture {
		image-rendering: pixelated;
		/* Checkerboard behind see-through blocks */
		background: repeating-conic-gradient(#0000001a 0 25%, transparent 0 50%) 0 0 / 25% 25%;
	}
</style>
