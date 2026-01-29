const API_BASE = import.meta.env.VITE_API_BASE ?? "";
const TOKEN_KEY = "bulwark.web.token";

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string) {
  if (!token) {
    localStorage.removeItem(TOKEN_KEY);
    return;
  }
  localStorage.setItem(TOKEN_KEY, token);
}

export async function apiFetch<T>(path: string, options: RequestInit = {}) {
  const headers = new Headers(options.headers ?? {});
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers
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
