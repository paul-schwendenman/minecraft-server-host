import { render } from 'vitest-browser-svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';

const { mockStatus, mockListWorlds } = vi.hoisted(() => {
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
			dispatch: vi.fn(() => Promise.resolve()),
			startWorld: vi.fn(() => Promise.resolve())
		},
		mockListWorlds: vi.fn()
	};
});

vi.mock('@minecraft/data', () => ({
	status: mockStatus,
	listWorlds: mockListWorlds
}));

import StartButton from './StartButton.svelte';

const worlds = ['default', 'old', 'world.bak2'].map((world) => ({ world }));

describe('StartButton', () => {
	beforeEach(() => {
		mockStatus.set({ instance: { state: 'stopped', active_world: 'default' }, dns_record: {} });
		mockStatus.dispatch.mockClear();
		mockStatus.startWorld.mockReset();
		mockStatus.startWorld.mockImplementation(() => Promise.resolve());
		mockListWorlds.mockReset();
		mockListWorlds.mockImplementation(() => Promise.resolve(worlds));
	});

	it('names the world the plain Start launches', async () => {
		const screen = render(StartButton);

		await screen.getByRole('button', { name: 'Start default' }).click();

		expect(mockStatus.dispatch).toHaveBeenCalledWith('startInstance');
	});

	it('says just Start when the active world is unknown', async () => {
		mockStatus.set({ instance: { state: 'stopped' }, dns_record: {} });

		const screen = render(StartButton);

		await expect
			.element(screen.getByRole('button', { name: 'Start', exact: true }))
			.toBeInTheDocument();
	});

	it('shows a plain Start without a dropdown when the world list fails', async () => {
		mockListWorlds.mockImplementation(() => Promise.reject(new Error('maps API down')));

		const screen = render(StartButton);

		await expect.element(screen.getByRole('button', { name: 'Start default' })).toBeInTheDocument();
		await expect
			.element(screen.getByRole('button', { name: 'Start a different world' }))
			.not.toBeInTheDocument();
	});

	it('lists the worlds and marks the current one', async () => {
		const screen = render(StartButton);

		await screen.getByRole('button', { name: 'Start a different world' }).click();

		await expect.element(screen.getByRole('button', { name: 'old' })).toBeInTheDocument();
		await expect.element(screen.getByRole('button', { name: 'world.bak2' })).toBeInTheDocument();
		await expect
			.element(screen.getByRole('button', { name: 'default current' }))
			.toBeInTheDocument();
	});

	it('starts the chosen world straight away', async () => {
		const screen = render(StartButton);

		await screen.getByRole('button', { name: 'Start a different world' }).click();
		await screen.getByRole('button', { name: 'old' }).click();

		expect(mockStatus.startWorld).toHaveBeenCalledWith('old');
		expect(mockStatus.dispatch).not.toHaveBeenCalled();
	});

	it('shows the API error when starting a world fails', async () => {
		mockStatus.startWorld.mockImplementation(() =>
			Promise.reject(new Error('Server is stopping; try again once it has stopped'))
		);

		const screen = render(StartButton);
		await screen.getByRole('button', { name: 'Start a different world' }).click();
		await screen.getByRole('button', { name: 'old' }).click();

		await expect
			.element(screen.getByText('Server is stopping; try again once it has stopped'))
			.toBeInTheDocument();
	});
});
