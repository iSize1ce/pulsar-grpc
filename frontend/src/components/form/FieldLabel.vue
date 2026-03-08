<script setup lang="ts">
  import type { ProtoField } from '@/types/proto'

  const props = defineProps<{
    field: ProtoField
    typeStr: string
    deprecated?: boolean
    showToggle?: boolean
    toggleOn?: boolean
    showAddBtn?: boolean
    clickableType?: boolean
  }>()

  const emit = defineEmits<{
    toggle: [on: boolean]
    add: []
    typeClick: []
  }>()

  function onToggle() {
    emit('toggle', !props.toggleOn)
  }
</script>

<template>
  <div class="label-row">
    <label :class="{ 'label-deprecated': deprecated }">{{ field.name }}</label>
    <span v-if="deprecated" class="badge-deprecated">deprecated</span>
    <span
      class="field-type"
      :class="{ 'field-type-clickable': clickableType }"
      :title="clickableType ? 'Randomize' : undefined"
      @click="clickableType && emit('typeClick')"
      >{{ typeStr }}</span
    >
    <button
      v-if="showToggle"
      type="button"
      class="btn-toggle"
      :class="{ active: toggleOn }"
      :data-on="String(!!toggleOn)"
      @click="onToggle"
    >
      {{ toggleOn ? '\u2713' : '\u25CB' }}
    </button>
    <button v-if="showAddBtn" type="button" class="btn-add" @click="emit('add')">+</button>
  </div>
</template>
