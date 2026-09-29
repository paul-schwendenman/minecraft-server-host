import { describe, it, expect, vi, beforeEach } from 'vitest';

const { getStatus, getDetails } = vi.hoisted(() => ({
	getStatus: vi.fn(),
	getDetails: vi.fn()
}));

vi.mock('./api/index.js', () => ({
	getStatus,
	getDetails,
	startInstance: vi.fn(),
	startWorld: vi.fn(),
	switchWorld: vi.fn(),
	stopInstance: vi.fn(),
	syncDnsRecord: vi.fn()
}));

import { status, playersOnline } from './stores.js';

const running = { instance: { state: 'running', ip_address: '203.0.113.7' }, dns_record: {} };

/** Subscribe, wait for pending pings to settle, and return the latest value */
async function settled() {
	let value: number | null = -1;
	const unsubscribe = playersOnline.subscribe((v) => (value = v));
	await new Promise((r) => setTimeout(r, 0));
	unsubscribe();
	return value;
}

describe('playersOnline', () => {
	beforeEach(() => {
		getStatus.mockReset();
		getDetails.mockReset();
	});

	it('is the ping count while the server runs', async () => {
		getStatus.mockResolvedValue(running);
		getDetails.mockResolvedValue({ players: { online: 3, max: 20 } });
		await status.refresh();

		expect(await settled()).toBe(3);
		expect(getDetails).toHaveBeenCalledWith('203.0.113.7');
	});

	it('is null while the server is stopped', async () => {
		getStatus.mockResolvedValue({ instance: { state: 'stopped' }, dns_record: {} });
		await status.refresh();

		expect(await settled()).toBeNull();
		expect(getDetails).not.toHaveBeenCalled();
	});

	it('is null when the ping fails', async () => {
		getStatus.mockResolvedValue(running);
		getDetails.mockRejectedValue(new Error('Server Timeout'));
		await status.refresh();

		expect(await settled()).toBeNull();
	});
});
