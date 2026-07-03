import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Server, MetaEntry } from '@/types/api'
import * as api from '@/api/endpoints'
import { useServersStore } from './servers'
import { useUiStore } from './ui'

sanitizePersistedConnectionMeta()

export const useConnectionStore = defineStore(
  'connection',
  () => {
    const grpcUrl = ref('')
    const currentServerId = ref(0)
    const metadata = ref<MetaEntry[]>([])

    let metaSaveTimer: ReturnType<typeof setTimeout> | null = null
    const lastSavedMeta: Record<number, string> = {}

    function selectServerData(srv: Server) {
      grpcUrl.value = srv.url
      currentServerId.value = srv.id
      metadata.value = parseMeta(srv.meta)
      lastSavedMeta[srv.id] = JSON.stringify(metadata.value)
    }

    function addMetaRow(key = '', value = '') {
      metadata.value.push({ key, value })
    }

    function removeMetaRow(index: number) {
      metadata.value.splice(index, 1)
      saveCurrentServerMeta()
    }

    function collectMeta(): MetaEntry[] {
      return metadata.value.filter((m) => m.key.trim())
    }

    function saveCurrentServerMeta(immediate = false) {
      if (!currentServerId.value) return
      if (metaSaveTimer) clearTimeout(metaSaveTimer)

      const serverId = currentServerId.value
      const doSave = async () => {
        const meta = JSON.stringify(collectMeta())
        if (lastSavedMeta[serverId] === meta) return
        try {
          await api.updateServerMeta(serverId, meta)
          lastSavedMeta[serverId] = meta
          useServersStore().setServerMeta(serverId, meta)
        } catch (e: any) {
          useUiStore().showStatus(`Metadata save failed: ${e.message}`, true)
        }
      }

      if (immediate) {
        void doSave()
        return
      }
      metaSaveTimer = setTimeout(() => void doSave(), 500)
    }

    function parseMeta(metaStr: string): MetaEntry[] {
      try {
        return JSON.parse(metaStr || '[]')
      } catch {
        return []
      }
    }

    return {
      grpcUrl,
      currentServerId,
      metadata,
      selectServerData,
      addMetaRow,
      removeMetaRow,
      collectMeta,
      saveCurrentServerMeta,
      parseMeta,
    }
  },
  {
    persist: {
      pick: ['grpcUrl', 'currentServerId'],
    },
  },
)

function sanitizePersistedConnectionMeta() {
  if (typeof localStorage === 'undefined') return

  const raw = localStorage.getItem('connection')
  if (!raw) return

  try {
    const persisted = JSON.parse(raw)
    if (!persisted || typeof persisted !== 'object' || !('metadata' in persisted)) return
    delete persisted.metadata
    localStorage.setItem('connection', JSON.stringify(persisted))
  } catch {
    /* Ignore malformed persisted state; Pinia will fall back to defaults. */
  }
}
