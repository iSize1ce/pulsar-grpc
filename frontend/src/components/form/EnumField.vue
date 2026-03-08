<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { randomValue } from '@/utils/randomValue'
  import { typeLabel } from './fieldUtils'
  import FieldLabel from './FieldLabel.vue'
  import CustomSelect from './CustomSelect.vue'

  const props = defineProps<{
    field: ProtoField
    fieldId: string
  }>()

  const formState = inject<Record<string, any>>('formState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const options = computed(() => {
    const opts: { value: string; label: string }[] = []
    const hasZero = props.field.enumValues?.some((v) => v.number === 0)
    if (!hasZero) {
      opts.push({ value: '', label: '0' })
    }
    for (const v of props.field.enumValues || []) {
      const label = `${v.name} (${v.number})`
      opts.push({ value: v.number === 0 ? '' : v.name, label })
    }
    return opts
  })

  function onSelect(val: string) {
    formState[props.fieldId] = val
    onFormChange()
  }

  function randomize() {
    const val = randomValue(props.field)
    formState[props.fieldId] = val
    onFormChange()
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
  <CustomSelect
    :id="fieldId"
    :options="options"
    :value="formState[fieldId] ?? ''"
    @change="onSelect"
  />
</template>
