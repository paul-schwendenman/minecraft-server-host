import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

// Use vi.hoisted to define mocks before vi.mock is hoisted
const { mockDetails, mockStatus } = vi.hoisted(() => {
	/** @param {any} initialValue */
	const createMockStore = (initialValue) => {
		let value = initialValue;
		/** @type {Set<(v: any) => void>} */
		const subscribers = new Set();
		return {
			/** @param {(v: any) => void} callback */
			subscribe: (callback) => {
				subscribers.add(callback);
				callback(value);
				return () => subscribers.delete(callback);
			},
			/** @param {any} newValue */
			set: (newValue) => {
				value = newValue;
				subscribers.forEach((callback) => callback(value));
			},
			/** @param {(v: any) => any} fn */
			update: (fn) => {
				value = fn(value);
				subscribers.forEach((callback) => callback(value));
			}
		};
	};
	return {
		mockDetails: createMockStore(null),
		mockStatus: {
			...createMockStore({ instance: { state: 'stopped' }, dns_record: {} }),
			refresh: vi.fn(() => Promise.resolve()),
			dispatch: vi.fn(() => Promise.resolve())
		}
	};
});

vi.mock('@minecraft/data', () => ({
	details: mockDetails,
	status: mockStatus,
	getStatus: vi.fn(),
	startInstance: vi.fn(),
	stopInstance: vi.fn(),
	syncDnsRecord: vi.fn(),
	getDetails: vi.fn()
}));

// Import component after mock is set up
import ServerDetails from './ServerDetails.svelte';

describe('ServerDetails', () => {
	beforeEach(() => {
		mockDetails.set(null);
	});

	describe('server details returned successfully', () => {
		it('handles no active users', async () => {
			mockDetails.set(
				Promise.resolve({
					players: { max: 20, online: 0 },
					version: { name: '1.15.2' }
				})
			);

			const screen = render(ServerDetails);
			await expect
				.element(screen.getByText('The server has no active players.'))
				.toBeInTheDocument();
		});

		it('handles 1 active user', async () => {
			mockDetails.set(
				Promise.resolve({
					players: {
						max: 20,
						online: 1,
						sample: [{ id: 'cdce37cd', name: 'example' }]
					},
					version: { name: '1.15.2' }
				})
			);

			const screen = render(ServerDetails);
			await expect.element(screen.getByText('The server has 1 active player:')).toBeInTheDocument();
			await expect.element(screen.getByText('example')).toBeInTheDocument();
		});

		it('handles multiple active users', async () => {
			mockDetails.set(
				Promise.resolve({
					players: {
						max: 20,
						online: 2,
						sample: [
							{ id: '1', name: 'Alice' },
							{ id: '2', name: 'Bob' }
						]
					},
					version: { name: '1.15.2' }
				})
			);

			const screen = render(ServerDetails);
			await expect
				.element(screen.getByText('The server has 2 active players:'))
				.toBeInTheDocument();
			await expect.element(screen.getByText('Alice')).toBeInTheDocument();
			await expect.element(screen.getByText('Bob')).toBeInTheDocument();
		});
	});

	describe('server details are still loading', () => {
		it('displays loading message', async () => {
			const pending = new Promise(() => {});
			mockDetails.set(pending);

			const screen = render(ServerDetails);
			await expect.element(screen.getByText('Loading details...')).toBeInTheDocument();
		});
	});

	describe('server details failed to load', () => {
		it('displays an error message', async () => {
			mockDetails.set(Promise.reject('Error'));

			const screen = render(ServerDetails);
			await expect.element(screen.getByText('Failed to load details.')).toBeInTheDocument();
		});
	});
});
