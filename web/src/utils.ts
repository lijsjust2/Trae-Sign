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
