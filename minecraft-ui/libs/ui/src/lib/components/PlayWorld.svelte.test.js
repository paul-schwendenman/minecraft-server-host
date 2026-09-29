import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { mockStatus, mockDetails } = vi.hoisted(() => {
	let value = {};
	/** @type {Set<(v: any) => void>} */
	const subscribers = new Set();
	/** @type {any} */
	let detailsValue = null;
	/** @type {Set<(v: any) => void>} */
	const detailsSubscribers = new Set();
	return {
		mockDetails: {
			/** @param {(v: any) => void} callback */
			subscribe: (callback) => {
				detailsSubscribers.add(callback);
				callback(detailsValue);
				return () => detailsSubscribers.delete(callback);
			},
			/** @param {any} newValue */
			set: (newValue) => {
				detailsValue = newValue;
				detailsSubscribers.forEach((callback) => callback(detailsValue));
			}
		},
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
			startWorld: vi.fn(() => Promise.resolve()),
			switchWorld: vi.fn(() => Promise.resolve())
		}
	};
});

vi.mock('@minecraft/data', () => ({
	status: mockStatus,
	details: mockDetails
}));

import PlayWorld from './PlayWorld.svelte';

/**
 * @param {string} state
 * @param {string} [activeWorld]
 */
const setServer = (state, activeWorld = 'default') =>
	mockStatus.set({ instance: { state, active_world: activeWorld }, dns_record: {} });

/** @param {number} online players the server list ping reports */
const setPlayers = (online) => mockDetails.set(Promise.resolve({ players: { online, max: 20 } }));

describe('PlayWorld', () => {
	beforeEach(() => {
		mockStatus.refresh.mockClear();
		mockStatus.startWorld.mockReset();
		mockStatus.startWorld.mockImplementation(() => Promise.resolve());
		mockStatus.switchWorld.mockReset();
		mockStatus.switchWorld.mockImplementation(() => Promise.resolve());
		mockDetails.set(null);
	});

	it('refreshes a previously loaded status on mount', async () => {
		setServer('running', 'default');

		render(PlayWorld, { world: 'old' });

		expect(mockStatus.refresh).toHaveBeenCalled();
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
		expect(mockStatus.switchWorld).not.toHaveBeenCalled();
	});

	it('shows the world as running when it is the active world', async () => {
		setServer('running', 'old');

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText('is running.')).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).not.toBeInTheDocument();
	});

	it('disables Play until it knows nobody is online', async () => {
		setServer('running', 'default');

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByRole('button', { name: 'Play old' })).toBeDisabled();
	});

	it('disables Play while players are online', async () => {
		setServer('running', 'default');
		setPlayers(2);

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText(/2 players online/)).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).toBeDisabled();
	});

	it('disables Play while the server is starting another world', async () => {
		setServer('pending', 'default');
		setPlayers(0);

		const screen = render(PlayWorld, { world: 'old' });

		await expect.element(screen.getByText(/Try again once it's running/)).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).toBeDisabled();
	});

	it('switches worlds when the server is running and empty', async () => {
		setServer('running', 'default');
		setPlayers(0);

		const screen = render(PlayWorld, { world: 'old' });
		await expect.element(screen.getByText(/with nobody online/)).toBeInTheDocument();
		await screen.getByRole('button', { name: 'Play old' }).click();

		expect(mockStatus.switchWorld).toHaveBeenCalledWith('old');
		expect(mockStatus.startWorld).not.toHaveBeenCalled();
		await expect.element(screen.getByText(/Switching to/)).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'Play old' })).not.toBeInTheDocument();
	});

	it('shows the API error when a switch is refused', async () => {
		setServer('running', 'default');
		setPlayers(0);
		mockStatus.switchWorld.mockImplementation(() =>
			Promise.reject(new Error('1 player is online; switch once the server is empty'))
		);

		const screen = render(PlayWorld, { world: 'old' });
		await screen.getByRole('button', { name: 'Play old' }).click();

		await expect
			.element(screen.getByText('1 player is online; switch once the server is empty'))
			.toBeInTheDocument();
		await expect.element(screen.getByText(/Switching to/)).not.toBeInTheDocument();
	});

	it('shows the API error when starting fails', async () => {
		setServer('stopped');
		mockStatus.startWorld.mockImplementation(() =>
			Promise.reject(new Error('Server is running default; switch worlds instead of starting'))
		);

		const screen = render(PlayWorld, { world: 'old' });
		await screen.getByRole('button', { name: 'Play old' }).click();

		await expect
			.element(screen.getByText('Server is running default; switch worlds instead of starting'))
			.toBeInTheDocument();
	});

	it.each(['running', 'stopped', 'stopping'])('offers a refresh while %s', async (state) => {
		setServer(state);

		const screen = render(PlayWorld, { world: 'old' });
		await expect.poll(() => mockStatus.refresh).toHaveBeenCalledTimes(1);
		await screen.getByRole('button', { name: 'Refresh server status' }).click();

		expect(mockStatus.refresh).toHaveBeenCalledTimes(2);
	});

	it('hides the refresh until the status has loaded', async () => {
		mockStatus.set({});

		const screen = render(PlayWorld, { world: 'old' });

		await expect
			.element(screen.getByRole('button', { name: 'Refresh server status' }))
			.not.toBeInTheDocument();
	});
});
