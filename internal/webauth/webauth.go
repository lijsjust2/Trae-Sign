// Package webauth Web 管理界面的登录认证：
// 首次使用设置账号密码（auth.json），登录后签发会话 cookie。
// 密码加盐 + SHA-256 多轮迭代哈希存储，会话保存在内存（重启后需重新登录）。
package webauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	sessionTTL    = 7 * 24 * time.Hour // 会话有效期（滑动续期）
	hashRounds    = 10000              // 密码哈希迭代轮数
	CookieName    = "ts_session"
	minPasswordLen = 6
)

// authFile 持久化的凭证。
type authFile struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	Salt         string `json:"salt"`
}

// Manager 认证管理器。
type Manager struct {
	mu       sync.RWMutex
	path     string
	auth     *authFile
	sessions map[string]time.Time // token -> 过期时间
}

// New 创建认证管理器。path 为 auth.json 存储路径。
func New(path string) (*Manager, error) {
	m := &Manager{path: path, sessions: map[string]time.Time{}}
	raw, err := os.ReadFile(path)
	if err == nil {
		var a authFile
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, fmt.Errorf("parse auth: %w", err)
		}
		m.auth = &a
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return m, nil
}

// Initialized 是否已设置账号密码。
func (m *Manager) Initialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.auth != nil && m.auth.Username != ""
}

// Username 当前登录账号名。
func (m *Manager) Username() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.auth == nil {
		return ""
	}
	return m.auth.Username
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword 加盐 + 多轮 SHA-256。
func hashPassword(password, salt string) string {
	h := sha256.Sum256([]byte(salt + ":" + password))
	for i := 0; i < hashRounds; i++ {
		h = sha256.Sum256(h[:])
	}
	return hex.EncodeToString(h[:])
}

func (m *Manager) save() error {
	if dir := filepath.Dir(m.path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	raw, err := json.MarshalIndent(m.auth, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, raw, 0o600)
}

// Setup 首次设置账号密码（仅未初始化时允许，防止覆盖）。
func (m *Manager) Setup(username, password string) error {
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(password) < minPasswordLen {
		return fmt.Errorf("密码至少 %d 位", minPasswordLen)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.auth != nil && m.auth.Username != "" {
		return fmt.Errorf("账号密码已设置，请直接登录")
	}
	m.auth = &authFile{
		Username:     username,
		PasswordHash: hashPassword(password, username),
		Salt:         username, // 盐用用户名（每账号唯一）
	}
	return m.save()
}

// Login 校验账号密码，成功返回会话 token。
func (m *Manager) Login(username, password string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.auth == nil || m.auth.Username == "" {
		return "", fmt.Errorf("尚未设置账号密码")
	}
	if m.auth.Username != username || hashPassword(password, m.auth.Salt) != m.auth.PasswordHash {
		return "", fmt.Errorf("用户名或密码错误")
	}
	token := randHex(32)
	m.sessions[token] = time.Now().Add(sessionTTL)
	m.gcSessionsLocked()
	return token, nil
}

// Logout 注销会话。
func (m *Manager) Logout(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, token)
}

// gcSessionsLocked 清理过期会话（调用方需持有写锁）。
func (m *Manager) gcSessionsLocked() {
	now := time.Now()
	for t, exp := range m.sessions {
		if now.After(exp) {
			delete(m.sessions, t)
		}
	}
}

// Valid 会话是否有效（滑动续期）。
func (m *Manager) Valid(token string) bool {
	if token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	exp, ok := m.sessions[token]
	if !ok || time.Now().After(exp) {
		return false
	}
	m.sessions[token] = time.Now().Add(sessionTTL)
	return true
}

// Change 修改账号密码（需验证旧密码）。
func (m *Manager) Change(oldPassword, newUsername, newPassword string) error {
	if newUsername == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if len(newPassword) < minPasswordLen {
		return fmt.Errorf("密码至少 %d 位", minPasswordLen)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.auth == nil || hashPassword(oldPassword, m.auth.Salt) != m.auth.PasswordHash {
		return fmt.Errorf("旧密码错误")
	}
	m.auth = &authFile{
		Username:     newUsername,
		PasswordHash: hashPassword(newPassword, newUsername),
		Salt:         newUsername,
	}
	return m.save()
}

// TokenFromRequest 从请求中取会话 token。
func TokenFromRequest(r *http.Request) string {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// SetSessionCookie 登录成功后下发会话 cookie。
func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

// ClearSessionCookie 注销时清除会话 cookie。
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
