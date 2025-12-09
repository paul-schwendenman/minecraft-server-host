/**
 * Central API configuration helper.
 * Uses environment variable `VITE_API_BASE` if provided,
 * otherwise defaults to `/api`.
 *
 * Example:
 *   VITE_API_BASE=https://api.example.com
 */
export const API_BASE = import.meta.env.VITE_API_BASE || '/api';

/**
 * Helper to safely build full API URLs.
 */
export function apiUrl(path: string): string {
	// Ensure exactly one slash between base and path
	const base = API_BASE.endsWith('/') ? API_BASE.slice(0, -1) : API_BASE;
	const normalized = path.startsWith('/') ? path : `/${path}`;
	return `${base}${normalized}`;
}
