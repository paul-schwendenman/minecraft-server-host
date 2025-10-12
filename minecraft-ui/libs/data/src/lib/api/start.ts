const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function startInstance(): Promise<string> {
	const resp = await fetch(`${API_BASE}/start`);

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}
