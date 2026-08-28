<script setup lang="ts">
import type { Account } from '../types'
import { fmtTime, accountName } from '../utils'

defineProps<{ account: Account }>()
const emit = defineEmits<{ checkin: [], edit: [], del: [], points: [] }>()
</script>

<template>
  <div class="card neu">
    <div class="head">
      <div class="name">{{ accountName(account) }}</div>
      <span class="tag" :class="account.enabled ? 'tag-ok' : 'tag-muted'">{{ account.enabled ? '启用' : '禁用' }}</span>
    </div>
    <div class="meta">
      <span>昵称：{{ account.nickname || '—' }}</span>
      <span>签到时间：{{ account.checkinTime || '默认' }}</span>
      <span>推送：{{ account.pushplusToken ? '已设置' : '默认值' }}</span>
    </div>
    <div class="stats">
      <div class="stat neu-inset">
        <div class="subl">累计积分</div>
        <div class="num">{{ account.totalCredits }}</div>
      </div>
      <div class="stat neu-inset">
        <div class="subl">今日签到</div>
        <div class="num">+{{ account.todayEarned }}</div>
      </div>
    </div>
    <div class="last">
      <span class="tag" :class="account.todayStatus === '已签到' ? 'tag-ok' : account.todayStatus === '失败' ? 'tag-fail' : account.todayStatus === '配额用尽' ? 'tag-warn' : 'tag-muted'">
        {{ account.todayStatus }}
      </span>
      <span class="time">{{ fmtTime(account.lastCheckinAt) }}</span>
      <span class="msg" v-if="account.lastCheckinMessage">{{ account.lastCheckinMessage }}</span>
    </div>
    <div class="actions">
      <button class="neu-btn neu-btn-sm neu-btn-primary" @click="emit('checkin')">签到</button>
      <button class="neu-btn neu-btn-sm" @click="emit('points')">刷新积分</button>
      <button class="neu-btn neu-btn-sm" @click="emit('edit')">编辑</button>
      <button class="neu-btn neu-btn-sm neu-btn-danger" @click="emit('del')">删除</button>
    </div>
  </div>
</template>

<style scoped>
.card { padding: 20px 22px; display: flex; flex-direction: column; gap: 14px; }
.head { display: flex; align-items: center; justify-content: space-between; }
.name { font-size: 17px; font-weight: 700; color: var(--text); letter-spacing: .1px; }

.meta {
  display: flex; flex-wrap: wrap; gap: 4px 20px;
  font-size: 12.5px; color: var(--text-soft);
  padding: 10px 14px;
  background: linear-gradient(180deg, #fafbff 0%, #f6f8ff 100%);
  border: 1px solid #ececfb;
  border-radius: 10px;
}

.stats { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.stat {
  padding: 14px 14px 12px;
  border-radius: 10px;
  display: flex; flex-direction: column; align-items: center; gap: 4px;
  overflow: hidden; position: relative;
}
.subl { font-size: 11.5px; letter-spacing: .2px; color: var(--muted); font-weight: 500; }
.num { font-size: 22px; font-weight: 700; color: var(--primary); letter-spacing: .2px; }

.last { display: flex; align-items: center; gap: 8px; font-size: 12.5px; color: var(--muted); flex-wrap: wrap; }
.msg { color: var(--muted); }
.actions { display: flex; gap: 8px; flex-wrap: wrap; padding-top: 2px; }
.actions :deep(.neu-btn-sm) { height: 30px; line-height: 1; }
</style>
