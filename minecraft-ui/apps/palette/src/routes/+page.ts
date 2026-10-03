import { error } from '@sveltejs/kit';
import { loadBlocks } from '$lib/data';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	try {
		return { blockData: await loadBlocks(fetch) };
	} catch (e) {
		error(500, (e as Error).message);
	}
};
