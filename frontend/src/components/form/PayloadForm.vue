<script setup lang="ts">
  import { reactive } from 'vue'
  import type { MapComponent, ProtoField } from '@/types/proto'
  import { toCamelCase } from '@/utils/camelCase'
  import { parseFieldVal, zeroVal, oneofDefaultValue } from '@/utils/parseValue'
  import {
    formatFieldValueForInput,
    formatCollectionValueForInput,
    formatOneofValueForInput,
  } from '@/utils/formatValue'
  import { isWktScalarField, wktScalarCfg } from '@/utils/wktScalar'
  import { useMethodStore } from '@/stores/method'
  import ProtoField_ from './ProtoField.vue'

  const props = defineProps<{
    fields: ProtoField[]
  }>()

  const emit = defineEmits<{
    change: []
  }>()

  const methodStore = useMethodStore()

  const MAX_DEPTH = 5

  // Reactive form state: flat map from fieldId → value
  const formState = reactive<Record<string, any>>({})
  // Track toggles separately
  const toggleState = reactive<Record<string, boolean>>({})
  // Track oneof selections
  const oneofState = reactive<Record<string, string>>({})
  // Track repeated items: fieldId → { counter, items: index[] }
  const repeatedState = reactive<Record<string, { counter: number; items: number[] }>>({})

  function resetForm() {
    Object.keys(formState).forEach((k) => delete formState[k])
    Object.keys(toggleState).forEach((k) => delete toggleState[k])
    Object.keys(oneofState).forEach((k) => delete oneofState[k])
    Object.keys(repeatedState).forEach((k) => delete repeatedState[k])
  }

  function collectPayload(fields?: ProtoField[], prefix = '', depth = 0): Record<string, any> {
    if (depth >= MAX_DEPTH) return {}
    const ff = fields || props.fields
    const obj: Record<string, any> = {}

    for (const f of ff) {
      const id = prefix ? `${prefix}_${f.name}` : f.name
      const jsonKey = toCamelCase(f.name)

      if (f.type === 'oneof') {
        collectOneofPayload(obj, f, id, depth)
        continue
      }
      if (f.type === 'map') {
        collectMapPayload(obj, f, id, jsonKey, depth)
        continue
      }
      if (f.repeated) {
        collectRepeatedPayload(obj, f, id, jsonKey, depth)
        continue
      }

      if (f.type === 'message') {
        if (!isToggleOn(id)) continue
        if (isWktScalarField(f)) {
          const v = formState[id]
          if (v === '' || v == null) {
            const cfg = wktScalarCfg(f)
            obj[jsonKey] = cfg ? zeroVal(cfg.kind) : ''
            continue
          }
          const parsed = parseFieldVal(v, f)
          if (parsed !== undefined) obj[jsonKey] = parsed
          continue
        }
        obj[jsonKey] = collectPayload(methodStore.getMessageFields(f), id, depth + 1)
        continue
      }

      if (f.optional) {
        if (!isToggleOn(id)) continue
        const v = formState[id]
        if (v === '' || v == null) {
          obj[jsonKey] = zeroVal(f.type)
          continue
        }
        const parsed = parseFieldVal(v, f)
        if (parsed !== undefined) obj[jsonKey] = parsed
        continue
      }

      // Plain scalar
      const v = formState[id]
      if (v === '' || v == null) continue
      const parsed = parseFieldVal(v, f)
      if (parsed !== undefined && parsed !== zeroVal(f.type)) obj[jsonKey] = parsed
    }
    return withFlatDefaultsIfEmpty(obj, ff)
  }

  function withFlatDefaultsIfEmpty(
    obj: Record<string, any>,
    fields: ProtoField[],
  ): Record<string, any> {
    if (Object.keys(obj).length || !fields.length) return obj
    return collectFlatDefaultPayload(fields)
  }

  function collectFlatDefaultPayload(fields: ProtoField[]): Record<string, any> {
    const obj: Record<string, any> = {}
    for (const f of fields) {
      if (f.type === 'oneof') continue
      obj[toCamelCase(f.name)] = defaultFieldValue(f)
    }
    return obj
  }

  function defaultFieldValue(f: ProtoField | MapComponent): any {
    const wkt = wktScalarCfg(f)
    if (wkt) {
      if (wkt.kind === 'jsonArray') return []
      if (wkt.kind === 'jsonObject' || wkt.kind === 'jsonValue') return null
      return zeroVal(wkt.kind)
    }

    if ('repeated' in f && f.repeated) return []
    if (f.type === 'map' || f.type === 'message') return null
    if (f.type === 'enum') {
      const values = ('enumValues' in f ? f.enumValues : undefined) || []
      return values.find((v) => v.number === 0)?.name || values[0]?.name || ''
    }
    return zeroVal(f.type)
  }

  function collectOneofPayload(obj: Record<string, any>, f: ProtoField, id: string, depth: number) {
    const selName = oneofState[id]
    if (!selName) return
    const opt = f.oneofOptions?.find((o) => o.name === selName)
    if (!opt) return
    const optKey = toCamelCase(selName)

    if (opt.type === 'message') {
      if (isWktScalarField(opt)) {
        const v = formState[`${id}_${selName}`]
        if (v === '' || v == null) {
          obj[optKey] = oneofDefaultValue(opt)
          return
        }
        const parsed = parseFieldVal(v, opt)
        obj[optKey] = parsed === undefined ? oneofDefaultValue(opt) : parsed
        return
      }
      obj[optKey] = collectPayload(methodStore.getMessageFields(opt), `${id}_${selName}`, depth + 1)
      return
    }

    const v = formState[`${id}_${selName}`]
    if (v === '' || v == null) {
      obj[optKey] = oneofDefaultValue(opt)
      return
    }
    const parsed = parseFieldVal(v, opt)
    obj[optKey] = parsed === undefined ? oneofDefaultValue(opt) : parsed
  }

  function collectRepeatedPayload(
    obj: Record<string, any>,
    f: ProtoField,
    id: string,
    jsonKey: string,
    depth: number,
  ) {
    const state = repeatedState[id]
    if (!state?.items.length) return

    if (f.type === 'message' && !isWktScalarField(f)) {
      const msgFields = methodStore.getMessageFields(f)
      const vals = state.items.map((idx) => collectPayload(msgFields, `${id}_${idx}`, depth + 1))
      if (vals.length) obj[jsonKey] = vals
      return
    }

    const vals = state.items.map((idx) => {
      const v = formState[`${id}_${idx}`]
      return v !== '' && v != null ? parseFieldVal(v, f) : zeroVal(f.type)
    })
    if (vals.length) obj[jsonKey] = vals
  }

  function collectMapPayload(
    obj: Record<string, any>,
    f: ProtoField,
    id: string,
    jsonKey: string,
    depth: number,
  ) {
    const state = repeatedState[id]
    if (!state?.items.length) return

    const mapValue = f.mapValue || { type: 'string' }
    const mapKey = f.mapKey || { type: 'string' }
    const out: Record<string, any> = {}
    const isMsgValue = mapValue.type === 'message' && !isWktScalarField(mapValue)

    for (const idx of state.items) {
      const keyRaw = formState[`${id}_${idx}__key`]
      const keyVal = parseFieldVal(keyRaw, mapKey)
      const key = keyVal == null ? '' : String(keyVal)

      if (isMsgValue) {
        out[key] = collectPayload(
          methodStore.getMessageFields(mapValue),
          `${id}_${idx}__value`,
          depth + 1,
        )
      } else {
        const rawVal = formState[`${id}_${idx}__value`]
        const parsed =
          rawVal === '' || rawVal == null ? zeroVal(mapValue.type) : parseFieldVal(rawVal, mapValue)
        out[key] = parsed === undefined ? zeroVal(mapValue.type) : parsed
      }
    }
    if (Object.keys(out).length) obj[jsonKey] = out
  }

  function isToggleOn(id: string): boolean {
    return !!toggleState[`${id}__toggle`]
  }

  // Fill form from payload data
  function fillForm(data: any, fields?: ProtoField[], prefix?: string, depth = 0) {
    if (!data || typeof data !== 'object') return
    if (depth >= MAX_DEPTH) return
    const ff = fields || props.fields

    for (const f of ff) {
      const id = prefix ? `${prefix}_${f.name}` : f.name
      const jsonKey = toCamelCase(f.name)

      if (f.type === 'oneof') {
        fillOneof(f, id, data, depth)
        continue
      }
      if (f.type === 'map') {
        fillMap(f, id, jsonKey, data, depth)
        continue
      }
      if (f.repeated) {
        fillRepeated(f, id, jsonKey, data, depth)
        continue
      }

      if (f.type === 'message' && !isWktScalarField(f)) {
        const val = data[jsonKey]
        if (val == null || typeof val !== 'object') continue
        toggleState[`${id}__toggle`] = true
        fillForm(val, methodStore.getMessageFields(f), id, depth + 1)
        continue
      }

      if (isWktScalarField(f)) {
        if (!(jsonKey in data)) continue
        const val = data[jsonKey]
        if (val == null) continue
        toggleState[`${id}__toggle`] = true
        formState[id] = formatFieldValueForInput(val, f)
        continue
      }

      if (f.optional) {
        if (!(jsonKey in data)) continue
        toggleState[`${id}__toggle`] = true
        formState[id] = formatFieldValueForInput(data[jsonKey], f)
        continue
      }

      if (jsonKey in data && data[jsonKey] != null) {
        formState[id] = formatFieldValueForInput(data[jsonKey], f)
      }
    }
  }

  function fillOneof(f: ProtoField, id: string, data: any, depth: number) {
    if (!f.oneofOptions) return
    for (const opt of f.oneofOptions) {
      const optKey = toCamelCase(opt.name)
      if (!(optKey in data)) continue
      const val = data[optKey]

      oneofState[id] = opt.name
      if (opt.type === 'message' && !isWktScalarField(opt) && opt.fields) {
        fillForm(val, opt.fields, `${id}_${opt.name}`, depth + 1)
      } else {
        formState[`${id}_${opt.name}`] = formatOneofValueForInput(val, opt)
      }
      break
    }
  }

  function fillRepeated(f: ProtoField, id: string, jsonKey: string, data: any, depth: number) {
    const val = data[jsonKey]
    if (!Array.isArray(val) || !val.length) return

    if (!repeatedState[id]) {
      repeatedState[id] = { counter: 0, items: [] }
    }

    if (f.type === 'message' && !isWktScalarField(f)) {
      const msgFields = methodStore.getMessageFields(f)
      for (const item of val) {
        const idx = repeatedState[id].counter++
        repeatedState[id].items.push(idx)
        fillForm(item, msgFields, `${id}_${idx}`, depth + 1)
      }
      return
    }

    for (const v of val) {
      const idx = repeatedState[id].counter++
      repeatedState[id].items.push(idx)
      formState[`${id}_${idx}`] = formatCollectionValueForInput(v, f)
    }
  }

  function fillMap(f: ProtoField, id: string, jsonKey: string, data: any, depth: number) {
    const val = data[jsonKey]
    if (val == null || typeof val !== 'object' || Array.isArray(val)) return
    const entries = Object.entries(val)
    if (!entries.length) return

    if (!repeatedState[id]) {
      repeatedState[id] = { counter: 0, items: [] }
    }

    const valueSpec = f.mapValue || { type: 'string' }
    const isMsgValue = valueSpec.type === 'message' && !isWktScalarField(valueSpec)

    for (const [key, value] of entries) {
      const idx = repeatedState[id].counter++
      repeatedState[id].items.push(idx)
      formState[`${id}_${idx}__key`] = key

      if (isMsgValue) {
        if (value && typeof value === 'object') {
          fillForm(value, methodStore.getMessageFields(valueSpec), `${id}_${idx}__value`, depth + 1)
        }
      } else {
        formState[`${id}_${idx}__value`] = formatCollectionValueForInput(value, valueSpec)
      }
    }
  }

  // Provide state and helpers to child components
  import { provide } from 'vue'
  provide('formState', formState)
  provide('toggleState', toggleState)
  provide('oneofState', oneofState)
  provide('repeatedState', repeatedState)
  provide('onFormChange', () => emit('change'))

  defineExpose({ collectPayload, fillForm, resetForm })
</script>

<template>
  <div id="requestForm">
    <p v-if="!fields.length" class="muted">No fields (empty request)</p>
    <ProtoField_ v-for="f in fields" :key="f.name" :field="f" prefix="" />
  </div>
</template>
