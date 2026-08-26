// Package store 账号、日志、设置的持久化（JSON 文件 + 互斥锁）。
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"trae-signin-web/internal/auth"
)

const maxLogs = 500

// Account 账号完整记录（含凭证，仅服务端内部使用）。
type Account struct {
	ID                 string `json:"id"`
	UID                string `json:"uid"`
	Nickname           string `json:"nickname"`
	Remark             string `json:"remark"`
	AccessToken       string `json:"accessToken"`
	RefreshToken       string `json:"refreshToken"`
	ExpiresAt          int64  `json:"expiresAt"`
	ApiHost            string `json:"apiHost"`
	Domain             string `json:"domain"`
	MachineID          string `json:"machineId"`
	DeviceID           string `json:"deviceId"`
	// 官方 ExchangeToken 流程所需设备签名密钥（从 TRAE 客户端实例导入）。
	// 为空时刷新会返回 NO_DEVICE_KEY 错误。
	PrivateKeyPEM      string `json:"privateKeyPem,omitempty"`
	PublicKeyPEM       string `json:"publicKeyPem,omitempty"`
	Region             string `json:"region"`
	Enabled            bool   `json:"enabled"`
	CheckinTime        string `json:"checkinTime"` // "HH:mm"，空则用默认
	PushPlusToken     string `json:"pushplusToken"`
	TotalCredits       int64  `json:"totalCredits"`
	LastCheckinAt      int64  `json:"lastCheckinAt"`
	LastCheckinResult  string `json:"lastCheckinResult"`
	LastCheckinMessage string `json:"lastCheckinMessage"`
	LastEarned         int64  `json:"lastEarned"`
	CreatedAt          int64  `json:"createdAt"`
}

// ToAuth 转成 upstream 使用的运行时凭证。
func (a Account) ToAuth() *auth.Auth {
	region := a.Region
	if region == "" {
		region = "CN"
	}
	return &auth.Auth{
		AccessToken:   a.AccessToken,
		RefreshToken:  a.RefreshToken,
		ExpiresAt:     a.ExpiresAt,
		Domain:        a.Domain,
		ApiHost:       a.ApiHost,
		MachineID:     a.MachineID,
		DeviceID:      a.DeviceID,
		PrivateKeyPEM: a.PrivateKeyPEM,
		PublicKeyPEM:  a.PublicKeyPEM,
		UID:           a.UID,
		Nickname:      a.Nickname,
		Region:        region,
	}
}

// PublicAccount 脱敏账号（返回前端，不含 token）。
type PublicAccount struct {
	Account
	HasRefreshToken bool `json:"hasRefreshToken"`
}

// CheckinLog 签到日志。
type CheckinLog struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName"`
	Time        int64  `json:"time"`
	Result      string `json:"result"` // "success" | "failed"
	Message     string `json:"message"`
	Earned      int64  `json:"earned"`
	Remain      int64  `json:"remain"`
}

// Settings 全局设置。
type Settings struct {
	DefaultCheckinTime   string `json:"defaultCheckinTime"`   // "HH:mm"
	DefaultPushPlusToken string `json:"defaultPushplusToken"` // 默认推送 token
	AutoCheckin          bool   `json:"autoCheckin"`          // 是否启用定时签到
}

type storeData struct {
	Accounts []Account   `json:"accounts"`
	Logs     []CheckinLog `json:"logs"`
	Settings Settings    `json:"settings"`
}

// Store 线程安全的持久化存储。
type Store struct {
	mu   sync.Mutex
	path string
	d    *storeData
}

// New 加载或创建存储。
func New(path string) (*Store, error) {
	s := &Store{path: path, d: &storeData{}}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &s.d); err != nil {
			return nil, fmt.Errorf("parse store: %w", err)
		}
	}
	if s.d.Settings.DefaultCheckinTime == "" {
		s.d.Settings.DefaultCheckinTime = "08:00"
		s.d.Settings.AutoCheckin = true // 首次默认开启定时签到
	}
	return s, nil
}

