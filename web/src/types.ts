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
