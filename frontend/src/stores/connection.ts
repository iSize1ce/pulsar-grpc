import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Server, MetaEntry } from '@/types/api'
import * as api from '@/api/endpoints'

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

      const doSave = () => {
        const meta = JSON.stringify(collectMeta())
        if (lastSavedMeta[currentServerId.value] === meta) return
        lastSavedMeta[currentServerId.value] = meta
        api.updateServerMeta(currentServerId.value, meta)
      }

      if (immediate) {
        doSave()
        return
      }
      metaSaveTimer = setTimeout(doSave, 500)
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
      pick: ['grpcUrl', 'currentServerId', 'metadata'],
    },
  },
)
