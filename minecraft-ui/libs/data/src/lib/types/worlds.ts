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
}

export interface WorldDetail extends World {
    last_rendered?: string;
}
