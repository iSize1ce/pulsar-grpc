<script setup lang="ts">
  import { ref, inject, nextTick, watch } from 'vue'
  import { useServersStore } from '@/stores/servers'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { useHistoryStore } from '@/stores/history'
  import { useUiStore } from '@/stores/ui'
  import type { Server } from '@/types/api'
  import RecordsListItem from '@/components/shared/RecordsListItem.vue'

  const servers = useServersStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const historyStore = useHistoryStore()
  const ui = useUiStore()

  const saveState = inject<() => void>('saveState', () => {})
  const serverListRef = ref<HTMLElement | null>(null)
  let needScrollToSelected = false

  watch(() => ui.activeTab, async (tab) => {
    if (tab === 'tab-server' && needScrollToSelected) {
      needScrollToSelected = false
      await scrollToSelectedServer()
    }
  })

  let searchTimer: ReturnType<typeof setTimeout> | null = null

  // Add server popover
  const newServerUrl = ref('')
  const newServerName = ref('')

  function onSearchInput(e: Event) {
    const q = (e.target as HTMLInputElement).value
    servers.searchQuery = q
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => servers.loadServers(q), 200)
  }

  async function scrollToSelectedServer() {
    await nextTick()
    const container = serverListRef.value
    if (!container) return
    const activeItem = container.querySelector<HTMLElement>('.records-item.active')
    if (!activeItem) return
    const containerRect = container.getBoundingClientRect()
    const itemRect = activeItem.getBoundingClientRect()
    if (itemRect.top < containerRect.top) {
      container.scrollTop += itemRect.top - containerRect.top
    } else if (itemRect.bottom > containerRect.bottom) {
      container.scrollTop += itemRect.bottom - containerRect.bottom
    }
  }

  async function selectServer(srv: Server) {
    if (srv.id === conn.currentServerId) {
      await method.loadServices('', true)
      return
    }

    conn.saveCurrentServerMeta(true)
    conn.selectServerData(srv)

    const loaded = await method.loadServices()
    if (!loaded) return

    if (servers.searchQuery) {
      servers.searchQuery = ''
      await servers.loadServers()
      needScrollToSelected = true
    }

    saveState()
  }

  async function deleteServerItem(id: number) {
    await servers.removeServer(id)
    await historyStore.loadHistory()
  }

  function toggleAddServer(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    if (ui.isPopoverOpen('addServer')) {
      ui.closePopover('addServer')
    } else {
      ui.openPopover('addServer')
      newServerUrl.value = ''
      newServerName.value = ''
    }
  }

  async function createServer() {
    const url = newServerUrl.value.trim()
    if (!url) return
    const name = newServerName.value.trim()
    const srv = await servers.addServer(url, name)
    if (srv) {
      ui.closePopover('addServer')
      selectServer(srv)
    }
  }

  function onNewServerKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      createServer()
    }
  }

  function addMetaRow() {
    conn.addMetaRow()
    saveState()
  }

  function onMetaInput() {
    saveState()
    conn.saveCurrentServerMeta()
  }

  function removeMetaRow(index: number) {
    conn.removeMetaRow(index)
    saveState()
  }
</script>

<template>
  <div id="tab-server" class="tab-pane">
    <div class="section">
      <div class="section-header-server">
        <input
          id="serverSearch"
          type="text"
          placeholder="Search server..."
          :value="servers.searchQuery"
          @input="onSearchInput"
        />
        <button
          type="button"
          class="btn-add input-row-action-btn"
          :class="{ open: ui.isPopoverOpen('addServer') }"
          @click="toggleAddServer"
        >
          +
        </button>
        <div
          v-show="ui.isPopoverOpen('addServer')"
          class="method-history-popover add-server-popover"
          @click.stop
        >
          <div class="method-history-title">Add Server</div>
          <input
            v-model="newServerUrl"
            type="text"
            placeholder="localhost:50051"
            @keydown="onNewServerKeydown"
          />
          <input
            v-model="newServerName"
            type="text"
            placeholder="Name (optional)"
            style="margin-top: 8px"
            @keydown="onNewServerKeydown"
          />
          <div class="row" style="margin-top: 8px">
            <button type="button" class="btn-create" @click="createServer">Create</button>
            <button type="button" class="btn-cancel" @click="ui.closePopover('addServer')">
              Cancel
            </button>
          </div>
        </div>
      </div>

      <div id="serverList" ref="serverListRef" class="records-list">
        <div v-if="!servers.servers.length" class="records-empty">No servers</div>
        <RecordsListItem
          v-for="srv in servers.servers"
          :key="srv.id"
          :title="srv.name || srv.url || ''"
          :subtitle="srv.name ? srv.url || '' : ''"
          :active="srv.id === conn.currentServerId"
          @click="selectServer(srv)"
          @delete="deleteServerItem(srv.id)"
        />
      </div>

      <div
        v-show="ui.statusMessage"
        id="connectStatus"
        class="status"
        :class="{ error: ui.statusIsError, success: !ui.statusIsError }"
      >
        {{ ui.statusMessage }}
      </div>
    </div>

    <div class="section">
      <div class="section-header">
        <label>Metadata</label>
        <button type="button" class="btn-add" @click="addMetaRow">+</button>
      </div>
      <div id="metaList">
        <div v-for="(meta, index) in conn.metadata" :key="index" class="meta-item">
          <input
            v-model="meta.key"
            type="text"
            class="meta-key"
            placeholder="key"
            @input="onMetaInput"
          />
          <input
            v-model="meta.value"
            type="text"
            class="meta-value"
            placeholder="value"
            @input="onMetaInput"
          />
          <button type="button" class="btn-remove" @click="removeMetaRow(index)">&minus;</button>
        </div>
      </div>
    </div>
  </div>
</template>
