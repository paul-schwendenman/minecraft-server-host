import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { mockStatus, mockPlayersOnline, mockListWorlds } = vi.hoisted(() => {
	/** @param {any} initial */
	const store = (initial) => {
		let value = initial;
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
			}
		};
	};
	return {
		mockStatus: {
			...store({}),
			dispatch: vi.fn(() => Promise.resolve()),
			switchWorld: vi.fn(() => Promise.resolve())
		},
		mockPlayersOnline: store(null),
		mockListWorlds: vi.fn()
	};
});

vi.mock('@minecraft/data', () => ({
	status: mockStatus,
	playersOnline: mockPlayersOnline,
	listWorlds: mockListWorlds
}));

import WorldSwitchButton from './WorldSwitchButton.svelte';

const worlds = ['default', 'old'].map((world) => ({ world }));

describe('WorldSwitchButton', () => {
	beforeEach(() => {
		mockStatus.set({ instance: { state: 'running', active_world: 'default' }, dns_record: {} });
		mockStatus.dispatch.mockClear();
		mockStatus.switchWorld.mockReset();
		mockStatus.switchWorld.mockImplementation(() => Promise.resolve());
		mockPlayersOnline.set(0);
		mockListWorlds.mockReset();
		mockListWorlds.mockImplementation(() => Promise.resolve(worlds));
	});

	it('stops the server from the main button', async () => {
		const screen = render(WorldSwitchButton);

		await screen.getByRole('button', { name: 'Stop', exact: true }).click();

		expect(mockStatus.dispatch).toHaveBeenCalledWith('stopInstance');
		expect(mockStatus.switchWorld).not.toHaveBeenCalled();
	});

	it('shows a plain Stop when the world list fails', async () => {
		mockListWorlds.mockImplementation(() => Promise.reject(new Error('maps API down')));

		const screen = render(WorldSwitchButton);

		await expect
			.element(screen.getByRole('button', { name: 'Stop', exact: true }))
			.toBeInTheDocument();
		await expect
			.element(screen.getByRole('button', { name: 'Switch to a different world' }))
			.not.toBeInTheDocument();
	});

	it('switches to the chosen world when nobody is online', async () => {
		const screen = render(WorldSwitchButton);

		await screen.getByRole('button', { name: 'Switch to a different world' }).click();
		await screen.getByRole('button', { name: 'old' }).click();

		expect(mockStatus.switchWorld).toHaveBeenCalledWith('old');
		await expect.element(screen.getByText(/Switching to old/)).toBeInTheDocument();
	});

	it('hides the switching message once the status refreshes', async () => {
		const screen = render(WorldSwitchButton);

		await screen.getByRole('button', { name: 'Switch to a different world' }).click();
		await screen.getByRole('button', { name: 'old' }).click();
		await expect.element(screen.getByText(/Switching to old/)).toBeInTheDocument();

		mockStatus.set({ instance: { state: 'running', active_world: 'old' }, dns_record: {} });

		await expect.element(screen.getByText(/Switching to old/)).not.toBeInTheDocument();
	});

	it("disables the running world's entry", async () => {
		const screen = render(WorldSwitchButton);

		await screen.getByRole('button', { name: 'Switch to a different world' }).click();

		await expect.element(screen.getByRole('button', { name: 'default current' })).toBeDisabled();
		await expect.element(screen.getByRole('button', { name: 'old' })).toBeEnabled();
	});

	it('disables the menu while players are online, and says why', async () => {
		mockPlayersOnline.set(2);

		const screen = render(WorldSwitchButton);
		const toggle = screen.getByRole('button', { name: 'Switch to a different world' });

		await expect.element(toggle).toBeDisabled();
		expect(toggle.element().closest('[data-tip]')?.getAttribute('data-tip')).toBe(
			'Switch worlds once the server is empty (2 players online)'
		);
		await expect.element(screen.getByRole('button', { name: 'old' })).not.toBeInTheDocument();
		// Stop still works
		await expect.element(screen.getByRole('button', { name: 'Stop', exact: true })).toBeEnabled();
	});

	it('disables the menu until the player count is known', async () => {
		mockPlayersOnline.set(null);

		const screen = render(WorldSwitchButton);
		const toggle = screen.getByRole('button', { name: 'Switch to a different world' });

		await expect.element(toggle).toBeDisabled();
		expect(toggle.element().closest('[data-tip]')?.getAttribute('data-tip')).toMatch(
			/Checking who's online/
		);
	});

	it('enables the menu once the server empties', async () => {
		mockPlayersOnline.set(1);
		const screen = render(WorldSwitchButton);
		const toggle = screen.getByRole('button', { name: 'Switch to a different world' });
		await expect.element(toggle).toBeDisabled();

		mockPlayersOnline.set(0);

		await expect.element(toggle).toBeEnabled();
	});

	it('shows the API error when a switch is refused', async () => {
		mockStatus.switchWorld.mockImplementation(() =>
			Promise.reject(new Error('1 player is online; switch once the server is empty'))
		);

		const screen = render(WorldSwitchButton);
		await screen.getByRole('button', { name: 'Switch to a different world' }).click();
		await screen.getByRole('button', { name: 'old' }).click();

		await expect
			.element(screen.getByText('1 player is online; switch once the server is empty'))
			.toBeInTheDocument();
		await expect.element(screen.getByText(/Switching to/)).not.toBeInTheDocument();
	});
});
