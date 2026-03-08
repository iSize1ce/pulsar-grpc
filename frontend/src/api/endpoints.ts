import { apiPost, apiGet, apiPut, apiDelete } from './client'
import type {
  Server,
  SavedRequest,
  HistoryEntry,
  ServiceInfo,
  InvokeResponse,
  MetaEntry,
} from '@/types/api'
import type { ProtoField } from '@/types/proto'

// gRPC reflection
export const fetchServices = (url: string) => apiPost<ServiceInfo[]>('/api/services', { url })

export const fetchDescribe = (url: string, service: string, method: string) =>
  apiPost<ProtoField[]>('/api/describe', { url, service, method })

export const invokeMethod = (
  url: string,
  service: string,
  method: string,
  payload: any,
  meta: MetaEntry[],
) => apiPost<InvokeResponse>('/api/invoke', { url, service, method, payload, meta })

// Servers
export const getServers = (q?: string) => apiGet<Server[]>('/api/servers', q ? { q } : undefined)

export const createServer = (url: string, name: string) =>
  apiPost<Server>('/api/servers', { url, name })

export const updateServerMeta = (id: number, meta: string) =>
  apiPut<{ ok: boolean }>('/api/servers', { id, meta })

export const deleteServer = (id: number) => apiDelete<{ ok: boolean }>('/api/servers', { id })

// Saved requests
export const getSavedRequests = (q?: string, method?: string) =>
  apiGet<SavedRequest[]>('/api/saved-requests', { q: q || '', method: method || '' })

export const createSavedRequest = (
  name: string,
  server_id: number,
  method: string,
  payload: string,
) => apiPost<SavedRequest>('/api/saved-requests', { name, server_id, method, payload })

export const updateSavedRequest = (
  id: number,
  server_id: number,
  method: string,
  payload: string,
) => apiPut<{ ok: boolean }>('/api/saved-requests', { id, server_id, method, payload })

export const deleteSavedRequest = (id: number) =>
  apiDelete<{ ok: boolean }>('/api/saved-requests', { id })

// History
export const getHistory = (q?: string, method?: string) =>
  apiGet<HistoryEntry[]>('/api/history', { q: q || '', method: method || '' })

export const createHistory = (
  server_id: number,
  method: string,
  payload: string,
  response: string,
  status_code: number,
) => apiPost<HistoryEntry>('/api/history', { server_id, method, payload, response, status_code })

export const deleteHistory = (id: number) => apiDelete<{ ok: boolean }>('/api/history', { id })

export const clearHistory = () => apiDelete<{ ok: boolean }>('/api/history', { all: true })
