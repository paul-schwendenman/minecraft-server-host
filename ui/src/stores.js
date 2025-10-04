import { writable, derived } from 'svelte/store';
import { getStatus, startInstance, stopInstance, syncDnsRecord, getDetails } from './data.service';

function createStatus() {
	const { subscribe, set } = writable(null);

	return {
        subscribe,
		refresh: async() => {
            set(await reducer());
        },
        dispatch: async (action) => {
            set(await reducer(action));
        }
	};
}

async function reducer(action) {
    switch(action) {
        case "startInstance":
            await startInstance();

            return getStatus();
        case "stopInstance":
            await stopInstance();

            return getStatus();
        case "syncDnsRecord":
            await syncDnsRecord();

            return getStatus();
        default:
            return getStatus()
    }
}

export const status = createStatus();

export const details = derived(status, $status => $status.instance.state === "running" && getDetails($status.instance.ip_address));
