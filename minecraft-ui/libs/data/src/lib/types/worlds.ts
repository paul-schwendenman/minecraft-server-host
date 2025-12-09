export interface Dimension {
	name: string;
	id: number;
	previewUrl: string;
	mapUrl: string;
}

export interface World {
	world: string;
	previewUrl: string;
	mapUrl: string;
	dimensions: Dimension[];
	version: string;
}

export interface WorldDetail extends World {
	last_rendered?: string;
}
