const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function stopInstance(): Promise<string> {
	const resp = await fetch(`${API_BASE}/stop`);

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}
