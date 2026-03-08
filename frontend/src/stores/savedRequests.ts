import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { SavedRequest } from '@/types/api'
import * as api from '@/api/endpoints'
import { useUiStore } from './ui'

export const useSavedRequestsStore = defineStore('savedRequests', () => {
  const savedRequests = ref<SavedRequest[]>([])
  const searchQuery = ref('')

  async function loadSavedRequests(query = '') {
    try {
      savedRequests.value = await api.getSavedRequests(query)
    } catch (e: any) {
      useUiStore().showStatus(`Error loading saved requests: ${e.message}`, true)
    }
  }

  async function removeSavedRequest(id: number) {
    try {
      await api.deleteSavedRequest(id)
      await loadSavedRequests(searchQuery.value)
    } catch (e: any) {
      useUiStore().showStatus(`Delete failed: ${e.message}`, true)
    }
  }

  async function saveRequest(name: string, serverId: number, method: string, payload: string) {
    try {
      const items = await api.getSavedRequests(name)
      const existing = items.find((i) => i.name === name)
      if (existing) {
        if (!confirm(`Request "${name}" already exists. Overwrite?`)) return
        await api.updateSavedRequest(existing.id, serverId, method, payload)
      } else {
        await api.createSavedRequest(name, serverId, method, payload)
      }
      await loadSavedRequests(searchQuery.value)
    } catch (e: any) {
      useUiStore().showStatus(`Save failed: ${e.message}`, true)
    }
  }

  return { savedRequests, searchQuery, loadSavedRequests, removeSavedRequest, saveRequest }
})
