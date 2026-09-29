import type { ServerStatusResponse } from '@minecraft/data';
import type { Plugin } from 'vite';
import type { IncomingMessage, ServerResponse } from 'http';

const mockWorlds = [
	{
		world: 'default',
		previewUrl: '/mock/maps/default/preview.png',
		mapUrl: '/mock/maps/default/',
		maps: [
			{
				name: 'overworld',
				dimension: 'overworld',
				previewUrl: '/mock/maps/default/overworld.png',
				mapUrl: '/mock/maps/default/overworld/'
			},
			{
				name: 'nether',
				dimension: 'nether',
				previewUrl: '/mock/maps/default/nether.png',
				mapUrl: '/mock/maps/default/nether/'
			},
			{
				name: 'end',
				dimension: 'end',
				previewUrl: '/mock/maps/default/end.png',
				mapUrl: '/mock/maps/default/end/'
			}
		]
	},
	{
		world: 'creative',
		previewUrl: '/mock/maps/creative/preview.png',
		mapUrl: '/mock/maps/creative/',
		maps: [
			{
				name: 'overworld',
				dimension: 'overworld',
				previewUrl: '/mock/maps/creative/overworld.png',
				mapUrl: '/mock/maps/creative/overworld/'
			}
		]
	}
];

// Helper for mock map details
function mockMaps(worldId: string) {
	return mockWorlds.find((w) => w.world === worldId)?.maps || [];
}

