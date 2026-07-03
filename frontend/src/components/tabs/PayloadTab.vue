<script setup lang="ts">
  import { ref, watch, computed, inject, onMounted, onUnmounted, shallowRef, nextTick } from 'vue'
  import { useMethodStore } from '@/stores/method'
  import { usePayloadStore } from '@/stores/payload'
  import { useConnectionStore } from '@/stores/connection'
  import { useUiStore } from '@/stores/ui'
  import { protoFieldsToJsonSchema } from '@/utils/jsonSchema'
  import MethodAutocomplete from '@/components/method/MethodAutocomplete.vue'
  import MethodHistoryPopover from '@/components/method/MethodHistoryPopover.vue'
  import ServerAutocomplete from '@/components/server/ServerAutocomplete.vue'
  import AddServerPopover from '@/components/server/AddServerPopover.vue'
  import ServerMetadataPopover from '@/components/server/ServerMetadataPopover.vue'
  import PayloadForm from '@/components/form/PayloadForm.vue'
  import type * as Monaco from 'monaco-editor'

  const methodStore = useMethodStore()
  const payload = usePayloadStore()
  const conn = useConnectionStore()
  const ui = useUiStore()

  const saveState = inject<() => void>('saveState', () => {})

  const REQUEST_SCHEMA_URI = 'inmemory://grpc-request-schema.json'

  // Monaco request editor
  const jsonEditorContainer = ref<HTMLElement | null>(null)
  const requestEditor = shallowRef<Monaco.editor.IStandaloneCodeEditor | null>(null)
  const monacoModule = shallowRef<typeof Monaco | null>(null)
  const jsonError = ref('')
  let requestEditorSyncing = false
  let jsonSyncTimer: ReturnType<typeof setTimeout> | null = null

  // Form ref for payload collection
  const payloadFormRef = ref<InstanceType<typeof PayloadForm> | null>(null)

  // Debounced save
  let saveTimer: ReturnType<typeof setTimeout> | null = null
  function debouncedSave() {
    if (payload.filling) return
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(saveState, 300)
  }

  // Collect payload from form or editor
  function collectPayloadFromForm(): Record<string, any> {
    if (!payloadFormRef.value) return {}
    return payloadFormRef.value.collectPayload()
  }

  function getPayload(): Record<string, any> {
    if (payload.jsonMode && requestEditor.value) {
      try {
        return JSON.parse(requestEditor.value.getValue().trim() || '{}')
      } catch {
        return {}
      }
    }
    return methodStore.currentFields.length ? collectPayloadFromForm() : {}
  }

  function clearForm() {
    if (payloadFormRef.value) {
      payloadFormRef.value.resetForm()
    }
    cacheRequestPayload({}, true)
  }

  // Expose for parent
  defineExpose({ getPayload, clearForm })

  function refreshRequestPayloadCache(forceEditorSync = false): Record<string, any> {
    const p = methodStore.currentFields.length ? collectPayloadFromForm() : {}
    cacheRequestPayload(p, forceEditorSync)
    return p
  }

  function cacheRequestPayload(p: Record<string, any>, forceEditorSync = false) {
    payload.requestPayloadCache = JSON.stringify(p || {}, null, 2)
    payload.requestPayloadCacheValid = true
    if (!payload.jsonMode || forceEditorSync) {
      setRequestEditorValue(payload.requestPayloadCache)
      jsonError.value = ''
    }
  }

  function setRequestEditorValue(value: string) {
    if (!requestEditor.value) return
    if (requestEditor.value.getValue() === value) return
    requestEditorSyncing = true
    requestEditor.value.setValue(value)
    requestEditorSyncing = false
  }

  // Monaco init
  onMounted(async () => {
    const monaco = await import('monaco-editor')
    monacoModule.value = monaco

    if (!jsonEditorContainer.value) return

    const wrapSetting = localStorage.getItem('responseWrap') !== 'off' ? 'on' : 'off'

    const requestModel = monaco.editor.createModel(
      '{}',
      'json',
      monaco.Uri.parse(REQUEST_SCHEMA_URI),
    )
    requestEditor.value = monaco.editor.create(jsonEditorContainer.value, {
      model: requestModel,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      folding: true,
      lineNumbers: 'off',
      renderLineHighlight: 'none',
      overviewRulerLanes: 0,
      scrollbar: { vertical: 'auto', horizontal: 'auto' },
      wordWrap: wrapSetting as any,
      automaticLayout: true,
      contextmenu: true,
      tabSize: 2,
      padding: { top: 12, bottom: 12 },
      suggest: { showWords: false },
      quickSuggestions: { strings: true, other: true, comments: false },
    })

    requestEditor.value.onDidChangeModelContent(() => {
      if (requestEditorSyncing) return
      if (jsonSyncTimer) clearTimeout(jsonSyncTimer)
      jsonSyncTimer = setTimeout(() => {
        const text = requestEditor.value!.getValue().trim()
        try {
          JSON.parse(text || '{}')
          jsonError.value = ''
        } catch (e: any) {
          jsonError.value = e.message
        }
        debouncedSave()
      }, 300)
    })

    if (payload.requestPayloadCacheValid) {
      setRequestEditorValue(payload.requestPayloadCache)
    }

    // Apply JSON schema on mount in case fields are already loaded (e.g. after hard reload)
    if (methodStore.currentFields.length) {
      const schema = protoFieldsToJsonSchema(
        methodStore.currentFields,
        methodStore.getMessageFields,
      )
      ;(schema as any).$schema = 'http://json-schema.org/draft-07/schema#'
      monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        schemas: [
          {
            uri: REQUEST_SCHEMA_URI,
            fileMatch: [REQUEST_SCHEMA_URI],
            schema: schema,
          },
        ],
      })
    }
  })

  onUnmounted(() => {
    requestEditor.value?.dispose()
  })

  // Watch theme changes for Monaco
  watch(
    () => ui.theme,
    () => {
      monacoModule.value?.editor.setTheme(ui.theme === 'dark' ? 'vs-dark' : 'vs')
    },
  )

  // Update Monaco JSON schema when fields change
  watch(
    () => methodStore.currentFields,
    (fields) => {
      if (!monacoModule.value) return
      const schema = protoFieldsToJsonSchema(fields, methodStore.getMessageFields)
      ;(schema as any).$schema = 'http://json-schema.org/draft-07/schema#'
      monacoModule.value.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        schemas: [
          {
            uri: REQUEST_SCHEMA_URI,
            fileMatch: [REQUEST_SCHEMA_URI],
            schema: schema,
          },
        ],
      })
    },
    { deep: true },
  )

  // Build form when fields change
  watch(
    () => methodStore.currentFields,
    async () => {
      if (!methodStore.currentFields.length) return
      // When the method schema changes we want JSON mode to reflect the new method immediately,
      // instead of keeping the previous method payload in the editor.
      let forceEditorSync = payload.jsonMode
      await nextTick()
      if (payloadFormRef.value) {
        payloadFormRef.value.resetForm()
        if (payload.savedPayload) {
          payload.filling = true
          payloadFormRef.value.fillForm(payload.savedPayload)
          payload.savedPayload = null
          payload.filling = false
          forceEditorSync = true
        }
      }
      await nextTick()
      refreshRequestPayloadCache(forceEditorSync)
      saveState()
    },
  )

  // JSON mode toggle
  function setJsonMode(mode: boolean) {
    if (payload.jsonMode === mode) return
    payload.setJsonMode(mode)
    if (mode) {
      refreshRequestPayloadCache(true)
      jsonError.value = ''
      if (requestEditor.value) requestEditor.value.layout()
    } else {
      applyJsonToForm()
    }
  }

  function applyJsonToForm() {
    if (!requestEditor.value) return
    const text = requestEditor.value.getValue().trim()
    let data: any
    try {
      data = JSON.parse(text || '{}')
      jsonError.value = ''
    } catch (e: any) {
      jsonError.value = e.message
      return
    }
    if (!methodStore.currentFields.length) return
    // Rebuild and fill
    if (payloadFormRef.value) {
      payloadFormRef.value.resetForm()
      payload.filling = true
      payloadFormRef.value.fillForm(data)
      payload.filling = false
      refreshRequestPayloadCache()
    }
  }

  let formSyncTimer: ReturnType<typeof setTimeout> | null = null
  function onFormChange() {
    if (formSyncTimer) clearTimeout(formSyncTimer)
    formSyncTimer = setTimeout(() => refreshRequestPayloadCache(), 300)
    debouncedSave()
  }

  const hasFields = computed(() => methodStore.currentFields.length > 0)
  const hasServer = computed(() => !!conn.grpcUrl || !!conn.currentServerId)
