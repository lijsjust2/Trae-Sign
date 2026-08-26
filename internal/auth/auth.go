// Package auth 账号凭证运行时模型（OAuth 登录后由 loginCallback 构建）。
package auth

import "time"

// Auth 归一化后的账号凭证（运行时对象）。
type Auth struct {
	AccessToken   string
	RefreshToken  string
	ExpiresAt     int64 // Unix 秒
	Domain        string
	ApiHost       string
	MachineID     string
	DeviceID      string
	PrivateKeyPEM string // 官方 ExchangeToken 设备签名私钥
	PublicKeyPEM  string // 官方 ExchangeToken 设备公钥
	UID           string
	EnterpriseID  string
	Nickname      string
	Region        string
}

// NeedsRefresh 判断 token 是否在 within 时间内过期。
func (a *Auth) NeedsRefresh(within int64) bool {
	if a.ExpiresAt <= 0 {
		return true
	}
	return time.Now().Unix()+within >= a.ExpiresAt
}
