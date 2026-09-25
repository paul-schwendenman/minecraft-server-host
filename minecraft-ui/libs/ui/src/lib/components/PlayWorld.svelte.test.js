import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { mockStatus } = vi.hoisted(() => {
	let value = {};
	/** @type {Set<(v: any) => void>} */
	const subscribers = new Set();
	return {
		mockStatus: {
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
			refresh: vi.fn(() => Promise.resolve()),
			startWorld: vi.fn(() => Promise.resolve())
		}
	};
});

vi.mock('@minecraft/data', () => ({
	status: mockStatus
}));

import PlayWorld from './PlayWorld.svelte';

/**
 * @param {string} state
 * @param {string} [activeWorld]
 */
const setServer = (state, activeWorld = 'default') =>
	mockStatus.set({ instance: { state, active_world: activeWorld }, dns_record: {} });

describe('PlayWorld', () => {
	beforeEach(() => {
		mockStatus.refresh.mockClear();
		mockStatus.startWorld.mockReset();
		mockStatus.startWorld.mockImplementation(() => Promise.resolve());
	});

	it('refreshes the status when none is loaded yet', async () => {
		mockStatus.set({});

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText('Checking the server…')).toBeInTheDocument();
		expect(mockStatus.refresh).toHaveBeenCalled();
	});

	it('starts the world when the server is off', async () => {
		setServer('stopped');

		const screen = render(PlayWorld, { world: 'old' });
		await screen.getByRole('button', { name: 'Play old' }).click();

		expect(mockStatus.startWorld).toHaveBeenCalledWith('old');
	});

	it('shows the world as running when it is the active world', async () => {
		setServer('running', 'old');

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText('is running.')).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).not.toBeInTheDocument();
	});

	it('disables Play while another world is running', async () => {
		setServer('running', 'default');

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText('Stop it to play this world.')).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).toBeDisabled();
	});

	it('shows the API error when starting fails', async () => {
		setServer('stopped');
		mockStatus.startWorld.mockImplementation(() =>
			Promise.reject(new Error('Server is running default; stop it before switching worlds'))
		);

		const screen = render(PlayWorld, { world: 'old' });
		await screen.getByRole('button', { name: 'Play old' }).click();

		await expect
			.element(screen.getByText('Server is running default; stop it before switching worlds'))
			.toBeInTheDocument();
	});

	it('offers a refresh while the server is stopping', async () => {
		setServer('stopping');

		const screen = render(PlayWorld, { world: 'old' });
		await screen.getByRole('button', { name: 'Refresh' }).click();

		expect(mockStatus.refresh).toHaveBeenCalled();
	});
});
