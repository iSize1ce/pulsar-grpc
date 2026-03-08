<script setup lang="ts">
  import { onMounted, ref, provide } from 'vue'
  import { useUiStore, type TabId } from '@/stores/ui'
  import { useServersStore } from '@/stores/servers'
  import { useSavedRequestsStore } from '@/stores/savedRequests'
  import { useHistoryStore } from '@/stores/history'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { usePayloadStore } from '@/stores/payload'
  import { useResponseStore } from '@/stores/response'
  import { invokeMethod } from '@/api/endpoints'
  import ServerTab from '@/components/tabs/ServerTab.vue'
  import SavedRequestsTab from '@/components/tabs/SavedRequestsTab.vue'
  import HistoryTab from '@/components/tabs/HistoryTab.vue'
  import PayloadTab from '@/components/tabs/PayloadTab.vue'

  const ui = useUiStore()
  const servers = useServersStore()
  const savedReqs = useSavedRequestsStore()
  const historyStore = useHistoryStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const payload = usePayloadStore()
  const response = useResponseStore()
  const isDesktopShell = navigator.userAgent.includes('Electron')
  const isMac = navigator.userAgent.includes('Mac')

  const payloadTabRef = ref<InstanceType<typeof PayloadTab> | null>(null)

  const tabs: { id: TabId; label: string }[] = [
    { id: 'tab-server', label: 'Server' },
    { id: 'tab-saved', label: 'Saved Requests' },
    { id: 'tab-history', label: 'History' },
    { id: 'tab-fields', label: 'Payload' },
  ]

  function getPayload(): Record<string, any> {
    return payloadTabRef.value?.getPayload?.() ?? {}
  }

  function saveState() {
    if (payload.restoring) return
    const state: any = {}
    if (conn.currentServerId) state.serverId = conn.currentServerId
    if (method.currentService && method.currentMethod) {
      state.method = `${method.currentService}::${method.currentMethod}`
    }
    if (method.currentFields.length) {
      const p = getPayload()
      if (Object.keys(p).length) state.payload = p
    }
    if (Object.keys(state).length) {
      localStorage.setItem('appState', JSON.stringify(state))
    } else {
      localStorage.removeItem('appState')
    }
  }

  provide('saveState', saveState)
  provide('getPayload', getPayload)

  onMounted(async () => {
    restoreState()

    await servers.loadServers()
    await savedReqs.loadSavedRequests()
    await historyStore.loadHistory()

    if (payload.pendingRestore) {
      const state = payload.pendingRestore
      payload.pendingRestore = null
      const srv = servers.servers.find((s) => s.id === state.serverId)
      if (srv) {
        conn.selectServerData(srv)
        await method.loadServices(state.method || '')
        if (state.method && method.allMethods.some((m) => m.value === state.method)) {
          method.selectMethod(state.method)
          const sep = state.method.indexOf('::')
          await method.loadDescribe(state.method.slice(0, sep), state.method.slice(sep + 2))
          ui.activateTab('tab-fields')
        }
      }
      payload.restoring = false
    }
  })

  function restoreState() {
    const raw = localStorage.getItem('appState')
    if (!raw) return
    let state: any
    try {
      state = JSON.parse(raw)
    } catch {
      return
    }

    payload.restoring = true
    if (state.payload && typeof state.payload === 'object' && Object.keys(state.payload).length) {
      payload.savedPayload = state.payload
    }
    if (!state.serverId) {
      payload.restoring = false
      return
    }
    conn.currentServerId = state.serverId
    payload.pendingRestore = state
  }

  function onClear() {
    if (!method.currentFields.length) return
    if (!confirm('Clear request?')) return
    payload.savedPayload = null
    payloadTabRef.value?.clearForm()
    response.clearResponse()
    saveState()
  }

  async function onSend() {
    if (!conn.grpcUrl || !method.currentService || !method.currentMethod) return

    response.clearResponse()
    response.responseJson = '// Sending...'

    try {
      const currentPayload = getPayload()
      const meta = conn.collectMeta()
      const resp = await invokeMethod(
        conn.grpcUrl,
        method.currentService,
        method.currentMethod,
        currentPayload,
        meta,
      )

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

        historyStore.saveToHistory(
          conn.currentServerId,
          `${method.currentService}::${method.currentMethod}`,
          JSON.stringify(currentPayload),
          JSON.stringify(resp),
          resp.grpcStatusCode || 2,
        )
      } else {
        response.lastResponseData = resp.data
        response.setResponse(JSON.stringify(resp.data, null, 2), false)

        historyStore.saveToHistory(
          conn.currentServerId,
          `${method.currentService}::${method.currentMethod}`,
          JSON.stringify(currentPayload),
          JSON.stringify(resp),
          0,
        )
      }
    } catch (e: any) {
      response.setResponse(`// Error: ${e.message}`, true)
    }
  }

  function onSaveRequest() {
    if (!conn.currentServerId || !method.currentService || !method.currentMethod) return
    const name = prompt('Request name:')
    if (name === null) return
    const p = JSON.stringify(getPayload())
    const m = `${method.currentService}::${method.currentMethod}`
    savedReqs.saveRequest(name, conn.currentServerId, m, p)
  }
</script>

<template>
  <div class="panel panel-request">
    <div
      class="panel-header"
      :class="{
        'panel-header-desktop': isDesktopShell,
        'panel-header-desktop-mac': isDesktopShell && isMac,
      }"
    >
      <div v-if="isDesktopShell" class="panel-desktop-title">Pulsar GRPC</div>

      <div class="tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab"
          :class="{ active: ui.activeTab === tab.id }"
          @click="ui.activateTab(tab.id)"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <ServerTab v-show="ui.activeTab === 'tab-server'" />
    <SavedRequestsTab v-show="ui.activeTab === 'tab-saved'" />
    <HistoryTab v-show="ui.activeTab === 'tab-history'" />
    <PayloadTab v-show="ui.activeTab === 'tab-fields'" ref="payloadTabRef" />

    <div class="panel-footer">
      <button id="btnClear" @click="onClear">Clear</button>
      <button
        v-show="method.currentService && method.currentMethod"
        id="btnSaveRequest"
        @click="onSaveRequest"
      >
        Save
      </button>
      <button id="btnSend" @click="onSend">Send</button>
    </div>
  </div>
</template>
