const backendPort = new URLSearchParams(window.location.search).get('backendPort')
export const apiBase = backendPort ? `http://127.0.0.1:${backendPort}` : ''

export class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
  ) {
    super(message)
  }
}

export async function apiFetch<T>(url: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(apiBase + url, options)
  if (!resp.ok) {
    let msg = resp.statusText
    try {
      const body = await resp.json()
      if (body.error) msg = body.error
    } catch {
      /* ignore */
    }
    throw new ApiError(msg, resp.status)
  }
  return resp.json()
}

export async function apiPost<T>(endpoint: string, body: any): Promise<T> {
  return apiFetch<T>(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function apiPut<T>(endpoint: string, body: any): Promise<T> {
  return apiFetch<T>(endpoint, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function apiDelete<T>(endpoint: string, body: any): Promise<T> {
  return apiFetch<T>(endpoint, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

export async function apiGet<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
  let url = endpoint
  if (params) {
    const qs = new URLSearchParams()
    for (const [k, v] of Object.entries(params)) {
      if (v) qs.set(k, v)
    }
    const str = qs.toString()
    if (str) url += '?' + str
  }
  return apiFetch<T>(url)
}
