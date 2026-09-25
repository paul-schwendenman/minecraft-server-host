const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function startInstance(fetchFn: typeof fetch = fetch): Promise<string> {
	const resp = await fetchFn(`${API_BASE}/start`, { method: 'POST' });

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}

/**
 * Start the server into a given world. The control API rejects this (409) when
 * the server is already running a different world.
 */
export async function startWorld(world: string, fetchFn: typeof fetch = fetch): Promise<string> {
	const params = new URLSearchParams({ world });
	const resp = await fetchFn(`${API_BASE}/start?${params}`, { method: 'POST' });

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
