import { defineStore } from 'pinia'
import { ref } from 'vue'

export type TabId = 'tab-server' | 'tab-saved' | 'tab-history' | 'tab-fields'

export const useUiStore = defineStore(
  'ui',
  () => {
    const activeTab = ref<TabId>('tab-server')
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
