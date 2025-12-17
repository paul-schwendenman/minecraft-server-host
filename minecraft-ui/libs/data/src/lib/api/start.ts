const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function startInstance(fetchFn: typeof fetch = fetch): Promise<string> {
	const resp = await fetchFn(`${API_BASE}/start`, { method: 'POST' });

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}
