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
      <span>推送：{{ account.pushplusToken ? '已设置' : '未设置' }}</span>
    </div>
    <div class="stats">
      <div class="stat neu-inset">
        <div class="num">{{ account.totalCredits }}</div>
        <div class="lbl">累计积分</div>
      </div>
      <div class="stat neu-inset">
        <div class="num">+{{ account.todayEarned }}</div>
        <div class="lbl">今日签到</div>
      </div>
    </div>
    <div class="last">
      <span class="tag" :class="account.todayStatus === '已签到' ? 'tag-ok' : account.todayStatus === '失败' || account.todayStatus === '配额用尽' ? 'tag-fail' : 'tag-muted'">
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
.card { padding: 16px 18px; display: flex; flex-direction: column; gap: 12px; }
.head { display: flex; align-items: center; justify-content: space-between; }
.name { font-size: 16px; font-weight: 600; }
.meta { display: flex; flex-wrap: wrap; gap: 6px 18px; font-size: 12px; color: var(--muted); }
.stats { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.stat { padding: 10px 12px; text-align: center; }
.num { font-size: 20px; font-weight: 700; color: var(--brand); }
.lbl { font-size: 11px; color: var(--muted); margin-top: 2px; }
.last { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--muted); flex-wrap: wrap; }
.msg { color: var(--faint); }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
</style>
