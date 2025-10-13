import type { World, WorldDetail, Dimension } from '$lib/types';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

/** Fetch the list of all worlds */
export async function listWorlds(): Promise<World[]> {
    const resp = await fetch(`${API_BASE}/worlds`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}

/** Fetch details for a specific world */
export async function getWorld(name: string): Promise<WorldDetail> {
    const resp = await fetch(`${API_BASE}/worlds/${name}`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}

/** Fetch manifest for a specific dimension */
export async function getWorldDimension(world: string, dim: string): Promise<Dimension> {
    const resp = await fetch(`${API_BASE}/worlds/${world}/${dim}`);
    if (!resp.ok) {
        throw new Error(await resp.text());
    }
    return resp.json();
}
