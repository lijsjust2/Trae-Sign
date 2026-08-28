import type { Account } from './types'

export function fmtTime(ms: number): string {
  if (!ms) return '—'
  const d = new Date(ms)
  return d.toLocaleString('zh-CN', {
    month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit'
  })
}

export function accountName(a: Account): string {
  return a.remark || a.nickname || a.uid || a.id
}

// isSameDay 判断毫秒戳是否是今天
function isSameDay(ms: number): boolean {
  if (!ms) return false
  const d = new Date(ms), n = new Date()
  return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate()
}

// todayCheckinStatus 账号今日签到状态文案
//   'success' 已签到  / 'rate_limited' 配额用尽  / 'failed' 失败  / 'pending' 未签到
export function todayCheckinStatus(a: Account): string {
  if (!a.lastCheckinAt || !isSameDay(a.lastCheckinAt)) return '未签到'
  switch (a.lastCheckinResult) {
    case 'success': return '已签到'
    case 'rate_limited': return '配额用尽'
    case 'failed': return '失败'
    default: return '未签到'
  }
}

// todayEarned 今日签到获得积分；非今日签到则返回 0
export function todayEarned(a: Account): number {
  if (!a.lastCheckinAt || !isSameDay(a.lastCheckinAt)) return 0
  return a.lastEarned
}
