<script setup lang="ts">
  import { ref, computed, watch, inject } from 'vue'
  import { useServersStore } from '@/stores/servers'
  import { useConnectionStore } from '@/stores/connection'
  import { useMethodStore } from '@/stores/method'
  import { useUiStore } from '@/stores/ui'
  import { convertLayout } from '@/utils/layoutConvert'
  import type { Server } from '@/types/api'
  import StartTrimText from '@/components/shared/StartTrimText.vue'

  const servers = useServersStore()
  const conn = useConnectionStore()
  const method = useMethodStore()
  const ui = useUiStore()

  const saveState = inject<() => void>('saveState', () => {})

  const inputValue = ref('')
  const dropdownOpen = ref(false)
  const acActiveIdx = ref(-1)
  const inputRef = ref<HTMLInputElement | null>(null)
  const isFocused = ref(false)
  let serverInputRestoreValue = ''

  const currentServer = computed(() =>
    servers.servers.find((srv) => srv.id === conn.currentServerId) || null,
  )

  const currentServerValue = computed(() => formatServer(currentServer.value))

  watch(
    () => currentServerValue.value,
    (val) => {
      if (!dropdownOpen.value) {
        inputValue.value = val
      }
    },
    { immediate: true },
  )

  const filteredServers = computed(() => {
    const q = inputValue.value.toLowerCase()
    const qAlt = q ? convertLayout(q) : ''

    return servers.servers.filter((srv) => {
      const value = `${srv.name || ''} ${srv.url || ''}`.toLowerCase()
      return !q || value.includes(q) || (!!qAlt && value.includes(qAlt))
    })
  })

  function formatServer(srv: Server | null) {
    if (!srv) return ''
    return srv.name ? `${srv.name} (${srv.url})` : srv.url
  }

  function onFocus() {
    isFocused.value = true
    const selected = currentServerValue.value
    serverInputRestoreValue = selected
    if (inputValue.value === selected) {
      inputValue.value = ''
    }
    ui.closePopover('serverMetadata')
    ui.closePopover('addServer')
    dropdownOpen.value = true
    acActiveIdx.value = -1
  }

  function onInput(e: Event) {
    inputValue.value = (e.target as HTMLInputElement).value
    ui.closePopover('serverMetadata')
    ui.closePopover('addServer')
    dropdownOpen.value = true
    acActiveIdx.value = -1
  }

  function onBlur() {
    setTimeout(() => {
      isFocused.value = false
      dropdownOpen.value = false
      acActiveIdx.value = -1
      const selected = currentServerValue.value
      inputValue.value = selected || serverInputRestoreValue
      serverInputRestoreValue = ''
    }, 150)
  }

  function focusInput() {
    inputRef.value?.focus()
  }

  const selectedDisplayValue = computed(() =>
    !isFocused.value && !dropdownOpen.value ? currentServerValue.value : '',
  )

  async function selectServer(srv: Server) {
    const currentId = conn.currentServerId
    inputValue.value = formatServer(srv)
    dropdownOpen.value = false
    acActiveIdx.value = -1
    inputRef.value?.blur()

    if (srv.id === currentId) {
      await method.loadServices('', true)
      return
    }

    conn.saveCurrentServerMeta(true)
    conn.selectServerData(srv)

    const loaded = await method.loadServices()
    if (loaded) {
      saveState()
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (!dropdownOpen.value || !filteredServers.value.length) {
      if (e.key === 'ArrowDown') {
        dropdownOpen.value = true
        acActiveIdx.value = -1
        e.preventDefault()
      }
      return
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        acActiveIdx.value = Math.min(acActiveIdx.value + 1, filteredServers.value.length - 1)
        scrollActiveIntoView()
        break
      case 'ArrowUp':
        e.preventDefault()
        acActiveIdx.value = Math.max(acActiveIdx.value - 1, 0)
        scrollActiveIntoView()
        break
      case 'Enter':
        e.preventDefault()
        if (acActiveIdx.value >= 0) {
          selectServer(filteredServers.value[acActiveIdx.value])
        }
        break
      case 'Escape':
        dropdownOpen.value = false
        acActiveIdx.value = -1
        break
    }
  }

  function scrollActiveIntoView() {
    const items = document.querySelectorAll('#serverDropdown .autocomplete-item')
    items[acActiveIdx.value]?.scrollIntoView({ block: 'nearest' })
  }

  function onMousedownItem(e: MouseEvent) {
    e.preventDefault()
  }
</script>

<template>
  <div class="autocomplete-wrap">
    <div
      v-if="selectedDisplayValue"
      class="method-selected-display"
      @click="focusInput"
    >
      <StartTrimText :text="selectedDisplayValue" />
    </div>
    <input
      id="serverInput"
      ref="inputRef"
      type="text"
      :placeholder="servers.servers.length ? 'Search server...' : 'No servers'"
      :value="inputValue"
      :class="{ 'method-input-hidden': !!selectedDisplayValue }"
      autocomplete="off"
      @focus="onFocus"
      @input="onInput"
      @blur="onBlur"
      @keydown="onKeydown"
    />
    <div id="serverDropdown" class="autocomplete-dropdown" :class="{ open: dropdownOpen }">
      <div
        v-for="(srv, idx) in filteredServers"
        :key="srv.id"
        class="autocomplete-item"
        :class="{ active: idx === acActiveIdx }"
        :data-value="srv.id"
        @mousedown="onMousedownItem"
        @mouseup.left="selectServer(srv)"
      >
        {{ formatServer(srv) }}
      </div>
      <div v-if="filteredServers.length === 0" class="autocomplete-empty">No servers found</div>
    </div>
  </div>
</template>
