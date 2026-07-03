import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ProtoField } from '@/types/proto'
import type { MethodEntry } from '@/types/api'
import * as api from '@/api/endpoints'
import { useConnectionStore } from './connection'
import { useUiStore } from './ui'

export const useMethodStore = defineStore(
  'method',
  () => {
    const allMethods = ref<MethodEntry[]>([])
    const currentService = ref('')
    const currentMethod = ref('')
    const currentFields = ref<ProtoField[]>([])
    const loading = ref(false)
    const messageSchemaCache = ref<Record<string, ProtoField[]>>({})

    const currentMethodValue = computed(() => {
      if (!currentService.value || !currentMethod.value) return ''
      return `${currentService.value}::${currentMethod.value}`
    })

    function cacheSchemas(fields: ProtoField[]) {
      for (const f of fields) {
        if (f.type === 'message' && f.messageType && f.fields?.length) {
          messageSchemaCache.value[f.messageType] = f.fields
          cacheSchemas(f.fields)
        }
        if (
          f.type === 'map' &&
          f.mapValue?.type === 'message' &&
          f.mapValue.messageType &&
          f.mapValue.fields?.length
        ) {
          messageSchemaCache.value[f.mapValue.messageType] = f.mapValue.fields
          cacheSchemas(f.mapValue.fields)
        }
        if (f.type === 'oneof' && f.oneofOptions) {
          for (const opt of f.oneofOptions) {
            if (opt.type === 'message' && opt.messageType && opt.fields?.length) {
              messageSchemaCache.value[opt.messageType] = opt.fields
              cacheSchemas(opt.fields)
            }
          }
        }
      }
    }

    function getMessageFields(
      f: ProtoField | { type: string; messageType?: string; fields?: ProtoField[] },
    ): ProtoField[] {
      if (f.fields?.length) return f.fields
      if ('messageType' in f && f.messageType && messageSchemaCache.value[f.messageType]) {
        return messageSchemaCache.value[f.messageType].map((sf) => ({ ...sf }))
      }
      return []
    }

    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    async function loadServices(initialMethod = '', keepForm = false): Promise<boolean> {
      const conn = useConnectionStore()
      const ui = useUiStore()

      if (!conn.grpcUrl) {
        ui.showStatus('Enter URL', true)
        return false
      }

      if (!keepForm) {
        currentService.value = ''
        currentMethod.value = ''
        currentFields.value = []
      }

      allMethods.value = []
      loading.value = true

      try {
        const services = await api.fetchServices(conn.grpcUrl)

        allMethods.value = []
        for (const { name: svc, methods } of services) {
          const unary = (methods || []).filter((m) => !m.clientStreaming && !m.serverStreaming)
          for (const m of unary) {
            allMethods.value.push({
              svc,
              name: m.name,
              value: `${svc}::${m.name}`,
              deprecated: m.deprecated,
            })
          }
        }

        ui.clearStatus()

        if (!keepForm) {
          ui.activateTab('tab-fields')
        }
        return true
      } catch (e: any) {
        ui.showStatus(`Error: ${e.message}`, true)
        return false
      } finally {
        loading.value = false
      }
    }

    async function loadDescribe(service: string, method: string): Promise<boolean> {
      const conn = useConnectionStore()
      const ui = useUiStore()
      try {
        const fields = await api.fetchDescribe(conn.grpcUrl, service, method)
        cacheSchemas(fields)
        currentFields.value = fields
        return true
      } catch (e: any) {
        ui.showStatus(`Error: ${e.message}`, true)
        return false
      }
    }

    function selectMethod(value: string) {
      if (!value) {
        currentService.value = ''
        currentMethod.value = ''
        currentFields.value = []
        return
      }
      const sep = value.indexOf('::')
      currentService.value = value.slice(0, sep)
      currentMethod.value = value.slice(sep + 2)
    }

    return {
      allMethods,
      currentService,
      currentMethod,
      currentFields,
      loading,
      messageSchemaCache,
      currentMethodValue,
      cacheSchemas,
      getMessageFields,
      loadServices,
      loadDescribe,
      selectMethod,
    }
  },
  {
    persist: {
      pick: ['currentService', 'currentMethod'],
    },
  },
)
