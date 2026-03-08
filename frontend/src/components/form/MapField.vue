<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { isWktScalarField } from '@/utils/wktScalar'
  import { randomValue } from '@/utils/randomValue'
  import { INT_TYPES, UINT_TYPES } from '@/utils/protoTypes'
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
  const formState = inject<Record<string, any>>('formState')!
  const repeatedState =
    inject<Record<string, { counter: number; items: number[] }>>('repeatedState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const valueSpec = computed(() => props.field.mapValue || { type: 'string' })
  const keySpec = computed(() => props.field.mapKey || { type: 'string' })
  const isMsgValue = computed(
    () => valueSpec.value.type === 'message' && !isWktScalarField(valueSpec.value),
  )
  const valueFields = computed(() =>
    isMsgValue.value ? methodStore.getMessageFields(valueSpec.value) : [],
  )

  function ensureState() {
    if (!repeatedState[props.fieldId]) {
      repeatedState[props.fieldId] = { counter: 0, items: [] }
    }
  }

  function addItem() {
    ensureState()
    const idx = repeatedState[props.fieldId].counter++
    repeatedState[props.fieldId].items.push(idx)
  }

  function removeItem(idx: number) {
    ensureState()
    const items = repeatedState[props.fieldId].items
    const pos = items.indexOf(idx)
    if (pos >= 0) items.splice(pos, 1)
    onFormChange()
  }

  function onKeyInput(idx: number, e: Event) {
    formState[`${props.fieldId}_${idx}__key`] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  function onValueInput(idx: number, e: Event) {
    formState[`${props.fieldId}_${idx}__value`] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  const keyInputType = computed(() => {
    const t = keySpec.value.type
    if (INT_TYPES.includes(t) || UINT_TYPES.includes(t)) return 'number'
    return 'text'
  })

  const valueInputType = computed(() => {
    if (isMsgValue.value) return 'none'
    const vs = valueSpec.value
    if (INT_TYPES.includes(vs.type) || UINT_TYPES.includes(vs.type)) return 'number'
    if (['float', 'double'].includes(vs.type)) return 'number'
    return 'text'
  })

  function randomizeAll() {
    ensureState()
    if (!repeatedState[props.fieldId].items.length) {
      addItem()
      addItem()
    }
    for (const idx of repeatedState[props.fieldId].items) {
      const kf = keySpec.value as ProtoField
      const kv = randomValue(kf)
      if (kv || kv === 'false') formState[`${props.fieldId}_${idx}__key`] = kv

      if (!isMsgValue.value) {
        const vf = valueSpec.value as ProtoField
        const vv = randomValue(vf)
        if (vv || vv === 'false') formState[`${props.fieldId}_${idx}__value`] = vv
      }
    }
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
    :clickable-type="true"
    @add="addItem"
    @type-click="randomizeAll"
  />
  <div class="map-field">
    <template v-for="idx in items" :key="idx">
      <!-- Scalar value map -->
      <div v-if="!isMsgValue" class="map-item">
        <input
          :type="keyInputType"
          :placeholder="keySpec.type || 'string'"
          :step="keyInputType === 'number' ? '1' : undefined"
          :value="formState[`${fieldId}_${idx}__key`] ?? ''"
          @input="onKeyInput(idx, $event)"
        />
        <input
          :type="valueInputType"
          :placeholder="valueSpec.type || 'string'"
          :step="valueInputType === 'number' ? 'any' : undefined"
          :value="formState[`${fieldId}_${idx}__value`] ?? ''"
          @input="onValueInput(idx, $event)"
        />
        <button type="button" class="btn-remove" @click="removeItem(idx)">&minus;</button>
      </div>

      <!-- Message value map -->
      <div v-else class="map-message-item">
        <div class="map-item">
          <input
            :type="keyInputType"
            :placeholder="keySpec.type || 'string'"
            :step="keyInputType === 'number' ? '1' : undefined"
            :value="formState[`${fieldId}_${idx}__key`] ?? ''"
            @input="onKeyInput(idx, $event)"
          />
          <button type="button" class="btn-remove" @click="removeItem(idx)">&minus;</button>
        </div>
        <div class="repeated-message-fields">
          <ProtoField_
            v-for="sf in valueFields"
            :key="sf.name"
            :field="sf"
            :prefix="`${fieldId}_${idx}__value`"
            :depth="depth + 1"
          />
        </div>
      </div>
    </template>
  </div>
</template>
