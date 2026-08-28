<script setup lang="ts">
import { ref } from 'vue'
import { useStore } from '../store'
import { api } from '../api'
import { accountName } from '../utils'
import AccountCard from './AccountCard.vue'
import AddAccountModal from './AddAccountModal.vue'
import EditAccountModal from './EditAccountModal.vue'
import type { Account } from '../types'

const store = useStore()
const showAdd = ref(false)
const editTarget = ref<Account | null>(null)
const busy = ref(false)

async function checkinOne(a: Account) {
  busy.value = true
  try {
    const r = await api.checkinOne(a.id)
    store.showToast(`${accountName(a)}：${r.detail}`)
    await store.loadAccounts(); await store.loadLogs()
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

async function checkinAll() {
  busy.value = true
  store.showToast('开始全部签到…')
  try {
    const rs = await api.checkinAll()
    const ok = rs.filter(r => r.status === 'OK').length
    store.showToast(`完成：成功 ${ok} / ${rs.length}`)
    await store.loadAccounts(); await store.loadLogs()
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

async function getPoints(a: Account) {
  busy.value = true
  try {
    const r = await api.getPoints(a.id)
    store.showToast(`${accountName(a)}：当前积分 ${r.totalPoints}`)
    await store.loadAccounts()
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

async function del(a: Account) {
  if (!confirm(`确认删除账号 ${accountName(a)}？`)) return
  try {
    await api.deleteAccount(a.id)
    store.showToast('已删除')
    await store.loadAccounts()
  } catch (e: any) { store.showToast(e.message) }
}

// ===== 配置导出 / 导入 =====
const fileInput = ref<HTMLInputElement | null>(null)

async function exportConfig() {
  if (store.accounts.length === 0) { store.showToast('没有账号可导出'); return }
  if (!confirm('导出文件包含账号登录凭证和设备密钥，请妥善保管。继续导出？')) return
  busy.value = true
  try {
    const text = await api.exportAccounts()
    const blob = new Blob([text], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `trae-signin-accounts-${new Date().toISOString().slice(0, 10).replace(/-/g, '')}.json`
    a.click()
    URL.revokeObjectURL(url)
    store.showToast('已导出')
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}

function pickImportFile() {
  fileInput.value?.click()
}

async function onImportFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = '' // 允许重复选择同一文件
  if (!file) return
  if (!confirm(`导入 ${file.name}？相同账号将被覆盖，新账号会被添加。`)) return
  busy.value = true
  try {
    const content = await file.text()
    const r = await api.importAccounts(content)
    store.showToast(`导入完成：新增 ${r.added}，更新 ${r.updated}`)
    await store.loadAccounts(); await store.loadLogs()
  } catch (e: any) { store.showToast(e.message) }
  finally { busy.value = false }
}
</script>

<template>
  <div class="list">
    <div class="toolbar neu">
      <div class="tool-left">
        <button class="neu-btn neu-btn-primary" @click="showAdd = true">
          <span class="plus">+</span> 添加账号
        </button>
        <button class="neu-btn" :disabled="busy || store.accounts.length === 0" @click="checkinAll">
          一键签到
        </button>
        <div class="divider"></div>
        <button class="neu-btn" :disabled="busy || store.accounts.length === 0" @click="exportConfig">
          导出配置
        </button>
        <button class="neu-btn" :disabled="busy" @click="pickImportFile">导入配置</button>
        <input ref="fileInput" type="file" accept=".json,application/json" style="display:none" @change="onImportFile" />
      </div>
      <div class="tool-right">
        <div class="stat">
          <div class="stat-num">{{ store.accounts.length }}</div>
          <div class="stat-lbl">账号</div>
        </div>
      </div>
    </div>

    <div v-if="store.accounts.length === 0" class="empty neu">
      <div class="empty-icon">T</div>
      <div class="empty-title">还没有账号</div>
      <div class="empty-sub">点击右上角 <b>添加账号</b>，通过 OAuth 网页登录添加你的第一个 TRAE 账号</div>
    </div>
    <div v-else class="grid">
      <AccountCard
        v-for="a in store.accounts" :key="a.id" :account="a"
        @checkin="checkinOne(a)" @points="getPoints(a)"
        @edit="editTarget = a" @del="del(a)"
      />
    </div>

    <AddAccountModal v-if="showAdd" @close="showAdd = false" @done="showAdd = false; store.loadAccounts()" />
    <EditAccountModal v-if="editTarget" :account="editTarget"
      @close="editTarget = null"
      @done="editTarget = null; store.loadAccounts()" />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; margin-bottom: 22px; gap: 12px;
}
.tool-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.plus { font-weight: 700; margin-right: 2px; font-size: 14px; }
.divider { width: 1px; height: 20px; background: var(--border); margin: 0 4px; }
.tool-right { display: flex; align-items: center; }
.stat {
  padding: 6px 14px;
  background: var(--primary-soft);
  border: 1px solid #dbe0fc;
  border-radius: 10px;
  display: flex; align-items: baseline; gap: 6px;
}
.stat-num { font-size: 18px; font-weight: 700; color: var(--primary); line-height: 1; }
.stat-lbl { font-size: 11.5px; color: var(--text-soft); }

.empty {
  padding: 56px 32px; text-align: center;
  display: flex; flex-direction: column; align-items: center; gap: 10px;
}
.empty-icon {
  width: 56px; height: 56px; border-radius: 16px;
  background: linear-gradient(135deg, var(--primary), var(--primary-tint));
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 24px; font-weight: 800; letter-spacing: 1px;
  box-shadow: 0 8px 22px rgba(94,114,228,.3);
  margin-bottom: 4px;
}
.empty-title { font-size: 17px; font-weight: 700; color: var(--text); }
.empty-sub { font-size: 13px; color: var(--muted); max-width: 460px; line-height: 1.6; }

.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 18px; }
</style>
