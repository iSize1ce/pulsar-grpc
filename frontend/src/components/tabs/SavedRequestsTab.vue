<script setup lang="ts">
  import { useSavedRequestsStore } from '@/stores/savedRequests'
  import { useServersStore } from '@/stores/servers'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { usePayloadStore } from '@/stores/payload'
  import { useUiStore } from '@/stores/ui'
  import type { SavedRequest } from '@/types/api'
  import RecordsListItem from '@/components/shared/RecordsListItem.vue'

  const savedReqs = useSavedRequestsStore()
  const servers = useServersStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const payload = usePayloadStore()
  const ui = useUiStore()

  let searchTimer: ReturnType<typeof setTimeout> | null = null

  function onSearchInput(e: Event) {
    const q = (e.target as HTMLInputElement).value
    savedReqs.searchQuery = q
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => savedReqs.loadSavedRequests(q), 200)
  }

  async function applySavedRequest(item: SavedRequest) {
    const srv = servers.servers.find((s) => s.id === item.server_id)
    if (!srv) {
      ui.showStatus('Server not found', true)
      return
    }

    conn.selectServerData(srv)

    try {
      payload.savedPayload = JSON.parse(item.payload || 'null')
    } catch {
      payload.savedPayload = null
    }

    if (savedReqs.searchQuery) {
      savedReqs.searchQuery = ''
      savedReqs.loadSavedRequests()
    }

    ui.activateTab('tab-fields')
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

  async function deleteSavedRequestItem(id: number) {
    if (!confirm('Delete saved request?')) return
    await savedReqs.removeSavedRequest(id)
  }
</script>

<template>
  <div id="tab-saved" class="tab-pane">
    <div class="section">
      <input
        type="text"
        class="tab-search-input"
        placeholder="Search request..."
        :value="savedReqs.searchQuery"
        @input="onSearchInput"
      />
      <div class="records-list">
        <div v-if="!savedReqs.savedRequests.length" class="records-empty">No saved requests</div>
        <RecordsListItem
          v-for="item in savedReqs.savedRequests"
          :key="item.id"
          :title="item.name || item.method || 'Unnamed request'"
          :subtitle="item.name ? item.method || '' : ''"
          @click="applySavedRequest(item)"
          @delete="deleteSavedRequestItem(item.id)"
        />
      </div>
    </div>
  </div>
</template>