func (s *Store) save() error {
	if dir := filepath.Dir(s.path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	raw, err := json.MarshalIndent(s.d, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func nowMs() int64 { return time.Now().UnixMilli() }

// ListAccounts 返回全部账号（脱敏）。
func (s *Store) ListAccounts() []PublicAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PublicAccount, 0, len(s.d.Accounts))
	for _, a := range s.d.Accounts {
		pa := PublicAccount{Account: a, HasRefreshToken: a.RefreshToken != ""}
		pa.AccessToken = ""
		pa.RefreshToken = ""
		out = append(out, pa)
	}
	return out
}

// GetAccount 取原始账号（含凭证，内部用）。
func (s *Store) GetAccount(id string) (Account, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, a := range s.d.Accounts {
		if a.ID == id {
			return a, true
		}
	}
	return Account{}, false
}

// upsertByUID 按 uid upsert：已存在则更新凭证并保留用户配置，否则新增。
func (s *Store) upsertByUID(acc Account) Account {
	for i, a := range s.d.Accounts {
		if acc.UID != "" && a.UID == acc.UID {
			// 保留用户配置字段
			acc.ID = a.ID
			acc.Remark = orVal(acc.Remark, a.Remark)
			acc.CheckinTime = orVal(acc.CheckinTime, a.CheckinTime)
			acc.PushPlusToken = orVal(acc.PushPlusToken, a.PushPlusToken)
			acc.TotalCredits = a.TotalCredits
			acc.LastCheckinAt = a.LastCheckinAt
			acc.LastCheckinResult = a.LastCheckinResult
			acc.LastCheckinMessage = a.LastCheckinMessage
			acc.LastEarned = a.LastEarned
			acc.CreatedAt = a.CreatedAt
			acc.Enabled = a.Enabled // 已有账号保留原启用状态
			s.d.Accounts[i] = acc
			return acc
		}
	}
	if acc.ID == "" {
		acc.ID = newID()
	}
	if acc.CreatedAt == 0 {
		acc.CreatedAt = nowMs()
	}
	acc.Enabled = true // 新导入账号默认启用
	if acc.Region == "" {
		acc.Region = "CN"
	}
	if acc.ApiHost == "" {
		acc.ApiHost = "https://api.trae.com.cn"
	}
	s.d.Accounts = append(s.d.Accounts, acc)
	return acc
}

func orVal(pref, fallback string) string {
	if pref != "" {
		return pref
	}
	return fallback
}

// UpsertAccount 导入/更新账号（凭证来源），返回最终账号（脱敏）。
func (s *Store) UpsertAccount(acc Account) (PublicAccount, error) {
	s.mu.Lock()
	saved := s.upsertByUID(acc)
	err := s.save()
	s.mu.Unlock()
	if err != nil {
		return PublicAccount{}, err
	}
	pa := PublicAccount{Account: saved, HasRefreshToken: saved.RefreshToken != ""}
	pa.AccessToken = ""
	pa.RefreshToken = ""
	return pa, nil
}

// applyPatch 按 patch map 部分更新账号可编辑字段。
func applyPatch(a *Account, p map[string]any) {
	for k, v := range p {
		switch k {
		case "remark":
			if s, ok := v.(string); ok {
				a.Remark = s
			}
		case "checkinTime":
			if s, ok := v.(string); ok {
				a.CheckinTime = s
			}
		case "pushplusToken":
			if s, ok := v.(string); ok {
				a.PushPlusToken = s
			}
		case "enabled":
			if b, ok := v.(bool); ok {
				a.Enabled = b
			}
		case "nickname":
			if s, ok := v.(string); ok {
				a.Nickname = s
			}
		}
	}
}

// UpdateAccount 部分更新账号（用户配置），返回脱敏账号。
func (s *Store) UpdateAccount(id string, patch map[string]any) (PublicAccount, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.d.Accounts {
		if a.ID == id {
			applyPatch(&s.d.Accounts[i], patch)
			_ = s.save()
			pa := PublicAccount{Account: s.d.Accounts[i], HasRefreshToken: s.d.Accounts[i].RefreshToken != ""}
			pa.AccessToken = ""
			pa.RefreshToken = ""
			return pa, true
		}
	}
	return PublicAccount{}, false
}

// UpdateCredentials 签到后回写凭证与状态（内部用）。
func (s *Store) UpdateCredentials(id string, acc Account) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.d.Accounts {
		if a.ID == id {
			if acc.AccessToken != "" {
				s.d.Accounts[i].AccessToken = acc.AccessToken
			}
			if acc.RefreshToken != "" {
				s.d.Accounts[i].RefreshToken = acc.RefreshToken
			}
			if acc.ExpiresAt != 0 {
				s.d.Accounts[i].ExpiresAt = acc.ExpiresAt
			}
			if acc.TotalCredits != 0 {
				s.d.Accounts[i].TotalCredits = acc.TotalCredits
			}
			s.d.Accounts[i].LastCheckinAt = acc.LastCheckinAt
			s.d.Accounts[i].LastCheckinResult = acc.LastCheckinResult
			s.d.Accounts[i].LastCheckinMessage = acc.LastCheckinMessage
			s.d.Accounts[i].LastEarned = acc.LastEarned
			_ = s.save()
			return true
		}
	}
	return false
}

