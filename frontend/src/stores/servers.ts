import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Server } from '@/types/api'
import * as api from '@/api/endpoints'
import { useUiStore } from './ui'

export const useServersStore = defineStore('servers', () => {
  const servers = ref<Server[]>([])
  const searchQuery = ref('')

  async function loadServers(query = '') {
    try {
      servers.value = await api.getServers(query)
    } catch (e: any) {
      useUiStore().showStatus(`Error loading servers: ${e.message}`, true)
    }
  }

  async function addServer(url: string, name: string): Promise<Server | null> {
    try {
      const srv = await api.createServer(url, name)
      await loadServers(searchQuery.value)
      return srv
    } catch (e: any) {
      useUiStore().showStatus(`Error: ${e.message}`, true)
      return null
    }
  }

  async function removeServer(id: number) {
    try {
      await api.deleteServer(id)
      await loadServers(searchQuery.value)
    } catch (e: any) {
      useUiStore().showStatus(`Delete failed: ${e.message}`, true)
    }
  }

  function setServerMeta(id: number, meta: string) {
    const srv = servers.value.find((item) => item.id === id)
    if (srv) srv.meta = meta
  }

  return { servers, searchQuery, loadServers, addServer, removeServer, setServerMeta }
})
