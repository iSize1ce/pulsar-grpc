<script setup lang="ts">
  import { computed, inject } from 'vue'
  import { useConnectionStore } from '@/stores/connection'
  import { useUiStore } from '@/stores/ui'

  const conn = useConnectionStore()
  const ui = useUiStore()

  const saveState = inject<() => void>('saveState', () => {})

  const hasServer = computed(() => !!conn.currentServerId)

  function toggleMetadata(e: Event) {
    e.preventDefault()
    e.stopPropagation()
    if (!hasServer.value) return

    if (ui.isPopoverOpen('serverMetadata')) {
      ui.closePopover('serverMetadata')
    } else {
      ui.openPopover('serverMetadata')
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
  <div class="method-history-popover-anchor">
    <button
      type="button"
      class="input-row-action-btn"
      title="Metadata"
      :disabled="!hasServer"
      :class="{ open: ui.isPopoverOpen('serverMetadata') }"
      @click="toggleMetadata"
    >
      M
    </button>

    <div
      v-show="ui.isPopoverOpen('serverMetadata')"
      class="method-history-popover server-metadata-popover"
      @click.stop
    >
      <div class="metadata-popover-header">
        <div class="method-history-title">Metadata</div>
        <button type="button" class="btn-add" @click="addMetaRow">+</button>
      </div>
      <div id="metaList">
        <div v-if="!conn.metadata.length" class="records-empty">No metadata</div>
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
