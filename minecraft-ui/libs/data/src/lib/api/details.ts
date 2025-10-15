import type { ServerDetailsResponse } from '../types/api.js';
import { apiUrl } from './config.js';

/**
 * Fetch detailed Minecraft server info, optionally for a given hostname.
 *
 * @param hostname - optional server hostname (if querying remote server)
 * @returns JSON response with version, description, and player info.
 * @throws Error if the API call fails.
 */
export async function getDetails(hostname?: string, fetchFn: typeof fetch = fetch): Promise<ServerDetailsResponse> {
	const url = new URL(apiUrl('details'), window.location.origin);

	if (hostname) {
		url.search = new URLSearchParams({ hostname }).toString();
	}

	const resp = await fetchFn(url.toString());

	if (!resp.ok) {
		throw new Error(await resp.text());
	}

	return resp.json() as Promise<ServerDetailsResponse>;
}
