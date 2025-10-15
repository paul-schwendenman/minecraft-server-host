import type { World, WorldDetail, Dimension } from '$lib/types';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

/** Fetch the list of all worlds */
export async function listWorlds(fetchFn: typeof fetch = fetch): Promise<World[]> {
    const resp = await fetchFn(`${API_BASE}/worlds`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}

/** Fetch details for a specific world */
export async function getWorld(name: string, fetchFn: typeof fetch = fetch): Promise<WorldDetail> {
    const resp = await fetchFn(`${API_BASE}/worlds/${name}`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}

/** Fetch manifest for a specific dimension */
export async function getWorldDimension(world: string, dim: string, fetchFn: typeof fetch = fetch): Promise<Dimension> {
    const resp = await fetchFn(`${API_BASE}/worlds/${world}/${dim}`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}
