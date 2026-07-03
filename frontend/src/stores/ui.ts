import { defineStore } from 'pinia'
import { ref } from 'vue'

export type TabId = 'tab-saved' | 'tab-history' | 'tab-fields'

sanitizePersistedActiveTab()

export const useUiStore = defineStore(
  'ui',
  () => {
    const activeTab = ref<TabId>('tab-fields')
    const theme = ref<'dark' | 'light'>(
      localStorage.getItem('theme') !== 'light' ? 'dark' : 'light',
    )
    const statusMessage = ref('')
    const statusIsError = ref(false)
    const openPopoverName = ref<string | null>(null)

    function activateTab(tabId: TabId) {
      activeTab.value = tabId
    }

    function toggleTheme() {
      theme.value = theme.value === 'dark' ? 'light' : 'dark'
      localStorage.setItem('theme', theme.value)
      document.documentElement.setAttribute('data-theme', theme.value)
    }

    function applyTheme() {
      document.documentElement.setAttribute('data-theme', theme.value)
    }

    function showStatus(msg: string, isError = false) {
      statusMessage.value = msg
      statusIsError.value = isError
    }

    function clearStatus() {
      statusMessage.value = ''
      statusIsError.value = false
    }

    function openPopover(name: string) {
      openPopoverName.value = name
    }

    function closePopover(name: string) {
      if (openPopoverName.value === name) {
        openPopoverName.value = null
      }
    }

    function closeAllPopovers() {
      openPopoverName.value = null
    }

    function isPopoverOpen(name: string): boolean {
      return openPopoverName.value === name
    }

    return {
      activeTab,
      theme,
      statusMessage,
      statusIsError,
      openPopoverName,
      activateTab,
      toggleTheme,
      applyTheme,
      showStatus,
      clearStatus,
      openPopover,
      closePopover,
      closeAllPopovers,
      isPopoverOpen,
    }
  },
  {
    persist: {
      pick: ['activeTab'],
    },
  },
)

function sanitizePersistedActiveTab() {
  if (typeof localStorage === 'undefined') return

  const raw = localStorage.getItem('ui')
  if (!raw) return

  try {
    const persisted = JSON.parse(raw)
    if (!persisted || typeof persisted !== 'object') return
    if (persisted.activeTab !== 'tab-server') return
    persisted.activeTab = 'tab-fields'
    localStorage.setItem('ui', JSON.stringify(persisted))
  } catch {
    /* Ignore malformed persisted state; Pinia will fall back to defaults. */
  }
}
