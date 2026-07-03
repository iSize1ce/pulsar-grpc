<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { useMethodStore } from '@/stores/method'
  import { useUiStore } from '@/stores/ui'
  import { convertLayout } from '@/utils/layoutConvert'

  const methodStore = useMethodStore()
  const ui = useUiStore()

  const inputValue = ref('')
  const dropdownOpen = ref(false)
  const acActiveIdx = ref(-1)
  const inputRef = ref<HTMLInputElement | null>(null)
  const isFocused = ref(false)

  // Sync input when method changes externally (restore from URL, saved requests, history)
  watch(
    () => methodStore.currentMethodValue,
    (val) => {
      if (!dropdownOpen.value) {
        inputValue.value = val
      }
    },
  )
  let methodInputRestoreValue = ''

  const grouped = computed(() => {
    const q = inputValue.value.toLowerCase()
    const qAlt = q ? convertLayout(q) : ''
    const groups: Record<string, { name: string; value: string }[]> = {}

    for (const m of methodStore.allMethods) {
      const lower = m.value.toLowerCase()
      if (q && lower.indexOf(q) === -1 && (!qAlt || lower.indexOf(qAlt) === -1)) continue
      if (!groups[m.svc]) groups[m.svc] = []
      groups[m.svc].push(m)
    }
    return groups
  })

  const flatItems = computed(() => {
    const items: { name: string; value: string }[] = []
    for (const svc of Object.keys(grouped.value)) {
      items.push(...grouped.value[svc])
    }
    return items
  })

  function onFocus() {
    isFocused.value = true
    const selected = methodStore.currentMethodValue
    methodInputRestoreValue = selected
    if (inputValue.value === selected) {
      inputValue.value = ''
    }
    ui.closePopover('methodHistory')
    dropdownOpen.value = true
    acActiveIdx.value = -1
  }

  function onInput(e: Event) {
    inputValue.value = (e.target as HTMLInputElement).value
    ui.closePopover('methodHistory')
    dropdownOpen.value = true
    acActiveIdx.value = -1
  }

  function onBlur() {
    setTimeout(() => {
      isFocused.value = false
      dropdownOpen.value = false
      acActiveIdx.value = -1
      const selected = methodStore.currentMethodValue
      inputValue.value = selected || methodInputRestoreValue
      methodInputRestoreValue = ''
    }, 150)
  }

  function focusInput() {
    inputRef.value?.focus()
  }

  const selectedDisplayValue = computed(() =>
    !isFocused.value && !dropdownOpen.value ? methodStore.currentMethodValue : '',
  )

  async function selectMethod(value: string) {
    if (!value) return
    const currentVal = methodStore.currentMethodValue
    methodStore.selectMethod(value)
    inputValue.value = value
    dropdownOpen.value = false
    acActiveIdx.value = -1
    ui.closePopover('methodHistory')
    inputRef.value?.blur()

    if (value !== currentVal) {
      const sep = value.indexOf('::')
      try {
        await methodStore.loadDescribe(value.slice(0, sep), value.slice(sep + 2))
      } catch (e: any) {
        // Show error in alert when method loading fails
        alert(`Error loading method: ${e.message}`)
        // Revert to previous method if loading failed
        methodStore.selectMethod(currentVal)
        inputValue.value = currentVal
      }
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if (!dropdownOpen.value || !flatItems.value.length) {
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
        acActiveIdx.value = Math.min(acActiveIdx.value + 1, flatItems.value.length - 1)
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
          selectMethod(flatItems.value[acActiveIdx.value].value)
        }
        break
      case 'Escape':
        dropdownOpen.value = false
        acActiveIdx.value = -1
        break
    }
  }

  function scrollActiveIntoView() {
    const items = document.querySelectorAll('#serviceMethodDropdown .autocomplete-item')
    let idx = 0
    for (const svc of Object.keys(grouped.value)) {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      for (const _m of grouped.value[svc]) {
        if (idx === acActiveIdx.value) {
          items[idx]?.scrollIntoView({ block: 'nearest' })
          return
        }
        idx++
      }
    }
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
      :title="selectedDisplayValue"
      @click="focusInput"
    >
      <span class="method-selected-display-text">{{ selectedDisplayValue }}</span>
    </div>
    <input
      id="serviceMethodInput"
      ref="inputRef"
      type="text"
      :placeholder="
        methodStore.loading
          ? 'Loading...'
          : methodStore.allMethods.length
            ? 'Search method...'
            : 'No services'
      "
      :value="inputValue"
      :class="{ 'method-input-hidden': !!selectedDisplayValue }"
      autocomplete="off"
      @focus="onFocus"
      @input="onInput"
      @blur="onBlur"
      @keydown="onKeydown"
    />
    <div id="serviceMethodDropdown" class="autocomplete-dropdown" :class="{ open: dropdownOpen }">
      <template v-for="(methods, svc) in grouped" :key="svc">
        <div class="autocomplete-group">
          {{ svc }}
        </div>
        <div
          v-for="m in methods"
          :key="m.value"
          class="autocomplete-item"
          :class="{ active: flatItems.indexOf(m) === acActiveIdx }"
          :data-value="m.value"
          @mousedown="onMousedownItem"
          @mouseup.left="selectMethod(m.value)"
        >
          {{ m.name }}
        </div>
      </template>
      <div v-if="flatItems.length === 0" class="autocomplete-empty">No methods found</div>
    </div>
  </div>
</template>
