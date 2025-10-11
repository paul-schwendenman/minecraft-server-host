import type { ServerStatusResponse } from '../types/api.js';
const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function getStatus(): Promise<ServerStatusResponse> {
    const resp = await fetch(`${API_BASE}/status`);

    if (!resp.ok) {
        throw new Error(await resp.text());
    }

    return resp.json();
}
