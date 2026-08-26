// Package api HTTP API：账号管理、签到、日志、设置、OAuth 登录。
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"trae-signin-web/internal/auth"
	"trae-signin-web/internal/checkin"
	"trae-signin-web/internal/scheduler"
	"trae-signin-web/internal/store"
	"trae-signin-web/internal/upstream"
	"trae-signin-web/internal/webauth"
)

// Deps API 依赖。
type Deps struct {
	Store *store.Store
	Up    *upstream.Client
	Svc   *checkin.Service
	Sched *scheduler.Scheduler
	Auth  *webauth.Manager
}

// NewHandler 构建路由。
func NewHandler(d Deps) http.Handler {
	mux := http.NewServeMux()

	// 认证相关（无需登录）
	mux.HandleFunc("GET /api/auth/status", d.authStatus)
	mux.HandleFunc("POST /api/auth/setup", d.authSetup)
	mux.HandleFunc("POST /api/auth/login", d.authLogin)
	mux.HandleFunc("POST /api/auth/logout", d.authLogout)
	mux.HandleFunc("POST /api/auth/change", d.authChange)

	// 业务接口（登录后可用）
	mux.HandleFunc("GET /api/accounts", d.requireAuth(d.listAccounts))
	mux.HandleFunc("GET /api/accounts/export", d.requireAuth(d.exportAccounts))
	mux.HandleFunc("POST /api/accounts/import", d.requireAuth(d.importAccounts))
	mux.HandleFunc("PATCH /api/accounts/{id}", d.requireAuth(d.updateAccount))
	mux.HandleFunc("DELETE /api/accounts/{id}", d.requireAuth(d.deleteAccount))
	mux.HandleFunc("POST /api/accounts/{id}/checkin", d.requireAuth(d.checkinOne))
	mux.HandleFunc("POST /api/accounts/checkin-all", d.requireAuth(d.checkinAll))
	mux.HandleFunc("GET /api/accounts/{id}/points", d.requireAuth(d.getPoints))
	mux.HandleFunc("GET /api/logs", d.requireAuth(d.listLogs))
	mux.HandleFunc("DELETE /api/logs", d.requireAuth(d.clearLogs))
	mux.HandleFunc("GET /api/settings", d.requireAuth(d.getSettings))
	mux.HandleFunc("POST /api/settings", d.requireAuth(d.saveSettings))
	mux.HandleFunc("POST /api/login/url", d.requireAuth(d.loginURL))
	mux.HandleFunc("POST /api/login/callback", d.requireAuth(d.loginCallback))

	return cors(mux)
}

// ===== 中间件 / 工具 =====

// requireAuth 会话校验中间件：未登录返回 401。
func (d *Deps) requireAuth(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !d.Auth.Valid(webauth.TokenFromRequest(r)) {
			jsonError(w, http.StatusUnauthorized, "未登录或会话已过期")
			return
		}
		next(w, r)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ===== 认证 =====

// authStatus 查询初始化/登录状态（前端据此渲染设置密码页、登录页或主界面）。
func (d *Deps) authStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"initialized": d.Auth.Initialized(),
		"loggedIn":    d.Auth.Valid(webauth.TokenFromRequest(r)),
		"username":    d.Auth.Username(),
	})
}

