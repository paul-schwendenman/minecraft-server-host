const mockWorlds = [
	{
		world: 'default',
		previewUrl: '/mock/maps/default/preview.png',
		mapUrl: '/mock/maps/default/',
		dimensions: [
			{ name: 'overworld', id: 0, previewUrl: '/mock/maps/default/overworld.png', mapUrl: '/mock/maps/default/overworld/' },
			{ name: 'nether', id: -1, previewUrl: '/mock/maps/default/nether.png', mapUrl: '/mock/maps/default/nether/' },
			{ name: 'end', id: 1, previewUrl: '/mock/maps/default/end.png', mapUrl: '/mock/maps/default/end/' }
		]
	},
	{
		world: 'creative',
		previewUrl: '/mock/maps/creative/preview.png',
		mapUrl: '/mock/maps/creative/',
		dimensions: [{ name: 'overworld', id: 0, previewUrl: '/mock/maps/creative/overworld.png', mapUrl: '/mock/maps/creative/overworld/' }]
	}
];

// Helper for mock dimension details
function mockDimensions(worldId) {
	return mockWorlds.find((w) => w.world === worldId)?.dimensions || [];
}

export function mockServer() {
	return {
		name: 'mock-server',
		configureServer(server) {
			console.log('[mock-server] plugin active');

			// ---------- Helpers ----------
			const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
			const ipv4 = {
				random(subnet, mask) {
					const rand = Math.floor(Math.random() * Math.pow(2, 32 - mask)) + 1;
					return this.lon2ip(this.ip2lon(subnet) | rand);
				},
				ip2lon(addr) {
					return addr.split('.').reduce((a, o) => (a << 8) + +o, 0) >>> 0;
				},
				lon2ip(lon) {
					return [lon >>> 24, (lon >> 16) & 255, (lon >> 8) & 255, lon & 255].join('.');
				}
			};

			// ---------- State ----------
			let errorThreshold = 0.9;
			const state = {
				instance: { state: 'stopped', ip_address: null },
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

			const send = (res, obj) => {
				res.setHeader('Content-Type', 'application/json');
				res.end(JSON.stringify(obj));
			};

			// ---------- Routes ----------
			server.middlewares.use(async (req, res, next) => {
				if (!req.url.startsWith('/api/')) return next();

				console.log('[mock]', req.method, req.url);
				const path = req.url.replace(/^\/api/, '').split('?')[0];

				if (path === '/start') {
					state.instance.state = 'pending';
					await sleep(250);
					setTimeout(() => {
						state.instance.state = 'running';
						state.instance.ip_address = ipv4.random('10.0.0.0', 8);
					}, 1000);
					return send(res, { message: 'Success' });
				}

				if (path === '/stop') {
					state.instance.state = 'stopping';
					await sleep(500);
					setTimeout(() => {
						state.instance.state = 'stopped';
						state.instance.ip_address = null;
					}, 5000);
					errorThreshold = 0.9;
					return send(res, { message: 'Success' });
				}

				if (path === '/syncdns') {
					state.dns_record.value = state.instance.ip_address;
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
						res.statusCode = [500, 503, 504][Math.floor(Math.random() * 3)];
						return res.end();
					}
					await sleep(333);
					const sample = [details.zero, details.one, details.two];
					return send(res, sample[Math.floor(Math.random() * 3)]);
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

				// /worlds/{name}/{dimension}
				const dimMatch = path.match(/^\/worlds\/([^/]+)\/([^/]+)$/);
				if (dimMatch) {
					const [_, worldId, dim] = dimMatch;
					const dimData = mockDimensions(worldId).find((d) => d.name === dim);
					if (!dimData) return send(res, { error: 'Dimension not found' }, 404);
					return send(res, dimData);
				}

				next();
			});
		}
	};
}
