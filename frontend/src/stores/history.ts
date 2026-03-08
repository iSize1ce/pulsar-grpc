import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { HistoryEntry } from '@/types/api'
import * as api from '@/api/endpoints'
import { useUiStore } from './ui'

export const useHistoryStore = defineStore('history', () => {
  const history = ref<HistoryEntry[]>([])
  const searchQuery = ref('')

  async function loadHistory(query = '') {
    try {
      history.value = await api.getHistory(query)
    } catch (e: any) {
      useUiStore().showStatus(`Error loading history: ${e.message}`, true)
    }
  }

  async function removeHistoryEntry(id: number) {
    try {
      await api.deleteHistory(id)
      await loadHistory(searchQuery.value)
    } catch (e: any) {
      useUiStore().showStatus(`Delete failed: ${e.message}`, true)
    }
  }

  async function clearAllHistory() {
    try {
      await api.clearHistory()
      await loadHistory()
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
      await loadHistory(searchQuery.value)
    } catch {
      /* ignore */
    }
  }

  return { history, searchQuery, loadHistory, removeHistoryEntry, clearAllHistory, saveToHistory }
})
