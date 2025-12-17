export async function getStatus() {
  const resp = await fetch("/api/status");

  if (resp.ok) {
    return await resp.json();
  } else {
    throw Error(await resp.text());
  }
}

export async function stopInstance() {
  const resp = await fetch("/api/stop", { method: "POST" });
  const text = await resp.text();

  if (resp.ok) {
    return text;
  } else {
    throw Error(text);
  }
}

export async function startInstance() {
  const resp = await fetch("/api/start", { method: "POST" });
  const text = await resp.text();

  if (resp.ok) {
    return text;
  } else {
    throw Error(text);
  }
}
export async function syncDnsRecord() {
  const resp = await fetch("/api/syncdns", { method: "POST" });
  const text = await resp.text();

  if (resp.ok) {
    return text;
  } else {
    throw Error(text);
  }
}

export async function getDetails(hostname) {
  const url = new URL("/api/details");

  if (hostname) {
    const params = { hostname };

    url.search = new URLSearchParams(params).toString();
  }

  const resp = await fetch(url);

  if (resp.ok) {
    return await resp.json();
  } else {
    throw Error(await resp.text());
  }
}
