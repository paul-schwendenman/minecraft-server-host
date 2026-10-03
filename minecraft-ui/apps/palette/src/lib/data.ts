import { asset } from '$app/paths';
import type { Asset } from '$app/types';
import type { Block, BlockData, Face } from './palette';

/** Minecraft version of the generated data in static/data (see `pnpm data`) */
export const DATA_VERSION = '26.3';

function dataUrl(path: string): string {
	// Generated files aren't known to SvelteKit's typed asset list
	return asset(`/data/${DATA_VERSION}/${path}` as Asset);
}

export async function loadBlocks(fetch: typeof globalThis.fetch): Promise<BlockData> {
	const res = await fetch(dataUrl('blocks.json'));
	if (!res.ok) {
		throw new Error(`Failed to load block data (${res.status}). Did you run \`pnpm data\`?`);
	}
	return res.json();
}

export function textureUrl(block: Block, face: Face): string {
	return dataUrl(`textures/${block.faces[face].texture}`);
}
