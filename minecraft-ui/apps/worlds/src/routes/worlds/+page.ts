import type { PageLoad } from './$types';
import { listWorlds } from '@minecraft/data';

export const prerender = false;

export const load: PageLoad = async ({ fetch }) => {
	const worlds = await listWorlds(fetch);
	return { worlds };
};
