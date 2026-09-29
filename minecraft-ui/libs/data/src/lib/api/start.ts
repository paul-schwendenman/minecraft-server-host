const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function startInstance(fetchFn: typeof fetch = fetch): Promise<string> {
	const resp = await fetchFn(`${API_BASE}/start`, { method: 'POST' });

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}

/**
 * Start a stopped server into a given world. The API rejects this (409) when
 * the server is running a different world; that's switchWorld.
 */
export async function startWorld(world: string, fetchFn: typeof fetch = fetch): Promise<string> {
	return postWorld('start', world, fetchFn);
}

/**
 * Switch a running server to another world. The API accepts it (202) and the
 * instance switches within a minute, or rejects it (409) while players are
 * online, the world is still loading, or the server isn't running.
 */
export async function switchWorld(world: string, fetchFn: typeof fetch = fetch): Promise<string> {
	return postWorld('switch', world, fetchFn);
}

async function postWorld(
	path: 'start' | 'switch',
	world: string,
	fetchFn: typeof fetch
): Promise<string> {
	const params = new URLSearchParams({ world });
	const resp = await fetchFn(`${API_BASE}/${path}?${params}`, { method: 'POST' });

	if (!resp.ok) {
		const text = await resp.text();
		let message = text;
		try {
			// FastAPI errors are {"detail": "..."}
			message = JSON.parse(text).detail ?? text;
		} catch {
			// not JSON; use the raw text
		}
		throw new Error(message);
	}

	return resp.text();
}
