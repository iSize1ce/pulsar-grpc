<script setup lang="ts">
  import { inject, computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { isWktScalarField } from '@/utils/wktScalar'
  import { randomValue } from '@/utils/randomValue'
  import { typeLabel } from './fieldUtils'
  import FieldLabel from './FieldLabel.vue'
  import CustomSelect from './CustomSelect.vue'
  import ProtoField_ from './ProtoField.vue'

  const props = withDefaults(
    defineProps<{
      field: ProtoField
      fieldId: string
      depth?: number
    }>(),
    { depth: 0 },
  )

  const formState = inject<Record<string, any>>('formState')!
  const oneofState = inject<Record<string, string>>('oneofState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const options = computed(() => {
    const opts = [{ value: '', label: 'null' }]
    for (const opt of props.field.oneofOptions || []) {
      opts.push({ value: opt.name, label: `${opt.name} (${typeLabel(opt)})` })
    }
    return opts
  })

  const selectedName = computed(() => oneofState[props.fieldId] || '')

  const selectedOption = computed(() =>
    props.field.oneofOptions?.find((o) => o.name === selectedName.value),
  )

  const isSelectedMessage = computed(
    () =>
      selectedOption.value?.type === 'message' &&
      !isWktScalarField(selectedOption.value) &&
      !!selectedOption.value?.fields?.length,
  )

  function onSelect(val: string) {
    oneofState[props.fieldId] = val
    onFormChange()
  }

  function randomize() {
    const opts = props.field.oneofOptions
    if (!opts?.length) return
    const opt = opts[Math.floor(Math.random() * opts.length)]
    oneofState[props.fieldId] = opt.name
    if (opt.type !== 'message' || isWktScalarField(opt)) {
      const val = randomValue(opt)
      if (val || val === 'false') {
        formState[`${props.fieldId}_${opt.name}`] = val
      }
    }
    onFormChange()
  }
</script>

<template>
  <FieldLabel
    :field="field"
    type-str="oneof"
    :deprecated="field.deprecated"
    :clickable-type="true"
    @type-click="randomize"
  />
  <CustomSelect :options="options" :value="selectedName" @change="onSelect" />
  <div v-if="selectedName" class="oneof-fields">
    <!-- Message variant: render sub-fields -->
    <template v-if="isSelectedMessage && selectedOption">
      <ProtoField_
        v-for="sf in selectedOption.fields"
        :key="sf.name"
        :field="sf"
        :prefix="`${fieldId}_${selectedName}`"
        :depth="depth + 1"
      />
    </template>
    <!-- Scalar variant: render input for the selected option -->
    <template v-else-if="selectedOption">
      <ProtoField_
        :field="{ ...selectedOption, name: selectedOption.name }"
        :prefix="fieldId"
        :depth="depth"
      />
    </template>
  </div>
</template>
