<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { typeLabel } from './fieldUtils'
  import { useMethodStore } from '@/stores/method'
  import FieldLabel from './FieldLabel.vue'
  import ProtoField_ from './ProtoField.vue'

  const props = withDefaults(
    defineProps<{
      field: ProtoField
      fieldId: string
      depth?: number
    }>(),
    { depth: 0 },
  )

  const methodStore = useMethodStore()
  const repeatedState =
    inject<Record<string, { counter: number; items: number[] }>>('repeatedState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const msgFields = computed(() => methodStore.getMessageFields(props.field))

  function ensureState() {
    if (!repeatedState[props.fieldId]) {
      repeatedState[props.fieldId] = { counter: 0, items: [] }
    }
  }

  function addItem() {
    ensureState()
    const idx = repeatedState[props.fieldId].counter++
    repeatedState[props.fieldId].items.push(idx)
    onFormChange()
  }

  function removeItem(idx: number) {
    ensureState()
    const items = repeatedState[props.fieldId].items
    const pos = items.indexOf(idx)
    if (pos >= 0) items.splice(pos, 1)
    onFormChange()
  }

  const items = computed(() => {
    ensureState()
    return repeatedState[props.fieldId]?.items || []
  })
</script>

<template>
  <FieldLabel
    :field="field"
    :type-str="typeLabel(field)"
    :deprecated="field.deprecated"
    :show-add-btn="true"
    @add="addItem"
  />
  <div class="repeated-messages">
    <div v-for="idx in items" :key="idx" class="repeated-message-item">
      <div class="repeated-message-fields">
        <ProtoField_
          v-for="sf in msgFields"
          :key="sf.name"
          :field="sf"
          :prefix="`${fieldId}_${idx}`"
          :depth="depth + 1"
        />
      </div>
      <button type="button" class="btn-remove" @click="removeItem(idx)">&minus;</button>
    </div>
  </div>
</template>