// DeleteAccount 删除账号。
func (s *Store) DeleteAccount(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, a := range s.d.Accounts {
		if a.ID == id {
			s.d.Accounts = append(s.d.Accounts[:i], s.d.Accounts[i+1:]...)
			_ = s.save()
			return true
		}
	}
	return false
}

// ExportAccounts 返回全部账号完整数据（含凭证，用于配置导出迁移）。
func (s *Store) ExportAccounts() []Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Account, len(s.d.Accounts))
	copy(out, s.d.Accounts)
	return out
}

// ImportAccounts 批量导入账号（按 UID 匹配：已存在则整体替换，否则新增）。
// 返回（新增数， 更新数）。
func (s *Store) ImportAccounts(accs []Account) (int, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	added, updated := 0, 0
	for _, acc := range accs {
		if acc.UID == "" {
			continue
		}
		replaced := false
		for i, a := range s.d.Accounts {
			if a.UID == acc.UID {
				if acc.ID == "" {
					acc.ID = a.ID
				}
				s.d.Accounts[i] = acc
				replaced = true
				updated++
				break
			}
		}
		if !replaced {
			if acc.ID == "" {
				acc.ID = newID()
			}
			if acc.CreatedAt == 0 {
				acc.CreatedAt = nowMs()
			}
			if acc.Region == "" {
				acc.Region = "CN"
			}
			if acc.ApiHost == "" {
				acc.ApiHost = "https://api.trae.com.cn"
			}
			s.d.Accounts = append(s.d.Accounts, acc)
			added++
		}
	}
	return added, updated, s.save()
}

// AddLog 追加签到日志，超出上限截断。
func (s *Store) AddLog(l CheckinLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l.ID == "" {
		l.ID = newID()
	}
	if l.Time == 0 {
		l.Time = nowMs()
	}
	s.d.Logs = append(s.d.Logs, l)
	if len(s.d.Logs) > maxLogs {
		s.d.Logs = s.d.Logs[len(s.d.Logs)-maxLogs:]
	}
	_ = s.save()
}

// ListLogs 返回最近 limit 条日志（倒序）。
func (s *Store) ListLogs(limit int) []CheckinLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	logs := s.d.Logs
	if limit <= 0 || limit > len(logs) {
		limit = len(logs)
	}
	out := make([]CheckinLog, limit)
	for i := 0; i < limit; i++ {
		out[i] = logs[len(logs)-1-i]
	}
	return out
}

// ClearLogs 清空日志。
func (s *Store) ClearLogs() {
	s.mu.Lock()
	s.d.Logs = nil
	_ = s.save()
	s.mu.Unlock()
}

// GetSettings 返回设置。
func (s *Store) GetSettings() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.d.Settings
}

// SaveSettings 更新设置（partial，非空字段才覆盖）。
func (s *Store) SaveSettings(p Settings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.DefaultCheckinTime != "" {
		s.d.Settings.DefaultCheckinTime = p.DefaultCheckinTime
	}
	if p.DefaultPushPlusToken != "" {
		s.d.Settings.DefaultPushPlusToken = p.DefaultPushPlusToken
	}
	// bool 无法区分"未提供"与"false"，这里始终覆盖
	s.d.Settings.AutoCheckin = p.AutoCheckin
	_ = s.save()
	return s.d.Settings
}
