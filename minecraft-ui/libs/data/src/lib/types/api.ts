export interface Player {
    id: string;
    name: string;
}

export interface ServerVersion {
    name: string;
    protocol: number;
}

export interface ServerPlayers {
    max: number;
    online: number;
    sample?: Player[];
}

export interface ServerDetailsResponse {
    version: ServerVersion;
    description: string;
    players: ServerPlayers;
}
