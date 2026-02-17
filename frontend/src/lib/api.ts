const BASE = '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export const api = {
  catches: {
    list: () => request<any[]>('/catches'),
    get: (id: number) => request<any>(`/catches/${id}`),
    create: (data: any) => request<any>('/catches', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: any) =>
      request<any>(`/catches/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request<void>(`/catches/${id}`, { method: 'DELETE' }),
    addPhotos: (id: number, formData: FormData) =>
      fetch(`${BASE}/catches/${id}/photos`, { method: 'POST', body: formData }).then((r) =>
        r.json(),
      ),
  },
  species: {
    list: (q?: string) => request<any[]>(`/species${q ? `?q=${encodeURIComponent(q)}` : ''}`),
    create: (data: any) => request<any>('/species', { method: 'POST', body: JSON.stringify(data) }),
  },
  weather: {
    get: (lat: number, lon: number) => request<any>(`/weather?lat=${lat}&lon=${lon}`),
  },
  settings: {
    get: () => request<any>('/settings'),
    update: (data: any) => request<any>('/settings', { method: 'PUT', body: JSON.stringify(data) }),
  },
  stats: {
    get: () => request<any>('/stats'),
  },
  trips: {
    list: () => request<any[]>('/trips'),
    get: (id: number) => request<any>(`/trips/${id}`),
    create: (data: any) => request<any>('/trips', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: any) =>
      request<any>(`/trips/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request<void>(`/trips/${id}`, { method: 'DELETE' }),
  },
  export: {
    json: () => `${BASE}/export?format=json`,
    csv: () => `${BASE}/export?format=csv`,
  },
};