func (d *Deps) authSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := d.Auth.Setup(req.Username, req.Password); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	// 设置成功直接进入已登录状态
	token, err := d.Auth.Login(req.Username, req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	webauth.SetSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (d *Deps) authLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, err := d.Auth.Login(req.Username, req.Password)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, err.Error())
		return
	}
	webauth.SetSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (d *Deps) authLogout(w http.ResponseWriter, r *http.Request) {
	d.Auth.Logout(webauth.TokenFromRequest(r))
	webauth.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// authChange 修改登录账号密码（需已登录 + 旧密码验证）。
func (d *Deps) authChange(w http.ResponseWriter, r *http.Request) {
	if !d.Auth.Valid(webauth.TokenFromRequest(r)) {
		jsonError(w, http.StatusUnauthorized, "未登录或会话已过期")
		return
	}
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewUsername string `json:"newUsername"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := d.Auth.Change(req.OldPassword, req.NewUsername, req.NewPassword); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== 账号 =====

func (d *Deps) listAccounts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, d.Store.ListAccounts())
}

// exportAccounts 导出全部账号配置（含凭证，附件下载）。
func (d *Deps) exportAccounts(w http.ResponseWriter, r *http.Request) {
	payload := struct {
		Version   int             `json:"version"`
		ExportedAt int64          `json:"exportedAt"`
		Accounts  []store.Account `json:"accounts"`
	}{
		Version:    1,
		ExportedAt: time.Now().Unix(),
		Accounts:   d.Store.ExportAccounts(),
	}
	w.Header().Set("Content-Disposition",
		`attachment; filename="trae-signin-accounts-`+time.Now().Format("20060102")+`.json"`)
	writeJSON(w, http.StatusOK, payload)
}

// importAccounts 导入账号配置：接受 {"accounts":[...]} 或纯数组 [...] 两种格式。
func (d *Deps) importAccounts(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	var accs []store.Account
	var wrapper struct {
		Accounts []store.Account `json:"accounts"`
	}
	if json.Unmarshal(body, &wrapper) == nil && wrapper.Accounts != nil {
		accs = wrapper.Accounts
	} else if err := json.Unmarshal(body, &accs); err != nil {
		jsonError(w, http.StatusBadRequest, "无法解析导入文件（期望 JSON 数组或 {\"accounts\":[...]}）: "+err.Error())
		return
	}
	if len(accs) == 0 {
		jsonError(w, http.StatusBadRequest, "文件中没有账号")
		return
	}
	added, updated, err := d.Store.ImportAccounts(accs)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"added": added, "updated": updated})
}

func (d *Deps) updateAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	pa, ok := d.Store.UpdateAccount(id, patch)
	if !ok {
		jsonError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, pa)
}

func (d *Deps) deleteAccount(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !d.Store.DeleteAccount(id) {
		jsonError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (d *Deps) checkinOne(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := d.Svc.CheckinAccount(id, true)
	if err != nil {
		jsonError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (d *Deps) checkinAll(w http.ResponseWriter, r *http.Request) {
	results := d.Svc.CheckinAll()
	writeJSON(w, http.StatusOK, results)
}

func (d *Deps) getPoints(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	acc, ok := d.Store.GetAccount(id)
	if !ok {
		jsonError(w, http.StatusNotFound, "account not found")
		return
	}
	a := acc.ToAuth()
	if a.RefreshToken != "" && a.NeedsRefresh(2*3600) {
		_ = d.Up.RefreshToken(a)
	}
	remain, err := d.Up.UserEntUsage(a)
	if err != nil {
		jsonError(w, http.StatusBadGateway, "查询积分失败: "+err.Error())
		return
	}
	// 回写（含可能的刷新后凭证）
	back := acc
	back.AccessToken = a.AccessToken
	back.RefreshToken = a.RefreshToken
	back.ExpiresAt = a.ExpiresAt
	if remain > 0 {
		back.TotalCredits = remain
	}
	d.Store.UpdateCredentials(id, back)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "totalPoints": remain})
}

// ===== 日志 =====

func (d *Deps) listLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	writeJSON(w, http.StatusOK, d.Store.ListLogs(limit))
}

func (d *Deps) clearLogs(w http.ResponseWriter, r *http.Request) {
	d.Store.ClearLogs()
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// ===== 设置 =====

func (d *Deps) getSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, d.Store.GetSettings())
}

func (d *Deps) saveSettings(w http.ResponseWriter, r *http.Request) {
	var p store.Settings
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d.Store.SaveSettings(p))
}

// ===== OAuth 登录 =====

const (
	oAuthLoginBase = "https://www.trae.cn/authorization"
	clientID       = upstream.ClientID
)

type loginURLReq struct {
	Remark       string `json:"remark"`
	CheckinTime  string `json:"checkinTime"`
	PushPlusToken string `json:"pushplusToken"`
}

type loginURLResp struct {
	LoginURL      string `json:"loginUrl"`
	MachineID     string `json:"machineId"`
	DeviceID      string `json:"deviceId"`
	PrivateKeyPEM string `json:"privateKeyPem"`
	PublicKeyPEM  string `json:"publicKeyPem"`
}

func (d *Deps) loginURL(w http.ResponseWriter, r *http.Request) {
	machineID := randHex(16)
	// 设备 ID 必须是 16 位纯数字（对齐客户端 AHA 设备服务注册的格式）。
	// hex 格式会被签到接口风控拒绝（9074）；且配额按设备每天一次，
	// 每个账号独立设备 ID，互不占用。已实测随机数字 ID 可正常签到。
	deviceID := upstream.NewDeviceID()
	// 生成设备签名密钥对（ECDSA P-256），与客户端 C_e() 等价。
	// 登录后 ExchangeToken 用此私钥签名、公钥随 DeviceInfo 提交，
	// 服务端校验配对即可，无需单独注册设备。
	privPEM, pubPEM, err := upstream.GenerateDeviceKeyPair()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "生成设备密钥失败: "+err.Error())
		return
	}
	params := url.Values{
		"login_version":    {"1"},
		"auth_from":        {"trae"},
		"login_channel":    {"native_ide"},
		"plugin_version":   {"2.3.62834"},
		"auth_type":        {"local"},
		"client_id":        {clientID},
		"redirect":         {"0"},
		"login_trace_id":   {randHex(8)},
		"auth_callback_url": {"http://127.0.0.1:18080/authorize"},
		"machine_id":       {machineID},
		"device_id":        {deviceID},
		"x_device_id":      {deviceID},
		"x_machine_id":    {machineID},
		"x_device_brand":   {"PC"},
		"x_device_type":    {"PC"},
		"x_os_version":     {"1.0"},
		"x_app_version":    {upstream.IdeVersion},
		"x_app_type":       {"stable"},
	}
	writeJSON(w, http.StatusOK, loginURLResp{
		LoginURL:      oAuthLoginBase + "?" + params.Encode(),
		MachineID:     machineID,
		DeviceID:      deviceID,
		PrivateKeyPEM: privPEM,
		PublicKeyPEM:  pubPEM,
	})
}

type loginCallbackReq struct {
	CallbackURL   string `json:"callbackUrl"`
	MachineID     string `json:"machineId"`
	DeviceID      string `json:"deviceId"`
	PrivateKeyPEM string `json:"privateKeyPem"`
	PublicKeyPEM  string `json:"publicKeyPem"`
	Remark        string `json:"remark"`
	CheckinTime   string `json:"checkinTime"`
	PushPlusToken string `json:"pushplusToken"`
}

func (d *Deps) loginCallback(w http.ResponseWriter, r *http.Request) {
	var req loginCallbackReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.CallbackURL == "" {
		jsonError(w, http.StatusBadRequest, "缺少 callbackUrl")
		return
	}
	u, err := url.Parse(req.CallbackURL)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "回调链接解析失败: "+err.Error())
		return
	}
	q := u.Query()
	refreshToken := q.Get("refreshToken")
	userInfo := parseJSONParam(q.Get("userInfo"))
	userJwt := parseJSONParam(q.Get("userJwt"))

	uid := strVal(userInfo, "UserID")
	nickname := strVal(userInfo, "ScreenName")
	jwtToken := strVal(userJwt, "Token")
	jwtRefresh := strVal(userJwt, "RefreshToken")
	if refreshToken == "" {
		refreshToken = jwtRefresh
	}

	var (
		accessToken  string
		expiresAt     int64
		newRefresh   string
		needReLogin  bool
	)
	if refreshToken != "" {
		tmp := &auth.Auth{
			RefreshToken:  refreshToken,
			ApiHost:       upstream.OAuthHost,
			DeviceID:      req.DeviceID,
			MachineID:     req.MachineID,
			PrivateKeyPEM: req.PrivateKeyPEM,
			PublicKeyPEM:  req.PublicKeyPEM,
		}
		if err := d.Up.RefreshToken(tmp); err != nil {
			jsonError(w, http.StatusBadGateway, "ExchangeToken 失败: "+err.Error())
			return
		}
		accessToken = tmp.AccessToken
		expiresAt = tmp.ExpiresAt
		newRefresh = tmp.RefreshToken
	} else if jwtToken != "" {
		accessToken = jwtToken
		expiresAt = parseExpireAt(strVal(userJwt, "TokenExpireAt"))
	} else {
		needReLogin = true
	}
	if needReLogin {
		jsonError(w, http.StatusBadRequest, "回调链接缺少 refreshToken，请重新登录")
		return
	}

	// 补全 uid/nickname
	if uid == "" || nickname == "" {
		if u2, n2, err := d.Up.GetUserInfo(accessToken); err == nil {
			if uid == "" {
				uid = u2
			}
			if nickname == "" {
				nickname = n2
			}
		}
	}

	acc := store.Account{
		UID: uid, Nickname: nickname,
		AccessToken: accessToken, RefreshToken: newRefresh, ExpiresAt: expiresAt,
		ApiHost: upstream.OAuthHost, Domain: "trae.cn",
		MachineID: req.MachineID, DeviceID: req.DeviceID, Region: "CN",
		PrivateKeyPEM: req.PrivateKeyPEM, PublicKeyPEM: req.PublicKeyPEM,
		Remark: req.Remark, CheckinTime: req.CheckinTime, PushPlusToken: req.PushPlusToken,
	}
	pa, err := d.Store.UpsertAccount(acc)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, pa)
}

// parseJSONParam 把回调里的 userInfo/userJwt 参数解析为 map。参数可能被 URL 编码一次或两次。
func parseJSONParam(raw string) map[string]any {
	if raw == "" {
		return nil
	}
	for _, v := range []string{raw, unescapeOnce(raw)} {
		var m map[string]any
		if json.Unmarshal([]byte(v), &m) == nil {
			return m
		}
	}
	return nil
}

func unescapeOnce(s string) string {
	if d, err := url.QueryUnescape(s); err == nil {
		return d
	}
	return s
}

func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case string:
			return t
		case float64:
			return fmt.Sprintf("%v", t)
		}
	}
	return ""
}

func parseExpireAt(s string) int64 {
	if s == "" {
		return 0
	}
	var n int64
	fmt.Sscanf(s, "%d", &n)
	if n > 1e12 {
		n /= 1000
	}
	return n
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

