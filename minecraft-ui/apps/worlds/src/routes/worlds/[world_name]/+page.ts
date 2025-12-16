import type { PageLoad } from './$types';
import { getWorld } from '@minecraft/data';

export const prerender = false;

export const load: PageLoad = async ({ params, fetch }) => {
	const { world_name: worldName } = params;

	// Fetch world info & maps
	const world = await getWorld(worldName, fetch);

	if (!world) {
		throw new Error(`World '${worldName}' not found`);
	}

	return {
		world,
		maps: world.maps || []
	};
};
