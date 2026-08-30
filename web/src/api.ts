// 后端 API 封装
import type { Account, CheckinLog, CheckinResult, LoginURLResp, Settings, AuthStatus } from './types'

const BASE = '/api'

// 会话过期时的全局回调（由 App 注册，跳回登录页）
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: () => void) { onUnauthorized = fn }

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json' },
    ...options
  })
  if (!res.ok) {
    if (res.status === 401 && onUnauthorized) onUnauthorized()
    let msg = `HTTP ${res.status}`
    try {
      const j = await res.json()
      if (j.error) msg = j.error
    } catch { /* ignore */ }
    throw new Error(msg)
  }
  return res.json() as Promise<T>
}

export const api = {
  authStatus: () => req<AuthStatus>('/auth/status'),
  authSetup: (username: string, password: string) =>
    req<{ ok: boolean }>('/auth/setup', { method: 'POST', body: JSON.stringify({ username, password }) }),
  authLogin: (username: string, password: string) =>
    req<{ ok: boolean }>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  authLogout: () => req<{ ok: boolean }>('/auth/logout', { method: 'POST' }),
  authChange: (oldPassword: string, newUsername: string, newPassword: string) =>
    req<{ ok: boolean }>('/auth/change', { method: 'POST', body: JSON.stringify({ oldPassword, newUsername, newPassword }) }),
  listAccounts: () => req<Account[]>('/accounts'),
  // 导出账号配置（返回原始 JSON 文本，前端存为文件）
  exportAccounts: async (): Promise<string> => {
    const res = await fetch(BASE + '/accounts/export')
    if (!res.ok) {
      if (res.status === 401 && onUnauthorized) onUnauthorized()
      throw new Error(`导出失败: HTTP ${res.status}`)
    }
    return res.text()
  },
  // 导入账号配置（JSON 文本，数组或 {accounts:[...]} 格式）
  importAccounts: (content: string) =>
    req<{ added: number; updated: number }>('/accounts/import', { method: 'POST', body: content }),
  updateAccount: (id: string, patch: Record<string, any>) =>
    req<Account>(`/accounts/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),
  deleteAccount: (id: string) =>
    req<{ ok: boolean }>(`/accounts/${id}`, { method: 'DELETE' }),
  checkinOne: (id: string) =>
    req<CheckinResult>(`/accounts/${id}/checkin`, { method: 'POST' }),
  checkinAll: () =>
    req<CheckinResult[]>('/accounts/checkin-all', { method: 'POST' }),
  getPoints: (id: string) =>
    req<{ success: boolean; totalPoints: number }>(`/accounts/${id}/points`),
  listLogs: (limit = 100) => req<CheckinLog[]>(`/logs?limit=${limit}`),
  clearLogs: () => req<{ ok: boolean }>('/logs', { method: 'DELETE' }),
  getSettings: () => req<Settings>('/settings'),
  saveSettings: (s: Settings) =>
    req<Settings>('/settings', { method: 'POST', body: JSON.stringify(s) }),
  // 测试推送（token 为空时后端用已保存的默认 token）
  testPush: (token: string) =>
    req<{ ok: boolean; title: string; content: string }>('/settings/test-push', { method: 'POST', body: JSON.stringify({ token }) }),
  loginURL: (remark: string, checkinTime: string, pushplusToken: string) =>
    req<LoginURLResp>('/login/url', { method: 'POST', body: JSON.stringify({ remark, checkinTime, pushplusToken }) }),
  loginCallback: (callbackUrl: string, machineId: string, deviceId: string, privateKeyPem: string, publicKeyPem: string, remark: string, checkinTime: string, pushplusToken: string) =>
    req<Account>('/login/callback', { method: 'POST', body: JSON.stringify({ callbackUrl, machineId, deviceId, privateKeyPem, publicKeyPem, remark, checkinTime, pushplusToken }) })
}
