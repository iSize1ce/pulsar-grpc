<script setup lang="ts">
  import { ref, inject, withDefaults } from 'vue'
  import type { ProtoField } from '@/types/proto'
  import { randomValue } from '@/utils/randomValue'
  import { typeLabel } from './fieldUtils'
  import { useUiStore } from '@/stores/ui'
  import FieldLabel from './FieldLabel.vue'

  const props = withDefaults(
    defineProps<{
      field: ProtoField
      fieldId: string
      hideLabel?: boolean
    }>(),
    { hideLabel: false },
  )

  const formState = inject<Record<string, any>>('formState')!
  const onFormChange = inject<() => void>('onFormChange', () => {})
  const ui = useUiStore()

  const fileInputRef = ref<HTMLInputElement | null>(null)
  const hasFile = ref(false)
  const fileName = ref('')
  const disabled = ref(false)

  function onTextInput(e: Event) {
    formState[props.fieldId] = (e.target as HTMLInputElement).value
    onFormChange()
  }

  function onFileClick() {
    fileInputRef.value?.click()
  }

  function onFileChange(e: Event) {
    const input = e.target as HTMLInputElement
    if (input.files?.[0]) readFileToBase64(input.files[0])
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault()
  }

  function onDrop(e: DragEvent) {
    e.preventDefault()
    if (e.dataTransfer?.files.length) {
      readFileToBase64(e.dataTransfer.files[0])
    }
  }

  function readFileToBase64(file: File) {
    const reader = new FileReader()
    reader.onload = () => {
      const base64 = (reader.result as string).split(',')[1]
      formState[props.fieldId] = base64
      formState[`${props.fieldId}__fileBase64`] = base64
      hasFile.value = true
      fileName.value = file.name
      disabled.value = true
      onFormChange()
    }
    reader.onerror = () => ui.showStatus('Failed to read file', true)
    reader.readAsDataURL(file)
  }

  function clearFile() {
    formState[props.fieldId] = ''
    delete formState[`${props.fieldId}__fileBase64`]
    hasFile.value = false
    fileName.value = ''
    disabled.value = false
    if (fileInputRef.value) fileInputRef.value.value = ''
    onFormChange()
  }

  function randomize() {
    const val = randomValue(props.field)
    if (val) {
      formState[props.fieldId] = val
      onFormChange()
    }
  }
</script>

<template>
  <FieldLabel
    v-if="!hideLabel"
    :field="field"
    :type-str="typeLabel(field)"
    :deprecated="field.deprecated"
    :clickable-type="true"
    @type-click="randomize"
  />
  <div class="bytes-input-wrap">
    <div class="bytes-input-row">
      <div class="bytes-inp-wrap" :class="{ 'has-clear': hasFile }">
        <span v-show="hasFile" class="bytes-clear" @click="clearFile">&times;</span>
        <input
          :id="fieldId"
          type="text"
          :placeholder="hasFile ? fileName : 'base64'"
          :disabled="disabled"
          :value="hasFile ? '' : (formState[fieldId] ?? '')"
          @input="onTextInput"
        />
      </div>
      <div
        class="bytes-drop-btn"
        :class="{ 'has-file': hasFile }"
        @click="onFileClick"
        @dragover="onDragOver"
        @drop="onDrop"
      >
        File
      </div>
      <input
        ref="fileInputRef"
        type="file"
        style="position: absolute; width: 0; height: 0; overflow: hidden; opacity: 0"
        @change="onFileChange"
      />
    </div>
  </div>
</template>
