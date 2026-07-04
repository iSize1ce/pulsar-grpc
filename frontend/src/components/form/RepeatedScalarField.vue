<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { randomValue } from '@/utils/randomValue'
  import { typeLabel } from './fieldUtils'
  import { wktScalarCfg } from '@/utils/wktScalar'
  import { INT_TYPES, UINT_TYPES, FLOAT_TYPES } from '@/utils/protoTypes'
  import FieldLabel from './FieldLabel.vue'
  import CustomSelect from './CustomSelect.vue'

  const props = defineProps<{
    field: ProtoField
    fieldId: string
  }>()

  const formState = inject<Record<string, any>>('formState')!
  const repeatedState =
    inject<Record<string, { counter: number; items: number[] }>>('repeatedState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

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

  function onInput(idx: number, e: Event) {
    formState[`${props.fieldId}_${idx}`] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  function onSelectChange(idx: number, val: string) {
    formState[`${props.fieldId}_${idx}`] = val
    onFormChange()
  }

  const inputType = computed(() => {
    const wkt = wktScalarCfg(props.field)
    if (wkt) {
      if (['int32', 'int64', 'uint32', 'uint64'].includes(wkt.kind)) return 'number'
      if (['float', 'double'].includes(wkt.kind)) return 'number'
      if (wkt.kind === 'bool') return 'bool'
      return 'text'
    }
    if (props.field.type === 'bool') return 'bool'
    if (props.field.type === 'enum') return 'enum'
    if (INT_TYPES.includes(props.field.type) || UINT_TYPES.includes(props.field.type))
      return 'number'
    if (FLOAT_TYPES.includes(props.field.type)) return 'number'
    return 'text'
  })

  const boolOptions = [
    { value: 'false', label: 'false' },
    { value: 'true', label: 'true' },
  ]

  const enumOptions = computed(() => {
    const opts: { value: string; label: string }[] = []
    for (const v of props.field.enumValues || []) {
      opts.push({ value: v.name, label: `${v.name} (${v.number})` })
    }
    return opts
  })

  function randomizeAll() {
    ensureState()
    if (!repeatedState[props.fieldId].items.length) {
      addItem()
      addItem()
    }
    for (const idx of repeatedState[props.fieldId].items) {
      const val = randomValue(props.field)
      if (val || val === 'false') {
        formState[`${props.fieldId}_${idx}`] = val
      }
    }
    onFormChange()
  }

  const items = computed(() => {
    ensureState()
    return repeatedState[props.fieldId]?.items || []
  })

  const isFloat = computed(() => FLOAT_TYPES.includes(props.field.type))
  const isUnsigned = computed(() => UINT_TYPES.includes(props.field.type))
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
  <div class="repeated-field">
    <div v-for="idx in items" :key="idx" class="repeated-item">
      <!-- Bool select -->
      <CustomSelect
        v-if="inputType === 'bool'"
        :options="boolOptions"
        :value="formState[`${fieldId}_${idx}`] ?? 'false'"
        @change="onSelectChange(idx, $event)"
      />
      <!-- Enum select -->
      <CustomSelect
        v-else-if="inputType === 'enum'"
        :options="enumOptions"
        :value="formState[`${fieldId}_${idx}`] ?? field.enumValues?.[0]?.name ?? ''"
        @change="onSelectChange(idx, $event)"
      />
      <!-- Scalar input -->
      <input
        v-else
        :type="inputType"
        :placeholder="field.type"
        :step="isFloat ? 'any' : '1'"
        :min="isUnsigned ? '0' : undefined"
        :value="formState[`${fieldId}_${idx}`] ?? ''"
        @input="onInput(idx, $event)"
      />
      <button type="button" class="btn-remove" @click="removeItem(idx)">&minus;</button>
    </div>
  </div>
</template>
