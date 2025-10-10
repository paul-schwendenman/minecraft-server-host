import type { ServerDetailsResponse } from '../types/api.js';

/**
 * Fetch detailed Minecraft server info, optionally for a given hostname.
 *
 * @param hostname - optional server hostname (if querying remote server)
 * @returns JSON response with version, description, and player info.
 * @throws Error if the API call fails.
 */
export async function getDetails(hostname?: string): Promise<ServerDetailsResponse> {
    const url = new URL('/api/details', window.location.origin);

    if (hostname) {
        url.search = new URLSearchParams({ hostname }).toString();
    }

    const resp = await fetch(url.toString());

    if (!resp.ok) {
        throw new Error(await resp.text());
    }

    return resp.json() as Promise<ServerDetailsResponse>;
}
