<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
  import { useUiStore } from '@/stores/ui'
  import { useResponseStore } from '@/stores/response'
  import { useMethodStore } from '@/stores/method'
  import { formatTime } from '@/utils/dateFormat'
  import type * as Monaco from 'monaco-editor'

  const ui = useUiStore()
  const response = useResponseStore()
  const method = useMethodStore()

  const editorContainer = ref<HTMLElement | null>(null)
  const editor = shallowRef<Monaco.editor.IStandaloneCodeEditor | null>(null)
  const monacoModule = shallowRef<typeof Monaco | null>(null)
  const wrapOn = ref(localStorage.getItem('responseWrap') !== 'off')

  let monacoReady = false

  onMounted(async () => {
    const monaco = await import('monaco-editor')
    monacoModule.value = monaco

    if (!editorContainer.value) return

    monaco.editor.setTheme(ui.theme === 'dark' ? 'vs-dark' : 'vs')

    editor.value = monaco.editor.create(editorContainer.value, {
      value: '',
      language: 'json',
      readOnly: true,
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      folding: true,
      lineNumbers: 'off',
      renderLineHighlight: 'none',
      overviewRulerLanes: 0,
      hideCursorInOverviewRuler: true,
      scrollbar: { vertical: 'auto', horizontal: 'auto' },
      wordWrap: wrapOn.value ? 'on' : 'off',
      automaticLayout: true,
      contextmenu: true,
      domReadOnly: true,
      padding: { top: 12, bottom: 12 },
    })

    monacoReady = true

    // Show response or placeholder
    if (response.responseJson !== null) {
      showResponse(response.responseJson)
    } else {
      editor.value.setValue('// Response will appear here...')
    }
  })

  onUnmounted(() => {
    editor.value?.dispose()
  })

  watch(
    () => ui.theme,
    () => {
      monacoModule.value?.editor.setTheme(ui.theme === 'dark' ? 'vs-dark' : 'vs')
    },
  )

  watch(
    () => response.responseJson,
    (json) => {
      if (!monacoReady) return
      if (json !== null) {
        showResponse(json)
      } else {
        editor.value?.setValue('// Response will appear here...')
      }
    },
  )

  function showResponse(json: string) {
    if (!editor.value) return
    editor.value.setValue(json)
    editor.value.getAction('editor.unfoldAll')?.run()
  }

  function toggleTheme() {
    ui.toggleTheme()
  }

  function toggleWrap() {
    if (!editor.value) return
    wrapOn.value = !wrapOn.value
    const newVal = wrapOn.value ? 'on' : 'off'
    editor.value.updateOptions({ wordWrap: newVal as any })
    localStorage.setItem('responseWrap', newVal)
  }

  function toggleDebug() {
    response.showDebug = !response.showDebug
  }

  function saveResponse() {
    if (response.lastResponseData == null) return
    const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
    let rand = ''
    for (let i = 0; i < 6; i++) rand += chars[Math.floor(Math.random() * chars.length)]
    const svc = method.currentService.split('.').pop() || 'service'
    const filename = `${svc}_${method.currentMethod}-${rand}.json`
    const blob = new Blob([JSON.stringify(response.lastResponseData, null, 2)], {
      type: 'application/json',
    })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = filename
    a.click()
    URL.revokeObjectURL(a.href)
  }

  function debugLines(): string {
    const d = response.debugInfo
    if (!d) return ''
    const lines = [
      `Method:    ${d.method}`,
      `Server:    ${d.requestUrl}`,
      `Status:    ${d.statusCode}${d.statusMessage ? ' \u2014 ' + d.statusMessage : ''}`,
      `Sent:      ${formatTime(d.sentAt)}`,
      `Received:  ${formatTime(d.receivedAt)}`,
      `Duration:  ${d.duration} (${d.durationMs} ms)`,
    ]
    if (d.responseSize != null) lines.push(`Size:      ${d.responseSize} bytes`)
    if (d.error) lines.push(`\nError:\n${d.error}`)
    lines.push(`\n\u2500\u2500 Request (payload) \u2500\u2500`)
    lines.push(JSON.stringify(d.requestPayload, null, 2))
    lines.push(`\n\u2500\u2500 Request metadata \u2500\u2500`)
    lines.push(JSON.stringify(d.requestMeta || {}, null, 2))
    lines.push(`\n\u2500\u2500 Response headers \u2500\u2500`)
    lines.push(JSON.stringify(d.responseHeaders || {}, null, 2))
    lines.push(`\n\u2500\u2500 Response trailers \u2500\u2500`)
    lines.push(JSON.stringify(d.responseTrailers || {}, null, 2))
    return lines.join('\n')
  }
</script>

<template>
  <div class="panel panel-response">
    <div class="panel-title">
      <span>Response</span>
      <div class="panel-title-buttons">
        <button class="btn-save-response" @click="toggleTheme">
          {{ ui.theme === 'dark' ? 'Light' : 'Dark' }}
        </button>
        <button class="btn-save-response" :class="{ active: wrapOn }" @click="toggleWrap">
          Wrap Lines
        </button>
        <button v-show="response.debugInfo" class="btn-save-response" @click="toggleDebug">
          Debug
        </button>
        <button
          v-show="response.lastResponseData != null"
          class="btn-save-response"
          @click="saveResponse"
        >
          Save
        </button>
      </div>
    </div>

    <pre v-show="response.showDebug && response.debugInfo" class="debug-panel">{{
      debugLines()
    }}</pre>

    <div
      ref="editorContainer"
      :class="{ error: response.isError && response.responseJson !== null }"
      style="flex: 1; min-height: 0"
    />
  </div>
</template>
