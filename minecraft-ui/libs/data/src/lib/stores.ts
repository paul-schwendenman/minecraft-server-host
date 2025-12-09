import { writable, derived } from 'svelte/store';
import { getStatus, startInstance, stopInstance, syncDnsRecord, getDetails } from './api/index.js';
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
