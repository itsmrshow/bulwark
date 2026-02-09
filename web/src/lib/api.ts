const API_BASE = import.meta.env.VITE_API_BASE ?? "";

export async function login(token: string) {
  const response = await fetch(`${API_BASE}/api/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ token }),
    credentials: "include" // Important: send cookies
  });

  if (!response.ok) {
    let details = "";
    try {
      const data = await response.json();
      details = data?.error ?? response.statusText;
    } catch {
      details = response.statusText;
    }
    throw new Error(details);
  }

  return await response.json();
}

export async function logout() {
  const response = await fetch(`${API_BASE}/api/logout`, {
    method: "POST",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error("Logout failed");
  }

  return await response.json();
}

export async function apiFetch<T>(path: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers ?? {});
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
    credentials: "include", // Important: send session cookies
    cache: options.cache ?? "no-store"
  });

  if (!response.ok) {
    let details = "";
    try {
      const data = await response.json();
      details = data?.error ?? response.statusText;
    } catch {
      details = response.statusText;
    }
    throw new Error(details);
  }

  if (response.status === 204) {
    return null as T;
  }

  return (await response.json()) as T;
}
