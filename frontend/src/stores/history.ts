import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { HistoryEntry } from '@/types/api'
import * as api from '@/api/endpoints'
import { useUiStore } from './ui'

const HISTORY_PAGE_SIZE = 40
const HISTORY_MAX_ITEMS = 120

export const useHistoryStore = defineStore('history', () => {
  const history = ref<HistoryEntry[]>([])
  const searchQuery = ref('')
  const total = ref(0)
  const isLoading = ref(false)
  const isLoadingOlder = ref(false)
  const isLoadingNewer = ref(false)
  const startOffset = ref(0)

  const canLoadOlder = computed(() => startOffset.value + history.value.length < total.value)
  const canLoadNewer = computed(() => startOffset.value > 0)

  async function fetchHistory(offset: number, limit: number) {
    const response = await api.getHistory({
      q: searchQuery.value || undefined,
      limit,
      offset,
    })
    total.value = response.total
    return response
  }

  async function loadHistory(options: { query?: string } = {}) {
    if (options.query !== undefined) searchQuery.value = options.query

    isLoading.value = true
    startOffset.value = 0
    history.value = []

    try {
      const response = await fetchHistory(0, HISTORY_PAGE_SIZE)
      startOffset.value = 0
      history.value = response.items
    } catch (e: any) {
      useUiStore().showStatus(`Error loading history: ${e.message}`, true)
      history.value = []
      total.value = 0
    } finally {
      isLoading.value = false
    }
  }

  async function loadOlder() {
    if (isLoading.value || isLoadingOlder.value || !canLoadOlder.value) return false

    isLoadingOlder.value = true
    try {
      const response = await fetchHistory(startOffset.value + history.value.length, HISTORY_PAGE_SIZE)
      if (!response.items.length) return false

      history.value = history.value.concat(response.items)

      if (history.value.length > HISTORY_MAX_ITEMS) {
        const dropped = history.value.length - HISTORY_MAX_ITEMS
        history.value = history.value.slice(dropped)
        startOffset.value += dropped
      }

      return true
    } catch (e: any) {
      useUiStore().showStatus(`Error loading history: ${e.message}`, true)
      return false
    } finally {
      isLoadingOlder.value = false
    }
  }

  async function loadNewer() {
    if (isLoading.value || isLoadingNewer.value || !canLoadNewer.value) return false

    isLoadingNewer.value = true
    try {
      const nextOffset = Math.max(0, startOffset.value - HISTORY_PAGE_SIZE)
      const response = await fetchHistory(nextOffset, HISTORY_PAGE_SIZE)
      if (!response.items.length) return false

      history.value = response.items.concat(history.value)
      startOffset.value = nextOffset

      if (history.value.length > HISTORY_MAX_ITEMS) {
        history.value = history.value.slice(0, HISTORY_MAX_ITEMS)
      }

      return true
    } catch (e: any) {
      useUiStore().showStatus(`Error loading history: ${e.message}`, true)
      return false
    } finally {
      isLoadingNewer.value = false
    }
  }

  async function reloadVisibleWindow() {
    const limit = Math.min(Math.max(history.value.length, HISTORY_PAGE_SIZE), HISTORY_MAX_ITEMS)
    let offset = startOffset.value

    try {
      let response = await fetchHistory(offset, limit)

      if (response.total > 0 && offset >= response.total) {
        offset = Math.max(0, response.total - HISTORY_PAGE_SIZE)
        response = await fetchHistory(offset, limit)
      }

      startOffset.value = offset
      history.value = response.items
    } catch (e: any) {
      useUiStore().showStatus(`Error loading history: ${e.message}`, true)
    }
  }

  async function removeHistoryEntry(id: number) {
    try {
      await api.deleteHistory(id)
      await reloadVisibleWindow()
    } catch (e: any) {
      useUiStore().showStatus(`Delete failed: ${e.message}`, true)
    }
  }

  async function clearAllHistory() {
    try {
      await api.clearHistory()
      history.value = []
      total.value = 0
      startOffset.value = 0
    } catch (e: any) {
      useUiStore().showStatus(`Clear failed: ${e.message}`, true)
    }
  }

  async function saveToHistory(
    serverId: number,
    method: string,
    payload: string,
    response: string,
    statusCode: number,
  ) {
    try {
      await api.createHistory(serverId, method, payload, response, statusCode)
      if (startOffset.value === 0) {
        await loadHistory()
      } else {
        total.value += 1
      }
    } catch {
      /* ignore */
    }
  }

  return {
    history,
    searchQuery,
    total,
    startOffset,
    isLoading,
    isLoadingOlder,
    isLoadingNewer,
    canLoadOlder,
    canLoadNewer,
    loadHistory,
    loadOlder,
    loadNewer,
    removeHistoryEntry,
    clearAllHistory,
    saveToHistory,
  }
})
