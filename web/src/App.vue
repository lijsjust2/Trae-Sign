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
.app {
  min-height: 100vh;
}
/* 顶栏：纯白细边 + 内边距（Argon 风） */
.topbar {
  display: flex; align-items: center; gap: 24px;
  padding: 0 24px;
  height: 60px;
  background: rgba(255,255,255,.85);
  backdrop-filter: saturate(1.4) blur(10px);
  -webkit-backdrop-filter: saturate(1.4) blur(10px);
  border-bottom: 1px solid var(--border);
  position: sticky; top: 0; z-index: 20;
}
.brand {
  font-size: 17px; font-weight: 700;
  background: linear-gradient(135deg, var(--primary), var(--primary-tint));
  -webkit-background-clip: text; background-clip: text; color: transparent;
  letter-spacing: .2px;
}
.tabs { display: flex; gap: 4px; margin-left: 10px; }
.tab {
  background: transparent; color: var(--muted); padding: 8px 16px;
  border-radius: 8px; font-size: 13px; font-weight: 600;
  transition: all .18s ease;
}
.tab:hover { color: var(--primary); background: var(--primary-soft); }
.tab.active {
  color: #fff;
  background: linear-gradient(135deg, var(--primary), var(--primary-tint));
  box-shadow: 0 2px 6px rgba(94,114,228,.25);
}
.logout {
  margin-left: auto;
  background: transparent !important;
  color: var(--danger-fg) !important;
  border: 1px solid transparent !important;
  box-shadow: none !important;
}
.logout:hover { background: var(--danger-bg) !important; border-color: #f3d3da !important; }

.content {
  max-width: 1120px;
  margin: 0 auto;
  padding: 28px 24px 80px;
}

.toast {
  position: fixed; bottom: 32px; left: 50%; transform: translateX(-50%);
  padding: 12px 22px; font-size: 13px; font-weight: 500;
  z-index: 99; max-width: 80vw;
  color: #fff;
  background: linear-gradient(135deg, #1a2140, #313c6e);
  border: 1px solid rgba(255,255,255,.08) !important;
  box-shadow: 0 12px 32px rgba(26,33,64,.35) !important;
  border-radius: 10px !important;
}
.fade-enter-active, .fade-leave-active { transition: opacity .2s, transform .2s; }
.fade-enter-from, .fade-leave-to { opacity: 0; transform: translate(-50%, 14px); }
</style>
