// 与后端 store.PublicAccount / CheckinLog / Settings / checkin.Result 对齐（脱敏）
export interface Account {
  id: string
  uid: string
  nickname: string
  remark: string
  enabled: boolean
  checkinTime: string
  pushplusToken: string
  hasRefreshToken: boolean
  totalCredits: number
  lastCheckinAt: number
  lastCheckinResult: string
  lastCheckinMessage: string
  lastEarned: number
  todayEarned: number   // 从今日 success 日志聚合，不被 already 覆盖
  todayStatus: string   // 今日签到状态（已签到/未签到/配额用尽/失败）
  createdAt: number
}

export interface CheckinLog {
  id: string
  accountId: string
  accountName: string
  time: number
  result: string
  message: string
  earned: number
  remain: number
}

export interface Settings {
  defaultCheckinTime: string
  defaultPushplusToken: string
  autoCheckin: boolean
}

export interface CheckinResult {
  accountId: string
  name: string
  status: string // "OK" | "ALREADY" | "FAIL"
  earned: number
  remain: number
  detail: string
}

export interface LoginURLResp {
  loginUrl: string
  machineId: string
  deviceId: string
  privateKeyPem: string
  publicKeyPem: string
}

export interface AuthStatus {
  initialized: boolean
  loggedIn: boolean
  username: string
}
