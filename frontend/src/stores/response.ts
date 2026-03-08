import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DebugInfo } from '@/types/api'

export const useResponseStore = defineStore(
  'response',
  () => {
    const responseJson = ref<string | null>(null)
    const isError = ref(false)
    const debugInfo = ref<DebugInfo | null>(null)
    const showDebug = ref(false)
    const lastResponseData = ref<any>(null)

    function setResponse(json: string, error: boolean) {
      responseJson.value = json
      isError.value = error
    }

    function clearResponse() {
      responseJson.value = null
      isError.value = false
      debugInfo.value = null
      showDebug.value = false
      lastResponseData.value = null
    }

    return {
      responseJson,
      isError,
      debugInfo,
      showDebug,
      lastResponseData,
      setResponse,
      clearResponse,
    }
  },
  {
    persist: {
      pick: ['responseJson', 'isError', 'debugInfo', 'showDebug', 'lastResponseData'],
    },
  },
)
