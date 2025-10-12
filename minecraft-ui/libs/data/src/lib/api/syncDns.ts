const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export async function syncDnsRecord(): Promise<string> {
	const resp = await fetch(`${API_BASE}/syncdns`);

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.text();
}
