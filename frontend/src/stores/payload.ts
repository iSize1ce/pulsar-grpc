import { defineStore } from 'pinia'
import { ref } from 'vue'

export const usePayloadStore = defineStore(
  'payload',
  () => {
    const jsonMode = ref(false)
    const savedPayload = ref<Record<string, any> | null>(null)
    const restoring = ref(false)
    const pendingRestore = ref<any>(null)
    const filling = ref(false)
    const requestPayloadCache = ref('{}')
    const requestPayloadCacheValid = ref(false)

    function setJsonMode(mode: boolean) {
      jsonMode.value = mode
    }

    return {
      jsonMode,
      savedPayload,
      restoring,
      pendingRestore,
      filling,
      requestPayloadCache,
      requestPayloadCacheValid,
      setJsonMode,
    }
  },
  {
    persist: {
      pick: ['jsonMode', 'requestPayloadCache', 'requestPayloadCacheValid'],
    },
  },
)
