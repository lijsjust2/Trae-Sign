<script setup lang="ts">
import { useStore } from '../store'
import { api } from '../api'
import { fmtTime } from '../utils'

const store = useStore()
async function clearAll() {
  if (!confirm('确认清空所有日志？')) return
  try { await api.clearLogs(); await store.loadLogs() } catch (e: any) { store.showToast(e.message) }
}
</script>

<template>
  <div>
    <div class="toolbar">
      <span class="count">共 {{ store.logs.length }} 条</span>
      <button class="neu-btn neu-btn-sm neu-btn-danger" @click="clearAll">清空</button>
      <button class="neu-btn neu-btn-sm" @click="store.loadLogs()">刷新</button>
    </div>
    <div v-if="store.logs.length === 0" class="empty neu-inset">暂无日志</div>
    <div v-else class="logs">
      <div v-for="l in store.logs" :key="l.id" class="log-item neu">
        <div class="log-head">
          <span class="name">{{ l.accountName }}</span>
          <span class="tag" :class="
            l.result === 'success' && l.earned > 0 ? 'tag-ok'
            : l.result === 'already' ? 'tag-ok'
            : 'tag-fail'
          ">
            <template v-if="l.result === 'success' && l.earned === 0">已签到</template>
            <template v-else-if="l.result === 'success'">成功</template>
            <template v-else-if="l.result === 'already'">已签到</template>
            <template v-else-if="l.result === 'rate_limited'">配额用尽</template>
            <template v-else>失败</template>
          </span>
          <span class="time">{{ fmtTime(l.time) }}</span>
        </div>
        <div class="log-body">
          <span v-if="l.result === 'success' || l.result === 'already'">+{{ l.earned }} 积分 · 累计 {{ l.remain }}</span>
          <span v-else class="err">{{ l.message }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
.count { font-size: 12px; color: var(--muted); margin-right: auto; }
.empty { padding: 40px; text-align: center; color: var(--muted); }
.logs { display: flex; flex-direction: column; gap: 10px; }
.log-item { padding: 12px 14px; }
.log-head { display: flex; align-items: center; gap: 10px; }
.name { font-weight: 600; font-size: 14px; }
.time { font-size: 12px; color: var(--muted); margin-left: auto; }
.log-body { font-size: 13px; color: var(--muted); margin-top: 4px; }
.err { color: var(--danger); }
</style>