</script>

<template>
  <div
    id="tab-fields"
    class="tab-pane"
    data-payload-tab
    :class="{ 'json-active': payload.jsonMode && hasFields }"
  >
    <div id="serverSelectSection" class="section selector-section">
      <label>Server</label>
      <div class="method-controls">
        <ServerAutocomplete />
        <ServerMetadataPopover />
        <AddServerPopover />
      </div>
      <div
        v-show="ui.statusMessage && ui.statusIsError"
        id="connectStatus"
        class="status"
        :class="{ error: ui.statusIsError, success: !ui.statusIsError }"
      >
        {{ ui.statusMessage }}
      </div>
    </div>

    <div v-show="!hasServer" class="payload-empty">Select a server</div>

    <div v-show="hasServer" id="serviceMethodSection" class="section selector-section">
      <label>Request</label>
      <div class="method-controls">
        <MethodAutocomplete />
      </div>
    </div>

    <div v-show="hasFields" class="fields-container" :class="{ 'json-active': payload.jsonMode }">
      <div class="view-mode-tabs" aria-label="Payload view mode">
        <div class="view-mode-toggle-group">
          <button
            type="button"
            class="btn-json-toggle"
            :class="{ active: !payload.jsonMode }"
            @click="setJsonMode(false)"
          >
            Form
          </button>
          <button
            type="button"
            class="btn-json-toggle"
            :class="{ active: payload.jsonMode }"
            @click="setJsonMode(true)"
          >
            JSON
          </button>
        </div>
        <MethodHistoryPopover />
      </div>

      <div v-show="!payload.jsonMode">
        <PayloadForm
          ref="payloadFormRef"
          :fields="methodStore.currentFields"
          @change="onFormChange"
        />
      </div>

      <div v-show="payload.jsonMode" class="json-editor-wrap">
        <div ref="jsonEditorContainer" class="json-editor-container" />
        <div v-show="jsonError" class="json-error">
          {{ jsonError }}
        </div>
      </div>
    </div>
  </div>
</template>
