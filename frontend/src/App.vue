<script setup lang="ts">
  import { onMounted, onUnmounted } from 'vue'
  import { useUiStore } from '@/stores/ui'
  import RequestPanel from '@/components/layout/RequestPanel.vue'
  import ResponsePanel from '@/components/layout/ResponsePanel.vue'

  const ui = useUiStore()

  onMounted(() => {
    ui.applyTheme()
  })

  function onDocumentClick() {
    ui.closeAllPopovers()
  }

  function onDocumentKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      ui.closeAllPopovers()
    }
  }

  onMounted(() => {
    document.addEventListener('click', onDocumentClick)
    document.addEventListener('keydown', onDocumentKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('click', onDocumentClick)
    document.removeEventListener('keydown', onDocumentKeydown)
  })
</script>

<template>
  <div class="app">
    <RequestPanel />
    <ResponsePanel />
  </div>
</template>
