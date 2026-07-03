import { apiPost, apiGet, apiPut, apiDelete } from './client'
import type {
  Server,
  SavedRequest,
  SavedRequestDetail,
  HistoryEntry,
  HistoryEntryDetail,
  HistoryListParams,
  HistoryListResponse,
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

export const getSavedRequest = (id: number) =>
  apiGet<SavedRequestDetail>('/api/saved-requests/detail', { id })

export const createSavedRequest = (
  name: string,
  server_id: number,
  method: string,
  payload: string,
) => apiPost<SavedRequestDetail>('/api/saved-requests', { name, server_id, method, payload })

export const updateSavedRequest = (
  id: number,
  server_id: number,
  method: string,
  payload: string,
) => apiPut<{ ok: boolean }>('/api/saved-requests', { id, server_id, method, payload })

export const deleteSavedRequest = (id: number) =>
  apiDelete<{ ok: boolean }>('/api/saved-requests', { id })

// History
export const getHistory = async (params?: HistoryListParams): Promise<HistoryListResponse> => {
  const response = await apiGet<HistoryListResponse | HistoryEntry[]>('/api/history', params)

  if (Array.isArray(response)) {
    return {
      items: response,
      total: response.length,
      limit: response.length,
      offset: 0,
    }
  }

  return {
    items: Array.isArray(response.items) ? response.items : [],
    total: typeof response.total === 'number' ? response.total : 0,
    limit: typeof response.limit === 'number' ? response.limit : 0,
    offset: typeof response.offset === 'number' ? response.offset : 0,
  }
}

export const getHistoryEntry = (id: number) =>
  apiGet<HistoryEntryDetail>('/api/history/detail', { id })

export const createHistory = (
  server_id: number,
  method: string,
  payload: string,
  response: string,
  status_code: number,
) =>
  apiPost<HistoryEntryDetail>('/api/history', {
    server_id,
    method,
    payload,
    response,
    status_code,
  })

export const deleteHistory = (id: number) => apiDelete<{ ok: boolean }>('/api/history', { id })

export const clearHistory = () => apiDelete<{ ok: boolean }>('/api/history', { all: true })
