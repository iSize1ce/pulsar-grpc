<script setup lang="ts">
  import { useHistoryStore } from '@/stores/history'
  import { useServersStore } from '@/stores/servers'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { usePayloadStore } from '@/stores/payload'
  import { useResponseStore } from '@/stores/response'
  import { useUiStore } from '@/stores/ui'
  import type { HistoryEntry } from '@/types/api'
  import { formatHistoryDateTime } from '@/utils/dateFormat'
  import RecordsListItem from '@/components/shared/RecordsListItem.vue'

  const historyStore = useHistoryStore()
  const servers = useServersStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const payload = usePayloadStore()
  const response = useResponseStore()
  const ui = useUiStore()

  let searchTimer: ReturnType<typeof setTimeout> | null = null

  function onSearchInput(e: Event) {
    const q = (e.target as HTMLInputElement).value
    historyStore.searchQuery = q
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => historyStore.loadHistory(q), 200)
  }

  async function applyHistoryEntry(item: HistoryEntry) {
    const srv = servers.servers.find((s) => s.id === item.server_id)
    if (!srv) {
      ui.showStatus('Server was deleted', true)
      return
    }

    conn.selectServerData(srv)

    try {
      payload.savedPayload = JSON.parse(item.payload || 'null')
    } catch {
      payload.savedPayload = null
    }

    // Restore response
    try {
      const resp = JSON.parse(item.response || 'null')
      if (resp) {
        response.debugInfo = resp.debug || null
        if (resp.grpcStatus) {
          const errObj: any = {
            status: resp.grpcStatusCode,
            statusName: resp.grpcStatus,
            message: resp.grpcMessage || '',
          }
          if (resp.errorDetails?.length) errObj.errorDetails = resp.errorDetails
          response.lastResponseData = errObj
          response.setResponse(JSON.stringify(errObj, null, 2), true)
        } else if (resp.data) {
          response.lastResponseData = resp.data
          response.setResponse(JSON.stringify(resp.data, null, 2), false)
        }
      }
    } catch {
      /* ignore */
    }

    historyStore.searchQuery = ''
    historyStore.loadHistory()
    ui.activateTab('tab-server')
    await method.loadServices(item.method)
    if (item.method && method.allMethods.some((m) => m.value === item.method)) {
      method.selectMethod(item.method)
      const sep = item.method.indexOf('::')
      await method.loadDescribe(item.method.slice(0, sep), item.method.slice(sep + 2))
      ui.activateTab('tab-fields')
    }
  }

  function clearHistory() {
    if (!confirm('Clear all history?')) return
    historyStore.clearAllHistory()
  }
</script>

<template>
  <div id="tab-history" class="tab-pane">
    <div class="section">
      <input
        type="text"
        class="tab-search-input"
        placeholder="Search history..."
        :value="historyStore.searchQuery"
        @input="onSearchInput"
      />
      <div class="records-list">
        <div v-if="!historyStore.history.length" class="records-empty">No history</div>
        <RecordsListItem
          v-for="item in historyStore.history"
          :key="item.id"
          :title="item.method || ''"
          :meta-left="item.server_name || item.server_url || ''"
          :meta-right="formatHistoryDateTime(item.created_at)"
          :status-class="item.status_code === 0 ? 'ok' : 'err'"
          @click="applyHistoryEntry(item)"
          @delete="historyStore.removeHistoryEntry(item.id)"
        />
      </div>
      <div class="section-actions-bottom">
        <button
          type="button"
          class="btn-clear-history"
          title="Clear all history"
          @click="clearHistory"
        >
          Clear history
        </button>
      </div>
    </div>
  </div>
</template>
