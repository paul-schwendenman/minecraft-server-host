import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

// Use vi.hoisted to define mocks before vi.mock is hoisted
const { mockStatus, mockDetails } = vi.hoisted(() => {
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
		mockStatus: {
			...createMockStore({
				instance: { state: 'stopped', ip_address: null },
				dns_record: {}
			}),
			refresh: vi.fn(() => Promise.resolve()),
			dispatch: vi.fn(() => Promise.resolve())
		},
		mockDetails: createMockStore(null)
	};
});

vi.mock('@minecraft/data', () => ({
	status: mockStatus,
	details: mockDetails,
	getStatus: vi.fn(),
	startInstance: vi.fn(),
	stopInstance: vi.fn(),
	syncDnsRecord: vi.fn(),
	getDetails: vi.fn()
}));

// Import component after mock is set up
import ServerStatus from './ServerStatus.svelte';

describe('ServerStatus', () => {
	beforeEach(() => {
		mockStatus.set({
			instance: { state: 'stopped', ip_address: null },
			dns_record: {}
		});
	});

	describe('base', () => {
		it('displays DNS name as heading', async () => {
			mockStatus.set({
				instance: { state: 'stopped', ip_address: null },
				dns_record: { name: 'example.test' }
			});

			const screen = render(ServerStatus);
			await expect
				.element(screen.getByRole('heading', { name: 'example.test' }))
				.toBeInTheDocument();
		});

		it('displays instance state', async () => {
			mockStatus.set({
				instance: { state: 'terminated' },
				dns_record: {}
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByText('Server is terminated.')).toBeInTheDocument();
		});

		it('has refresh button', async () => {
			mockStatus.set({
				instance: { state: 'stopped' },
				dns_record: {}
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument();
		});
	});

	describe('running', () => {
		it('has a stop button', async () => {
			mockStatus.set({
				instance: { state: 'running' },
				dns_record: {}
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByRole('button', { name: 'Stop' })).toBeInTheDocument();
		});

		it('displays the current IP address', async () => {
			mockStatus.set({
				instance: { state: 'running', ip_address: '10.0.0.1' },
				dns_record: {}
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByText(/IP address:/)).toBeInTheDocument();
		});
	});

	describe('stopped', () => {
		it('has a start button', async () => {
			mockStatus.set({
				instance: { state: 'stopped' },
				dns_record: {}
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByRole('button', { name: 'Start' })).toBeInTheDocument();
		});
	});

	describe('mismatched DNS', () => {
		it('allows updating of DNS record', async () => {
			mockStatus.set({
				instance: { state: 'running', ip_address: '10.0.0.1' },
				dns_record: { value: '10.0.0.2' }
			});

			const screen = render(ServerStatus);
			await expect.element(screen.getByRole('button', { name: 'Update DNS' })).toBeInTheDocument();
		});
	});
});
