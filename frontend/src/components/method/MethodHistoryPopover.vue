<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { useMethodStore } from '@/stores/method'
  import { useUiStore } from '@/stores/ui'
  import { usePayloadStore } from '@/stores/payload'
  import {
    getSavedRequests,
    getSavedRequest,
    getHistory,
    getHistoryEntry,
  } from '@/api/endpoints'
  import type { SavedRequest, HistoryEntry } from '@/types/api'
  import { formatHistoryDateTime } from '@/utils/dateFormat'
  import RecordsListItem from '@/components/shared/RecordsListItem.vue'

  const methodStore = useMethodStore()
  const ui = useUiStore()
  const payload = usePayloadStore()

  const savedItems = ref<SavedRequest[]>([])
  const historyItems = ref<HistoryEntry[]>([])
  const loading = ref(false)
  let requestSeq = 0

  const hasMethod = computed(() => !!methodStore.currentMethodValue)

  async function togglePopover(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    if (!hasMethod.value) return
    if (ui.isPopoverOpen('methodHistory')) {
      ui.closePopover('methodHistory')
      return
    }
    await openPopover()
  }

  async function openPopover() {
    const method = methodStore.currentMethodValue
    if (!method) return

    const seq = ++requestSeq
    loading.value = true
    ui.openPopover('methodHistory')

    try {
      const [saved, history] = await Promise.all([
        getSavedRequests('', method),
        getHistory({ method, limit: 20 }),
      ])
      if (seq !== requestSeq) return
      savedItems.value = saved
      historyItems.value = history.items
    } catch (e: any) {
      if (seq !== requestSeq) return
      ui.showStatus(`Error loading method history: ${e.message}`, true)
      savedItems.value = []
      historyItems.value = []
    } finally {
      loading.value = false
    }
  }

  async function applyPayload(payloadJson: string) {
    if (!methodStore.currentService || !methodStore.currentMethod) return
    try {
      payload.savedPayload = JSON.parse(payloadJson || 'null')
    } catch {
      ui.showStatus('Selected item has invalid payload JSON', true)
      return
    }
    ui.closePopover('methodHistory')
    try {
      await methodStore.loadDescribe(methodStore.currentService, methodStore.currentMethod)
    } catch (e: any) {
      ui.showStatus(`Error loading method: ${e.message}`, true)
    }
  }

  async function applySavedItem(item: SavedRequest) {
    try {
      const detail = await getSavedRequest(item.id)
      await applyPayload(detail.payload)
    } catch (e: any) {
      ui.showStatus(`Error loading saved request: ${e.message}`, true)
    }
  }

  async function applyHistoryItem(item: HistoryEntry) {
    try {
      const detail = await getHistoryEntry(item.id)
      await applyPayload(detail.payload)
    } catch (e: any) {
      ui.showStatus(`Error loading history entry: ${e.message}`, true)
    }
  }
</script>

<template>
  <div class="method-history-popover-anchor">
    <button
      type="button"
      class="btn-method-history input-row-action-btn"
      title="Saved payloads and history"
      :disabled="!hasMethod"
      :class="{ open: ui.isPopoverOpen('methodHistory') }"
      @click="togglePopover"
    >
      &#x1f558;
    </button>

    <div v-show="ui.isPopoverOpen('methodHistory')" class="method-history-popover" @click.stop>
      <div class="method-history-section">
        <div class="method-history-title">Saved</div>
        <div class="records-list">
          <div v-if="loading" class="records-empty">Loading...</div>
          <div v-else-if="!savedItems.length" class="records-empty">
            No saved requests for this method
          </div>
          <RecordsListItem
            v-for="item in savedItems"
            :key="item.id"
            :title="item.name || item.method || 'Unnamed request'"
            :subtitle="
              item.name ? item.method || `Server #${item.server_id}` : `Server #${item.server_id}`
            "
            @click="applySavedItem(item)"
          />
        </div>
      </div>

      <div class="method-history-section">
        <div class="method-history-title">History</div>
        <div class="records-list">
          <div v-if="loading" class="records-empty">Loading...</div>
          <div v-else-if="!historyItems.length" class="records-empty">
            No history for this method
          </div>
          <RecordsListItem
            v-for="item in historyItems"
            :key="item.id"
            :title="item.method || ''"
            title-trim-start
            :meta-left="item.server_name || item.server_url || `Server #${item.server_id}`"
            :meta-right="formatHistoryDateTime(item.created_at)"
            :status-class="item.status_code === 0 ? 'ok' : 'err'"
            @click="applyHistoryItem(item)"
          />
        </div>
      </div>
    </div>
  </div>
</template>
