<script setup lang="ts">
  import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
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

  const historyListRef = ref<HTMLElement | null>(null)
  const topSentinelRef = ref<HTMLElement | null>(null)
  const bottomSentinelRef = ref<HTMLElement | null>(null)
  let searchTimer: ReturnType<typeof setTimeout> | null = null
  let listObserver: IntersectionObserver | null = null
  let observerLoadInFlight = false

  function onSearchInput(e: Event) {
    const q = (e.target as HTMLInputElement).value
    historyStore.searchQuery = q
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(async () => {
      await historyStore.loadHistory({ query: q })
      await nextTick()
      if (historyListRef.value) historyListRef.value.scrollTop = 0
    }, 200)
  }

  function captureAnchor() {
    const container = historyListRef.value
    if (!container) return null

    const containerTop = container.getBoundingClientRect().top
    const items = Array.from(container.querySelectorAll<HTMLElement>('[data-history-id]'))

    for (const item of items) {
      const rect = item.getBoundingClientRect()
      if (rect.bottom > containerTop + 1) {
        return {
          id: item.dataset.historyId || '',
          top: rect.top - containerTop,
        }
      }
    }

    return null
  }

  async function restoreAnchor(anchor: { id: string; top: number } | null) {
    if (!anchor || !anchor.id) return

    await nextTick()
    const container = historyListRef.value
    if (!container) return

    const item = container.querySelector<HTMLElement>(`[data-history-id="${anchor.id}"]`)
    if (!item) return

    const containerTop = container.getBoundingClientRect().top
    const nextTop = item.getBoundingClientRect().top - containerTop
    container.scrollTop += nextTop - anchor.top
  }

  async function withAnchor(loader: () => Promise<boolean>) {
    const anchor = captureAnchor()
    const changed = await loader()
    if (!changed) return
    await restoreAnchor(anchor)
  }

  async function maybeLoadOlder() {
    if (observerLoadInFlight || !historyStore.canLoadOlder) return

    observerLoadInFlight = true
    try {
      await withAnchor(() => historyStore.loadOlder())
    } finally {
      observerLoadInFlight = false
    }
  }

  async function maybeLoadNewer() {
    if (observerLoadInFlight || !historyStore.canLoadNewer) return

    observerLoadInFlight = true
    try {
      await withAnchor(() => historyStore.loadNewer())
    } finally {
      observerLoadInFlight = false
    }
  }

  function setupObserver() {
    listObserver?.disconnect()

    const root = historyListRef.value
    if (!root || !topSentinelRef.value || !bottomSentinelRef.value) return

    listObserver = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue
          if (entry.target === bottomSentinelRef.value) {
            maybeLoadOlder()
          } else if (entry.target === topSentinelRef.value) {
            maybeLoadNewer()
          }
        }
      },
      {
        root,
        rootMargin: '120px 0px 120px 0px',
        threshold: 0,
      },
    )

    listObserver.observe(topSentinelRef.value)
    listObserver.observe(bottomSentinelRef.value)
  }

  onMounted(async () => {
    await nextTick()
    setupObserver()
  })

  onBeforeUnmount(() => {
    listObserver?.disconnect()
    listObserver = null
  })

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

    if (historyStore.searchQuery) {
      await historyStore.loadHistory({ query: '' })
      await nextTick()
      if (historyListRef.value) historyListRef.value.scrollTop = 0
    }
    ui.activateTab('tab-server')
    await method.loadServices(item.method)
    if (item.method && method.allMethods.some((m) => m.value === item.method)) {
      method.selectMethod(item.method)
      const sep = item.method.indexOf('::')
      try {
        await method.loadDescribe(item.method.slice(0, sep), item.method.slice(sep + 2))
        ui.activateTab('tab-fields')
      } catch (e: any) {
        ui.showStatus(`Error loading method: ${e.message}`, true)
      }
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
        id="historySearch"
        type="text"
        class="tab-search-input"
        placeholder="Search history..."
        :value="historyStore.searchQuery"
        @input="onSearchInput"
      />
      <div id="historyList" ref="historyListRef" class="records-list">
        <div ref="topSentinelRef" class="history-sentinel" aria-hidden="true" />
        <div v-if="historyStore.isLoading" class="records-empty">Loading history...</div>
        <div v-else-if="!historyStore.history.length" class="records-empty">No history</div>
        <div v-for="item in historyStore.history" :key="item.id" :data-history-id="item.id">
          <RecordsListItem
            :title="item.method || ''"
            title-trim-start
            :meta-left="item.server_name || item.server_url || ''"
            :meta-right="formatHistoryDateTime(item.created_at)"
            :status-class="item.status_code === 0 ? 'ok' : 'err'"
            @click="applyHistoryEntry(item)"
            @delete="historyStore.removeHistoryEntry(item.id)"
          />
        </div>
        <div ref="bottomSentinelRef" class="history-sentinel" aria-hidden="true" />
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
