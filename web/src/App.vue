<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useStore } from './store'
import { api, setUnauthorizedHandler } from './api'
import AuthGate from './components/AuthGate.vue'
import AccountList from './components/AccountList.vue'
import CheckinLog from './components/CheckinLog.vue'
import SettingsPanel from './components/SettingsPanel.vue'

const store = useStore()
const tab = ref<'accounts' | 'logs' | 'settings'>('accounts')
const authed = ref(false)

setUnauthorizedHandler(() => { authed.value = false })

function onAuthed() {
  authed.value = true
  store.loadAll()
}

async function logout() {
  try { await api.authLogout() } catch { /* ignore */ }
  authed.value = false
}

onMounted(() => {
  // 已有会话则直接进入
  api.authStatus().then(s => { if (s.loggedIn) onAuthed() }).catch(() => {})
})
</script>

<template>
  <AuthGate v-if="!authed" @authed="onAuthed" />
  <div v-else class="app">
    <header class="topbar">
      <div class="brand">TRAE 签到</div>
      <nav class="tabs">
        <button class="tab" :class="{ active: tab === 'accounts' }" @click="tab = 'accounts'">账号</button>
        <button class="tab" :class="{ active: tab === 'logs' }" @click="tab = 'logs'; store.loadLogs()">日志</button>
        <button class="tab" :class="{ active: tab === 'settings' }" @click="tab = 'settings'">设置</button>
      </nav>
      <button class="tab logout" @click="logout">退出登录</button>
    </header>
    <main class="content">
      <AccountList v-if="tab === 'accounts'" />
      <CheckinLog v-else-if="tab === 'logs'" />
      <SettingsPanel v-else />
    </main>
    <transition name="fade">
      <div v-if="store.toast" class="toast neu">{{ store.toast }}</div>
    </transition>
  </div>
</template>

<style scoped>
.app { max-width: 960px; margin: 0 auto; padding: 24px 20px 48px; }
.topbar { display: flex; align-items: center; gap: 24px; margin-bottom: 24px; }
.brand { font-size: 20px; font-weight: 700; color: var(--brand); }
.tabs { display: flex; gap: 8px; }
.tab {
  background: transparent; color: var(--muted); padding: 6px 14px;
  border-radius: 999px; font-size: 13px; font-weight: 500;
}
.tab.active { color: var(--brand); background: var(--brand-soft); }
.logout { margin-left: auto; }
.toast {
  position: fixed; bottom: 28px; left: 50%; transform: translateX(-50%);
  padding: 12px 22px; font-size: 13px; z-index: 99; max-width: 80vw;
}
.fade-enter-active, .fade-leave-active { transition: opacity .2s, transform .2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translate(-50%, 10px); }
</style>
