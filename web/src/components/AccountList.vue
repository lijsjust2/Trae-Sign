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
</script>

<template>
  <div class="list">
    <div class="toolbar">
      <button class="neu-btn neu-btn-primary" @click="showAdd = true">+ 添加账号</button>
      <button class="neu-btn" :disabled="busy || store.accounts.length === 0" @click="checkinAll">
        一键签到
      </button>
      <span class="count">共 {{ store.accounts.length }} 个账号</span>
    </div>

    <div v-if="store.accounts.length === 0" class="empty neu-inset">
      还没有账号，点「添加账号」开始
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
.toolbar { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; }
.count { font-size: 12px; color: var(--muted); }
.empty { padding: 48px; text-align: center; color: var(--muted); font-size: 14px; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px; }
</style>
