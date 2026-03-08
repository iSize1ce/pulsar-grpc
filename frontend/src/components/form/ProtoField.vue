<script setup lang="ts">
  import { computed } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { isWktScalarField } from '@/utils/wktScalar'
  import ScalarField from './ScalarField.vue'
  import BoolField from './BoolField.vue'
  import EnumField from './EnumField.vue'
  import BytesField from './BytesField.vue'
  import MessageField from './MessageField.vue'
  import OneofField from './OneofField.vue'
  import RepeatedScalarField from './RepeatedScalarField.vue'
  import RepeatedMessageField from './RepeatedMessageField.vue'
  import MapField from './MapField.vue'

  const MAX_DEPTH = 5

  const props = withDefaults(
    defineProps<{
      field: ProtoField
      prefix: string
      depth?: number
    }>(),
    { depth: 0 },
  )

  const fieldId = computed(() =>
    props.prefix ? `${props.prefix}_${props.field.name}` : props.field.name,
  )

  const isScalar = computed(() => {
    const t = props.field.type
    return [
      'string',
      'int32',
      'int64',
      'uint32',
      'uint64',
      'sint32',
      'sint64',
      'fixed32',
      'fixed64',
      'sfixed32',
      'sfixed64',
      'float',
      'double',
    ].includes(t)
  })

  const isWktScalar = computed(() => isWktScalarField(props.field))
</script>

<template>
  <div class="field-group">
    <!-- Depth limit reached -->
    <p v-if="depth >= MAX_DEPTH" class="muted">...</p>

    <!-- Oneof -->
    <OneofField
      v-else-if="field.type === 'oneof'"
      :field="field"
      :field-id="fieldId"
      :depth="depth"
    />

    <!-- Map -->
    <MapField v-else-if="field.type === 'map'" :field="field" :field-id="fieldId" :depth="depth" />

    <!-- Repeated message -->
    <RepeatedMessageField
      v-else-if="field.repeated && field.type === 'message' && !isWktScalar"
      :field="field"
      :field-id="fieldId"
      :depth="depth"
    />

    <!-- Repeated scalar/enum/bool/wkt -->
    <RepeatedScalarField v-else-if="field.repeated" :field="field" :field-id="fieldId" />

    <!-- Message (toggleable) or WKT scalar (toggleable) -->
    <MessageField
      v-else-if="field.type === 'message'"
      :field="field"
      :field-id="fieldId"
      :depth="depth"
    />

    <!-- Optional scalar (toggleable) -->
    <MessageField v-else-if="field.optional" :field="field" :field-id="fieldId" :depth="depth" />

    <!-- Bool -->
    <BoolField v-else-if="field.type === 'bool'" :field="field" :field-id="fieldId" />

    <!-- Enum -->
    <EnumField v-else-if="field.type === 'enum'" :field="field" :field-id="fieldId" />

    <!-- Bytes -->
    <BytesField v-else-if="field.type === 'bytes'" :field="field" :field-id="fieldId" />

    <!-- Scalar (string, int, float, etc) -->
    <ScalarField v-else-if="isScalar" :field="field" :field-id="fieldId" />
  </div>
</template>
