<script setup lang="ts">
  import { inject, computed, withDefaults } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { isWktScalarField, wktScalarCfg } from '@/utils/wktScalar'
  import { randomValue } from '@/utils/randomValue'
  import { INT_TYPES, UINT_TYPES, FLOAT_TYPES } from '@/utils/protoTypes'
  import { typeLabel } from './fieldUtils'
  import { useMethodStore } from '@/stores/method'
  import FieldLabel from './FieldLabel.vue'
  import CustomSelect from './CustomSelect.vue'
  import BytesField from './BytesField.vue'

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
  const toggleState = inject<Record<string, boolean>>('toggleState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})

  const toggleId = `${props.fieldId}__toggle`

  const isWkt = computed(() => isWktScalarField(props.field))
  const isMessage = computed(() => props.field.type === 'message' && !isWkt.value)
  const isOptionalScalar = computed(() => props.field.optional && !isMessage.value && !isWkt.value)

  const isOn = computed(() => toggleState[toggleId] ?? false)

  function onToggle(on: boolean) {
    toggleState[toggleId] = on
    onFormChange()
  }

  const subFields = computed(() => {
    if (!isMessage.value) return []
    return methodStore.getMessageFields(props.field)
  })

  // WKT scalar input config
  const wktCfg = computed(() => wktScalarCfg(props.field))

  const inputType = computed(() => {
    if (isWkt.value && wktCfg.value) {
      const k = wktCfg.value.kind
      if (['int32', 'int64', 'uint32', 'uint64'].includes(k)) return 'number'
      if (['float', 'double'].includes(k)) return 'number'
      if (k === 'bool') return 'bool'
      if (k === 'bytes') return 'bytes'
      return 'text'
    }
    if (isOptionalScalar.value) {
      if (INT_TYPES.includes(props.field.type) || UINT_TYPES.includes(props.field.type))
        return 'number'
      if (FLOAT_TYPES.includes(props.field.type)) return 'number'
      if (props.field.type === 'bool') return 'bool'
      if (props.field.type === 'bytes') return 'bytes'
      if (props.field.type === 'enum') return 'enum'
      return 'text'
    }
    return 'text'
  })

  const inputPlaceholder = computed(() => {
    if (isWkt.value && wktCfg.value) return wktCfg.value.placeholder || wktCfg.value.label
    return props.field.type
  })

  const inputStep = computed(() => {
    if (isWkt.value && wktCfg.value) {
      if (['float', 'double'].includes(wktCfg.value.kind)) return 'any'
      return '1'
    }
    if (FLOAT_TYPES.includes(props.field.type)) return 'any'
    return '1'
  })

  const inputMin = computed(() => {
    if (isWkt.value && wktCfg.value) {
      if (wktCfg.value.kind.startsWith('u')) return '0'
    }
    if (UINT_TYPES.includes(props.field.type)) return '0'
    return undefined
  })

  const boolOptions = [
    { value: 'false', label: 'false' },
    { value: 'true', label: 'true' },
  ]

  const enumOptions = computed(() => {
    if (props.field.type !== 'enum') return []
    const opts: { value: string; label: string }[] = []
    const hasZero = props.field.enumValues?.some((v) => v.number === 0)
    if (!hasZero) opts.push({ value: '', label: '0' })
    for (const v of props.field.enumValues || []) {
      opts.push({ value: v.number === 0 ? '' : v.name, label: `${v.name} (${v.number})` })
    }
    return opts
  })

  function onInput(e: Event) {
    formState[props.fieldId] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  function onSelectChange(val: string) {
    formState[props.fieldId] = val
    onFormChange()
  }

  function randomize() {
    if (isMessage.value) return
    if (!isOn.value) toggleState[toggleId] = true
    const val = randomValue(props.field)
    if (val || val === 'false') {
      formState[props.fieldId] = val
      onFormChange()
    }
  }

  // Lazy import for recursive component
  import ProtoField_ from './ProtoField.vue'
</script>

<template>
  <FieldLabel
    :field="field"
    :type-str="typeLabel(field)"
    :deprecated="field.deprecated"
    :show-toggle="true"
    :toggle-on="isOn"
    :clickable-type="!isMessage"
    @toggle="onToggle"
    @type-click="randomize"
  />

  <!-- Message: nested fields -->
  <div v-if="isMessage" v-show="isOn" class="field-nested">
    <ProtoField_
      v-for="sf in subFields"
      :key="sf.name"
      :field="sf"
      :prefix="fieldId"
      :depth="depth + 1"
    />
  </div>

  <!-- WKT bool -->
  <template v-else-if="inputType === 'bool'">
    <CustomSelect
      v-show="isOn"
      :id="fieldId"
      :options="boolOptions"
      :value="formState[fieldId] ?? 'false'"
      @change="onSelectChange"
    />
  </template>

  <!-- Optional enum -->
  <template v-else-if="inputType === 'enum'">
    <CustomSelect
      v-show="isOn"
      :id="fieldId"
      :options="enumOptions"
      :value="formState[fieldId] ?? ''"
      @change="onSelectChange"
    />
  </template>

  <!-- Bytes input -->
  <div v-else-if="inputType === 'bytes'" v-show="isOn">
    <BytesField :field="field" :field-id="fieldId" :hide-label="true" />
  </div>

  <!-- Scalar input -->
  <input
    v-else
    v-show="isOn"
    :id="fieldId"
    :type="inputType"
    :placeholder="inputPlaceholder"
    :step="inputStep"
    :min="inputMin"
    :value="formState[fieldId] ?? ''"
    @input="onInput"
  />
</template>
