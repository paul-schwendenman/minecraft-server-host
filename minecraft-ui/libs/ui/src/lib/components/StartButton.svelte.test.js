import { render } from 'vitest-browser-svelte';
import { describe, it, expect, vi } from 'vitest';

const { mockStatus, mockListWorlds } = vi.hoisted(() => ({
	mockStatus: {
		/** @param {(v: any) => void} callback */
		subscribe: (callback) => {
			callback({ instance: { state: 'stopped' }, dns_record: {} });
			return () => {};
		},
		dispatch: vi.fn(() => Promise.resolve())
	},
	mockListWorlds: vi.fn()
}));

vi.mock('@minecraft/data', () => ({
	status: mockStatus,
	listWorlds: mockListWorlds
}));

import StartButton from './StartButton.svelte';

describe('StartButton', () => {
	it('starts the instance', async () => {
		const screen = render(StartButton);

		await screen.getByRole('button', { name: 'Start', exact: true }).click();

		expect(mockStatus.dispatch).toHaveBeenCalledWith('startInstance');
	});

	it('has no world menu and never loads the world list', async () => {
		const screen = render(StartButton);

		await expect
			.element(screen.getByRole('button', { name: 'Start', exact: true }))
			.toBeInTheDocument();
		expect(screen.container.querySelector('[aria-label="Start a different world"]')).toBeNull();
		expect(mockListWorlds).not.toHaveBeenCalled();
	});
});
