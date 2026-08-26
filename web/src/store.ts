import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from './api'
import type { Account, CheckinLog, Settings } from './types'

export const useStore = defineStore('app', () => {
  const accounts = ref<Account[]>([])
  const logs = ref<CheckinLog[]>([])
  const settings = ref<Settings>({ defaultCheckinTime: '08:00', defaultPushplusToken: '', autoCheckin: true })
  const loading = ref(false)
  const toast = ref('')

  let toastTimer: ReturnType<typeof setTimeout>
  function showToast(msg: string) {
    toast.value = msg
    clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toast.value = '' }, 2500)
  }

  async function loadAccounts() { accounts.value = await api.listAccounts() }
  async function loadLogs() { logs.value = await api.listLogs() }
  async function loadSettings() { settings.value = await api.getSettings() }
  async function loadAll() {
    loading.value = true
    try { await Promise.all([loadAccounts(), loadLogs(), loadSettings()]) }
    catch (e: any) { showToast(e.message || '加载失败') }
    finally { loading.value = false }
  }

  return { accounts, logs, settings, loading, toast, loadAccounts, loadLogs, loadSettings, loadAll, showToast }
})
