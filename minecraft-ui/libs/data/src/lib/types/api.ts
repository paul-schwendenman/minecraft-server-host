export interface DnsRecord {
    name: string;
    value: string;
    type: 'A' | 'CNAME';
}

export interface Instance {
    state: 'pending' | 'running' | 'stopping' | 'stopped' | 'terminated';
    ip_address: string | undefined;
}

export interface ServerStatusResponse {
    instance: Instance;
    dns_record: DnsRecord;
}

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

export type Action = 'startInstance' | 'stopInstance' | 'syncDnsRecord'
