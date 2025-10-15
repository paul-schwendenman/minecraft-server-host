import type { PageLoad } from './$types';
import { getWorldDimension } from '@minecraft/data';

export const prerender = false;

export const load: PageLoad = async ({ fetch, params }) => {
    const { world_name: worldName, dim: dimName } = params;

    const dimension = await getWorldDimension(worldName, dimName, fetch);

    if (!dimension) {
        throw new Error(`Dimension '${dimName}' not found for world '${worldName}'`);
    }

    return {
        world: { name: worldName },
        dimension,
        preview: dimension.previewUrl,
        mapUrl: dimension.mapUrl
    };
};
