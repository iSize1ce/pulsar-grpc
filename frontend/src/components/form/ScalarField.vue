<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { INT_TYPES, UINT_TYPES, FLOAT_TYPES } from '@/utils/protoTypes'
  import { randomValue } from '@/utils/randomValue'
  import { typeLabel } from '@/components/form/fieldUtils'
  import FieldLabel from './FieldLabel.vue'

  const props = defineProps<{
    field: ProtoField
    fieldId: string
  }>()

  const formState = inject<Record<string, any>>('formState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const inputType = computed(() => {
    if (INT_TYPES.includes(props.field.type) || UINT_TYPES.includes(props.field.type))
      return 'number'
    if (FLOAT_TYPES.includes(props.field.type)) return 'number'
    return 'text'
  })

  const step = computed(() => {
    if (FLOAT_TYPES.includes(props.field.type)) return 'any'
    return '1'
  })

  const min = computed(() => {
    if (UINT_TYPES.includes(props.field.type)) return '0'
    return undefined
  })

  function onInput(e: Event) {
    formState[props.fieldId] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  function randomize() {
    const val = randomValue(props.field)
    if (val || val === 'false') {
      formState[props.fieldId] = val
      onFormChange()
    }
  }
</script>

<template>
  <FieldLabel
    :field="field"
    :type-str="typeLabel(field)"
    :deprecated="field.deprecated"
    :clickable-type="true"
    @type-click="randomize"
  />
  <input
    :id="fieldId"
    :type="inputType"
    :placeholder="field.type"
    :step="step"
    :min="min"
    :value="formState[fieldId] ?? ''"
    @input="onInput"
  />
</template>
