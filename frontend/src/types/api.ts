export interface Server {
  id: number
  url: string
  name: string
  meta: string
}

export interface MetaEntry {
  key: string
  value: string
}

export interface SavedRequest {
  id: number
  name: string
  server_id: number
  method: string
}

export interface SavedRequestDetail extends SavedRequest {
  payload: string
}

export interface HistoryEntry {
  id: number
  server_id: number
  method: string
  status_code: number
  created_at: string
  server_url?: string
  server_name?: string
}

export interface HistoryEntryDetail extends HistoryEntry {
  payload: string
  response: string
}

export type HistoryStatusFilter = 'all' | 'ok' | 'error'

export interface HistoryListParams {
  q?: string
  method?: string
  server_id?: number
  status?: Exclude<HistoryStatusFilter, 'all'>
  limit?: number
  offset?: number
}

export interface HistoryListResponse {
  items: HistoryEntry[]
  total: number
  limit: number
  offset: number
}

export interface ServiceInfo {
  name: string
  methods: MethodInfo[]
}

export interface MethodInfo {
  name: string
  input: string
  output: string
  clientStreaming: boolean
  serverStreaming: boolean
}

export interface MethodEntry {
  svc: string
  name: string
  value: string
}

export interface InvokeResponse {
  data?: any
  debug?: DebugInfo
  grpcStatus?: string
  grpcStatusCode?: number
  grpcMessage?: string
  errorDetails?: any[]
}

export interface DebugInfo {
  requestUrl: string
  method: string
  statusCode: string
  statusMessage?: string
  sentAt: string
  receivedAt: string
  duration: string
  durationMs: number
  responseSize?: number
  error?: string
  requestPayload: any
  requestMeta: Record<string, any>
  responseHeaders: Record<string, any>
  responseTrailers: Record<string, any>
}
