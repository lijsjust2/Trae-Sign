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
      <div class="count-block">
        <div class="count-num">{{ store.logs.length }}</div>
        <div class="count-lbl">日志条目</div>
      </div>
      <div class="spacer"></div>
      <button class="neu-btn neu-btn-sm neu-btn-danger" @click="clearAll">清空</button>
      <button class="neu-btn neu-btn-sm neu-btn-primary" @click="store.loadLogs()">刷新</button>
    </div>
    <div v-if="store.logs.length === 0" class="empty neu-inset">暂无日志</div>
    <div v-else class="logs neu">
      <div v-for="l in store.logs" :key="l.id" class="log-item">
        <div class="log-left">
          <div class="head">
            <span class="name">{{ l.accountName }}</span>
            <span class="tag" :class="
              l.result === 'success' && l.earned > 0 ? 'tag-ok'
              : l.result === 'already' ? 'tag-info'
              : l.result === 'rate_limited' ? 'tag-warn'
              : 'tag-fail'
            ">
              <template v-if="(l.result === 'success' && l.earned === 0) || l.result === 'already'">已签到</template>
              <template v-else-if="l.result === 'success'">成功</template>
              <template v-else-if="l.result === 'rate_limited'">配额用尽</template>
              <template v-else>失败</template>
            </span>
          </div>
          <div class="body">
            <span v-if="l.result === 'success' || l.result === 'already'">
              <span class="earned">+{{ l.earned }}</span> 积分 · 累计 <span class="remain">{{ l.remain }}</span>
              <span class="msg" v-if="l.earned === 0 && l.message">· {{ l.message }}</span>
            </span>
            <span v-else class="err">{{ l.message }}</span>
          </div>
        </div>
        <div class="log-right">
          <span class="time">{{ fmtTime(l.time) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
.count-block {
  padding: 6px 16px;
  background: var(--primary-soft);
  border: 1px solid #dbe0fc;
  border-radius: 10px;
  display: flex; align-items: baseline; gap: 6px;
}
.count-num { font-size: 18px; font-weight: 700; color: var(--primary); }
.count-lbl { font-size: 11.5px; color: var(--text-soft); }
.spacer { flex: 1; }

.empty { padding: 50px; text-align: center; color: var(--muted); font-size: 13px; }
.logs { overflow: hidden; display: flex; flex-direction: column; }
.log-item {
  display: flex; align-items: center; gap: 14px;
  padding: 14px 20px;
  border-bottom: 1px solid var(--border);
  transition: background .15s ease;
}
.log-item:last-child { border-bottom: none; }
.log-item:hover { background: #fafbff; }

.log-left { flex: 1; min-width: 0; }
.head { display: flex; align-items: center; gap: 10px; margin-bottom: 5px; }
.name { font-weight: 600; font-size: 14.5px; color: var(--text); }
.body { font-size: 13px; color: var(--text-soft); }
.earned {
  font-weight: 700; color: var(--success-fg);
  background: var(--success-bg);
  padding: 1px 8px; border-radius: 6px; margin-right: 2px;
}
.remain { font-weight: 600; color: var(--text); }
.msg { color: var(--muted); font-size: 12px; }
.err {
  font-size: 13px; font-weight: 500;
  color: var(--danger-fg);
  background: var(--danger-bg);
  padding: 4px 10px; border-radius: 6px;
  display: inline-block; max-width: 100%;
}
.log-right { flex-shrink: 0; }
.time { font-size: 12.5px; color: var(--muted); font-variant-numeric: tabular-nums; }
</style>
