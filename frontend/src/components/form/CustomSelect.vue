<script setup lang="ts">
  import { ref, computed, onMounted, onUnmounted } from 'vue'

  const props = defineProps<{
    id?: string
    options: { value: string; label: string }[]
    value: string
  }>()

  const emit = defineEmits<{
    change: [value: string]
  }>()

  const open = ref(false)
  const activeIndex = ref(-1)
  const wrapRef = ref<HTMLElement | null>(null)

  const selectedLabel = computed(() => {
    const opt = props.options.find((o) => o.value === props.value)
    return opt ? opt.label : ''
  })

  function toggle(e: Event) {
    e.stopPropagation()
    open.value = !open.value
    activeIndex.value = -1
  }

  function selectOption(val: string, e: Event) {
    e.stopPropagation()
    emit('change', val)
    open.value = false
  }

  function onKeydown(e: KeyboardEvent) {
    if (!props.options.length) return
    switch (e.key) {
      case 'ArrowDown':
      case 'ArrowUp':
        e.preventDefault()
        if (!open.value) {
          open.value = true
          return
        }
        if (e.key === 'ArrowDown') {
          activeIndex.value = Math.min(activeIndex.value + 1, props.options.length - 1)
        } else {
          activeIndex.value = Math.max(activeIndex.value - 1, 0)
        }
        break
      case 'Enter':
        e.preventDefault()
        if (open.value && activeIndex.value >= 0) {
          selectOption(props.options[activeIndex.value].value, e)
        } else {
          open.value = true
        }
        break
      case 'Escape':
        open.value = false
        break
    }
  }

  function onDocumentClick(e: MouseEvent) {
    if (wrapRef.value && !wrapRef.value.contains(e.target as Node)) {
      open.value = false
    }
  }

  onMounted(() => {
    document.addEventListener('click', onDocumentClick)
  })
  onUnmounted(() => {
    document.removeEventListener('click', onDocumentClick)
  })
</script>

<template>
  <div ref="wrapRef" class="custom-select" :class="{ open }">
    <select
      :id="id"
      :value="value"
      style="display: none"
      @change="emit('change', ($event.target as HTMLSelectElement).value)"
    >
      <option v-for="opt in options" :key="opt.value" :value="opt.value">
        {{ opt.label }}
      </option>
    </select>
    <button type="button" class="custom-select-trigger" @click="toggle" @keydown="onKeydown">
      {{ selectedLabel }}
    </button>
    <div v-show="open" class="custom-select-dropdown">
      <div
        v-for="(opt, idx) in options"
        :key="opt.value"
        class="custom-select-option"
        :class="{ selected: opt.value === value, active: idx === activeIndex }"
        @click="selectOption(opt.value, $event)"
      >
        {{ opt.label }}
      </div>
    </div>
  </div>
</template>
