<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

  const props = defineProps<{
    text: string
    title?: string
  }>()

  const containerRef = ref<HTMLElement | null>(null)
  const contentRef = ref<HTMLElement | null>(null)
  const offsetPx = ref(0)
  let resizeObserver: ResizeObserver | null = null

  function measure() {
    const container = containerRef.value
    const content = contentRef.value
    if (!container || !content) {
      offsetPx.value = 0
      return
    }

    offsetPx.value = Math.max(0, Math.ceil(content.scrollWidth - container.clientWidth))
  }

  watch(
    () => props.text,
    async () => {
      await nextTick()
      measure()
    },
  )

  onMounted(async () => {
    await nextTick()
    measure()

    resizeObserver = new ResizeObserver(() => {
      measure()
    })

    if (containerRef.value) resizeObserver.observe(containerRef.value)
    if (contentRef.value) resizeObserver.observe(contentRef.value)
  })

  onBeforeUnmount(() => {
    resizeObserver?.disconnect()
    resizeObserver = null
  })

  const contentStyle = computed(() =>
    offsetPx.value ? { transform: `translateX(-${offsetPx.value}px)` } : undefined,
  )
</script>

<template>
  <span ref="containerRef" class="start-trim-text" :title="title || text">
    <span ref="contentRef" class="start-trim-text-content" :style="contentStyle">
      {{ text }}
    </span>
  </span>
</template>
