export async function getStatus() {
    const resp = await fetch('__baseUrl__/status');

    if (resp.ok) {
        return await resp.json();
    } else {
        throw Error(await resp.text())
    }
}

export async function stopInstance() {
    const resp = await fetch('__baseUrl__/stop');
    const text = await resp.text();

    if (resp.ok) {
        return text;
    } else {
        throw Error(text);
    }
}

export async function startInstance() {
    const resp = await fetch('__baseUrl__/start');
    const text = await resp.text();

    if (resp.ok) {
        return text;
    } else {
        throw Error(text);
    }
}
export async function syncDnsRecord() {
    const resp = await fetch('__baseUrl__/syncdns');
    const text = await resp.text();

    if (resp.ok) {
        return text;
    } else {
        throw Error(text);
    }
}
