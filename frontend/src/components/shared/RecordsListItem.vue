<script setup lang="ts">
  import StartTrimText from '@/components/shared/StartTrimText.vue'

  defineProps<{
    title: string
    subtitle?: string
    metaLeft?: string
    metaRight?: string
    statusClass?: string
    active?: boolean
    titleTrimStart?: boolean
  }>()

  const emit = defineEmits<{
    click: []
    delete: []
  }>()
</script>

<template>
  <div class="records-item" :class="{ active }" @click="emit('click')">
    <div class="records-item-content">
      <div class="records-item-title">
        <span v-if="statusClass" class="history-status-dot" :class="statusClass" />
        <span class="records-item-title-text" :class="{ 'trim-start': titleTrimStart }">
          <StartTrimText v-if="titleTrimStart" :text="title" />
          <span v-else class="records-item-title-text-inner">{{ title }}</span>
        </span>
      </div>
      <div v-if="metaLeft || metaRight" class="records-item-meta">
        <div v-if="metaLeft" class="records-item-subtitle">
          {{ metaLeft }}
        </div>
        <div v-if="metaRight" class="records-item-meta-right">
          {{ metaRight }}
        </div>
      </div>
      <div v-else-if="subtitle" class="records-item-subtitle">
        {{ subtitle }}
      </div>
    </div>
    <button type="button" class="btn-remove btn-remove-records-item" @click.stop="emit('delete')">
      &minus;
    </button>
  </div>
</template>