export function mockServer(): Plugin {
	return {
		name: 'mock-server',
		configureServer(server) {
			console.log('[mock-server] plugin active');

			// ---------- Helpers ----------
			const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
			const ipv4 = {
				random(subnet: string, mask: number) {
					const rand = Math.floor(Math.random() * Math.pow(2, 32 - mask)) + 1;
					return this.lon2ip(this.ip2lon(subnet) | rand);
				},
				ip2lon(addr: string) {
					return addr.split('.').reduce((a, o) => (a << 8) + +o, 0) >>> 0;
				},
				lon2ip(lon: number) {
					return [lon >>> 24, (lon >> 16) & 255, (lon >> 8) & 255, lon & 255].join('.');
				}
			};

			// ---------- State ----------
			let errorThreshold = 0.9;
			// Players in the last details response; /switch refuses while any are online
			let lastPlayersOnline: number | null = null;
			const state: ServerStatusResponse = {
				instance: { state: 'stopped', ip_address: undefined, active_world: 'default' },
				dns_record: { name: 'minecraft.example.com.', value: '10.0.0.1', type: 'A' }
			};

			const details = {
				zero: {
					description: { text: 'A Minecraft Server' },
					players: { max: 20, online: 0 },
					version: { name: '1.15.2', protocol: 578 }
				},
				one: {
					description: { text: 'A Minecraft Server' },
					players: {
						max: 20,
						online: 1,
						sample: [{ id: 'cdce37cd-2215-42ef-a4a4-c8b9189c9259', name: 'example' }]
					},
					version: { name: '1.15.2', protocol: 578 }
				},
				two: {
					description: { text: 'A Minecraft Server' },
					players: {
						max: 20,
						online: 2,
						sample: [
							{ id: 'cdce37cd-2215-42ef-a4a4-c8b9189c9259', name: 'example' },
							{ id: 'd720a93f-da90-41fa-8653-d09d81fa4b77', name: 'example2' }
						]
					},
					version: { name: '1.15.2', protocol: 578 }
				}
			};

			const send = (res: ServerResponse, obj: unknown, statusCode?: number) => {
				res.setHeader('Content-Type', 'application/json');
				if (statusCode) {
					res.statusCode = statusCode;
				}
				res.end(JSON.stringify(obj));
			};

			// ---------- Routes ----------
			server.middlewares.use(
				async (
					req: IncomingMessage & { url?: string; method?: string },
					res: ServerResponse,
					next: (err?: unknown) => void
				) => {
					if (!req.url?.startsWith('/api/')) return next();

					console.log('[mock]', req.method, req.url);
					const path = req.url.replace(/^\/api/, '').split('?')[0];

					if (path === '/start') {
						const world = new URL(req.url, 'http://mock').searchParams.get('world');
						if (world) {
							const up = ['pending', 'running'].includes(state.instance.state);
							if (up && world !== state.instance.active_world) {
								return send(
									res,
									{
										detail: `Server is running ${state.instance.active_world}; switch worlds instead of starting`
									},
									409
								);
							}
							if (up) return send(res, { message: 'Success', world });
							state.instance.active_world = world;
						}
						state.instance.state = 'pending';
						await sleep(250);
						setTimeout(() => {
							state.instance.state = 'running';
							state.instance.ip_address = ipv4.random('10.0.0.0', 8) as string;
						}, 1000);
						return send(res, { message: 'Success' });
					}

					if (path === '/switch') {
						const world = new URL(req.url, 'http://mock').searchParams.get('world') ?? '';
						if (!mockWorlds.some((w) => w.world === world)) {
							return send(res, { detail: `Unknown world: ${world}` }, 400);
						}
						if (state.instance.state === 'pending') {
							return send(res, { detail: "Server is starting; try again once it's running" }, 409);
						}
						if (state.instance.state !== 'running') {
							return send(
								res,
								{ detail: `Server is ${state.instance.state}; start it instead of switching` },
								409
							);
						}
						if (world === state.instance.active_world) {
							return send(res, { message: 'Success', world });
						}
						if (lastPlayersOnline === null) {
							return send(
								res,
								{ detail: "Server isn't answering yet; try again once the world has loaded" },
								409
							);
						}
						if (lastPlayersOnline > 0) {
							const who =
								lastPlayersOnline === 1 ? '1 player is' : `${lastPlayersOnline} players are`;
							return send(res, { detail: `${who} online; switch once the server is empty` }, 409);
						}
						await sleep(250);
						state.instance.active_world = world;
						return send(res, { message: 'Switching', world }, 202);
					}

					if (path === '/stop') {
						state.instance.state = 'stopping';
						await sleep(500);
						setTimeout(() => {
							state.instance.state = 'stopped';
							state.instance.ip_address = undefined;
						}, 5000);
						errorThreshold = 0.9;
						return send(res, { message: 'Success' });
					}

					if (path === '/syncdns') {
						state.dns_record.value = state.instance.ip_address ?? '';
						await sleep(250);
						setTimeout(() => (errorThreshold = 0.2), 1000);
						return send(res, { message: 'Success' });
					}

					if (path === '/status') {
						await sleep(100);
						return send(res, state);
					}

					if (path.startsWith('/details')) {
						const url = new URL(req.url, 'http://localhost');
						const hostname = url.searchParams.get('hostname');
						if (!hostname) {
							res.statusCode = 400;
							return res.end();
						}
						if (Math.random() < errorThreshold) {
							lastPlayersOnline = null;
							res.statusCode = [500, 503, 504][Math.floor(Math.random() * 3)];
							return res.end();
						}
						await sleep(333);
						const sample = [details.zero, details.one, details.two];
						const reply = sample[Math.floor(Math.random() * 3)];
						lastPlayersOnline = reply.players.online;
						return send(res, reply);
					}

					if (path === '/worlds') {
						return send(res, mockWorlds);
					}

					// /worlds/{name}
					const worldMatch = path.match(/^\/worlds\/([^/]+)$/);
					if (worldMatch) {
						const worldId = worldMatch[1];
						const world = mockWorlds.find((w) => w.world === worldId);
						if (!world) return send(res, { error: 'World not found' }, 404);
						return send(res, world);
					}

					// /worlds/{name}/{map}
					const mapMatch = path.match(/^\/worlds\/([^/]+)\/([^/]+)$/);
					if (mapMatch) {
						const [, worldId, mapName] = mapMatch;
						const mapData = mockMaps(worldId).find((m) => m.name === mapName);
						if (!mapData) return send(res, { error: 'Map not found' }, 404);
						return send(res, mapData);
					}

					// An unmocked API route would otherwise fall through to the app's HTML
					return send(res, { detail: `No mock for ${req.method} ${path}` }, 404);
				}
			);
		}
	};
}
