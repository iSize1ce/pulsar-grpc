<script setup lang="ts">
  import { ref, inject } from 'vue'
  import { useServersStore } from '@/stores/servers'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { useUiStore } from '@/stores/ui'

  const servers = useServersStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const ui = useUiStore()

  const saveState = inject<() => void>('saveState', () => {})

  const newServerUrl = ref('')
  const newServerName = ref('')

  function toggleAddServer(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    if (ui.isPopoverOpen('addServer')) {
      ui.closePopover('addServer')
      return
    }

    ui.openPopover('addServer')
    newServerUrl.value = ''
    newServerName.value = ''
  }

  async function createServer() {
    const url = newServerUrl.value.trim()
    if (!url) return

    const name = newServerName.value.trim()
    const srv = await servers.addServer(url, name)
    if (!srv) return

    ui.closePopover('addServer')
    conn.saveCurrentServerMeta(true)
    conn.selectServerData(srv)

    const loaded = await method.loadServices()
    if (loaded) {
      saveState()
    }
  }

  function onNewServerKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      createServer()
    }
  }
</script>

<template>
  <div class="method-history-popover-anchor">
    <button
      type="button"
      class="input-row-action-btn"
      title="Add server"
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
        @keydown="onNewServerKeydown"
      />
      <div class="row">
        <button type="button" class="btn-create" @click="createServer">Create</button>
        <button type="button" class="btn-cancel" @click="ui.closePopover('addServer')">
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>
