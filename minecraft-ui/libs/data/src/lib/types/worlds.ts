export interface MapInfo {
	name: string;
	dimension: string;
	previewUrl: string;
	mapUrl: string;
}

export interface World {
	world: string;
	previewUrl: string;
	mapUrl: string;
	maps: MapInfo[];
	version: string;
}

export interface WorldDetail extends World {
	last_rendered?: string;
}
