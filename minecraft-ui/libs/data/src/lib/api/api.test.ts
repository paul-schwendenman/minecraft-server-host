import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { getStatus } from './status.js';
import { startInstance } from './start.js';
import { stopInstance } from './stop.js';
import { syncDnsRecord } from './syncDns.js';
import { getDetails } from './details.js';
import { listWorlds, getWorld, getWorldMap } from './worlds.js';
import { apiUrl } from './config.js';

// Mock fetch helper for JSON responses
function createMockFetch(response: unknown, ok = true, status = 200) {
	return vi.fn().mockResolvedValue({
		ok,
		status,
		json: () => Promise.resolve(response),
		text: () => Promise.resolve(typeof response === 'string' ? response : JSON.stringify(response))
	});
}

// Mock fetch helper for text responses
function createMockFetchText(response: string, ok = true, status = 200) {
	return vi.fn().mockResolvedValue({
		ok,
		status,
		text: () => Promise.resolve(response)
	});
}

// Setup window.location for tests that need it
beforeEach(() => {
	// @ts-expect-error - mocking window for Node environment
	global.window = { location: { origin: 'http://localhost:3000' } };
});

afterEach(() => {
	// @ts-expect-error - cleanup
	delete global.window;
});

describe('apiUrl helper', () => {
	it('builds URL with path starting with slash', () => {
		const result = apiUrl('/status');
		expect(result).toContain('/status');
	});

	it('builds URL with path not starting with slash', () => {
		const result = apiUrl('status');
		expect(result).toContain('/status');
	});

	it('handles trailing slash in base', () => {
		// This tests the normalization logic
		const result = apiUrl('/test');
		expect(result).not.toContain('//test');
	});
});

describe('getStatus', () => {
	it('returns server status on success', async () => {
		const mockResponse = {
			instance: { state: 'running', ip_address: '10.0.0.1' },
			dns_record: { name: 'minecraft.example.com', value: '10.0.0.1', type: 'A' }
		};
		const mockFetch = createMockFetch(mockResponse);

		const result = await getStatus(mockFetch);

		expect(result).toEqual(mockResponse);
		expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/status'));
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetch('Server error', false, 500);

		await expect(getStatus(mockFetch)).rejects.toThrow();
	});
});

describe('startInstance', () => {
	it('calls start endpoint', async () => {
		const mockFetch = createMockFetchText('Success');

		const result = await startInstance(mockFetch);

		expect(result).toBe('Success');
		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/start'),
			expect.objectContaining({ method: 'POST' })
		);
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetchText('Failed to start', false, 500);

		await expect(startInstance(mockFetch)).rejects.toThrow();
	});
});

describe('stopInstance', () => {
	it('calls stop endpoint', async () => {
		const mockFetch = createMockFetchText('Success');

		const result = await stopInstance(mockFetch);

		expect(result).toBe('Success');
		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/stop'),
			expect.objectContaining({ method: 'POST' })
		);
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetchText('Failed to stop', false, 500);

		await expect(stopInstance(mockFetch)).rejects.toThrow();
	});
});

describe('syncDnsRecord', () => {
	it('calls syncdns endpoint', async () => {
		const mockFetch = createMockFetchText('Success');

		const result = await syncDnsRecord(mockFetch);

		expect(result).toBe('Success');
		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/syncdns'),
			expect.objectContaining({ method: 'POST' })
		);
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetchText('Failed to sync', false, 500);

		await expect(syncDnsRecord(mockFetch)).rejects.toThrow();
	});
});

describe('getDetails', () => {
	it('returns server details on success', async () => {
		const mockResponse = {
			version: { name: '1.20.4', protocol: 765 },
			players: { max: 20, online: 5, sample: [] }
		};
		const mockFetch = createMockFetch(mockResponse);

		const result = await getDetails('10.0.0.1', mockFetch);

		expect(result).toEqual(mockResponse);
		expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/details'));
	});

	it('includes hostname in query params', async () => {
		const mockResponse = { version: { name: '1.20.4' }, players: {} };
		const mockFetch = createMockFetch(mockResponse);

		await getDetails('minecraft.example.com', mockFetch);

		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('hostname=minecraft.example.com')
		);
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetch('Server offline', false, 503);

		await expect(getDetails('10.0.0.1', mockFetch)).rejects.toThrow();
	});
});

describe('listWorlds', () => {
	it('returns list of worlds on success', async () => {
		const mockResponse = [
			{ world: 'survival', name: 'Survival World', previewUrl: 'url1', mapUrl: 'url2' },
			{ world: 'creative', name: 'Creative World', previewUrl: 'url3', mapUrl: 'url4' }
		];
		const mockFetch = createMockFetch(mockResponse);

		const result = await listWorlds(mockFetch);

		expect(result).toEqual(mockResponse);
		expect(result).toHaveLength(2);
		expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/worlds'));
	});

	it('throws error on failed response', async () => {
		const mockFetch = createMockFetch('Not found', false, 404);

		await expect(listWorlds(mockFetch)).rejects.toThrow();
	});
});

describe('getWorld', () => {
	it('returns world details on success', async () => {
		const mockResponse = {
			name: 'Survival World',
			previewUrl: 'url',
			maps: [{ name: 'overworld', dimension: 'minecraft:overworld' }]
		};
		const mockFetch = createMockFetch(mockResponse);

		const result = await getWorld('survival', mockFetch);

		expect(result).toEqual(mockResponse);
		expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/worlds/survival'));
	});

	it('throws error for non-existent world', async () => {
		const mockFetch = createMockFetch('World not found', false, 404);

		await expect(getWorld('nonexistent', mockFetch)).rejects.toThrow();
	});
});

describe('getWorldMap', () => {
	it('returns map info on success', async () => {
		const mockResponse = {
			name: 'overworld',
			dimension: 'minecraft:overworld',
			previewUrl: 'preview',
			mapUrl: 'map'
		};
		const mockFetch = createMockFetch(mockResponse);

		const result = await getWorldMap('survival', 'overworld', mockFetch);

		expect(result).toEqual(mockResponse);
		expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/worlds/survival/overworld'));
	});

	it('throws error for non-existent map', async () => {
		const mockFetch = createMockFetch('Map not found', false, 404);

		await expect(getWorldMap('survival', 'nonexistent', mockFetch)).rejects.toThrow();
	});
});
