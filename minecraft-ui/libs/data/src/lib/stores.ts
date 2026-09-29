import { writable, derived } from 'svelte/store';
import {
	getStatus,
	startInstance,
	startWorld,
	switchWorld,
	stopInstance,
	syncDnsRecord,
	getDetails
} from './api/index.js';
import type { Action, ServerStatusResponse } from './types/api.ts';

function createStatus() {
	const { subscribe, set } = writable({} as ServerStatusResponse);

	return {
		subscribe,
		refresh: async () => {
			set(await reducer(null));
		},
		dispatch: async (action: Action) => {
			set(await reducer(action));
		},
		startWorld: async (world: string) => {
			await startWorld(world);
			set(await getStatus());
		},
		switchWorld: async (world: string) => {
			await switchWorld(world);
			set(await getStatus());
		}
	};
}

async function reducer(action: Action | null) {
	switch (action) {
		case 'startInstance':
			await startInstance();

			return getStatus();
		case 'stopInstance':
			await stopInstance();

			return getStatus();
		case 'syncDnsRecord':
			await syncDnsRecord();

			return getStatus();
		default:
			return getStatus();
	}
}

export const status = createStatus();

export const details = derived(status, ($status) =>
	$status?.instance?.state === 'running' ? getDetails($status.instance.ip_address) : null
);

/**
 * Players online while the server runs, from the details ping; null while
 * it's unknown (stopped, not answering yet, or still being asked). Switching
 * worlds is only offered at 0: the API refuses it while anyone's online.
 */
export const playersOnline = derived(
	details,
	($details, set) => {
		set(null);
		if (!$details) return;
		let current = true;
		$details.then((d) => current && set(d.players.online)).catch(() => {});
		return () => {
			current = false;
		};
	},
	null as number | null
);
