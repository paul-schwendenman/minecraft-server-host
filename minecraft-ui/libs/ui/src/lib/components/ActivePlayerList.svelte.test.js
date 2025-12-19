import { render } from 'vitest-browser-svelte';
import { describe, it, expect } from 'vitest';
import ActivePlayerList from './ActivePlayerList.svelte';

describe('ActivePlayerList', () => {
	it('handles no active users', async () => {
		const { container } = render(ActivePlayerList);

		await expect.element(container).toHaveTextContent('');
	});

	it('handles one active user', async () => {
		const samplePlayers = [{ id: '1', name: 'Bob' }];
		const screen = render(ActivePlayerList, { players: samplePlayers });

		await expect.element(screen.getByRole('listitem')).toBeInTheDocument();
		await expect.element(screen.getByText('Bob')).toBeInTheDocument();
	});

	it('handles multiple active users', async () => {
		const samplePlayers = [
			{ id: '1', name: 'Bob' },
			{ id: '2', name: 'Bill' }
		];
		const screen = render(ActivePlayerList, { players: samplePlayers });

		const items = screen.getByRole('listitem').all();
		expect(items).toHaveLength(2);
		await expect.element(screen.getByText('Bob')).toBeInTheDocument();
		await expect.element(screen.getByText('Bill')).toBeInTheDocument();
	});
});
