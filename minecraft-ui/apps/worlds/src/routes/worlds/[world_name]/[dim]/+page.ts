import type { PageLoad } from './$types';
import { getWorldMap } from '@minecraft/data';

export const prerender = false;

export const load: PageLoad = async ({ fetch, params }) => {
	const { world_name: worldName, dim: mapName } = params;

	const map = await getWorldMap(worldName, mapName, fetch);

	if (!map) {
		throw new Error(`Map '${mapName}' not found for world '${worldName}'`);
	}

	return {
		world: { name: worldName },
		map,
		preview: map.previewUrl,
		mapUrl: map.mapUrl
	};
};
