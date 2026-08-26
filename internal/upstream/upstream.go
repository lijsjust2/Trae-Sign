// Package upstream 封装 TRAE SOLO 上游 API：签到、积分查询、Token 刷新、
// 用户信息查询、PushPlus 推送。移植并扩展自 trae-signin-main。
package upstream

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"trae-signin-web/internal/auth"
)

const (
	UgHost     = "https://api.trae.cn"
	OAuthHost  = "https://api.trae.cn"
	ClientID   = "ono9krqynydwx5" // TRAE(IDE)官方 OAuth 应用，提取自客户端 main.js rv() 默认值
	IdeVersion = "3.3.80"

	// EpExchange 官方刷新端点。与 en1oxy7wnw8j9n 用的 /cloudide/... 不同，
	// 本端点要求 ClientID + DeviceInfo + DeviceProof(ECDSA 设备签名)，
	// 被服务端识别为正规客户端，签到不受 9074 限制。
	EpExchange      = "/trae/api/v3/oauth/ExchangeToken"
	EpCheckinStatus = "/trae/api/v2/ug/checkin_credits/status"
	EpCheckinClaim  = "/trae/api/v2/ug/checkin_credits/claim"
	EpEntUsage      = "/trae/api/v2/pay/ide_user_ent_usage"
	EpGetUserInfo   = "/cloudide/api/v3/trae/GetUserInfo"

	PushPlusURL = "https://www.pushplus.plus/send"
)

// clientUA 对齐 Trae 客户端（VSCode 内核）。用 "Trae/0.1.43" 会被服务端按非客户端
// 来源计配额/风控，触发 9074。对齐可正常签到的开源实现（trae-mate / ql 版）。
var clientUA = "VSCode 1.107.1 (TRAE SOLO CN)"

// Client 上游 API 客户端。
type Client struct {
	HTTP *http.Client
}

// New 创建客户端，复用连接池。
func New() *Client {
	tr := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{HTTP: &http.Client{Timeout: 60 * time.Second, Transport: tr}}
}

// doJSON 发送请求并返回响应体。HTTP>=400 返回错误（错误信息含状态码，便于上层判断 401）。
func (c *Client) doJSON(req *http.Request) (json.RawMessage, int, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}
	return raw, resp.StatusCode, nil
}

// deviceInfo 设备信息（对齐客户端 this.j() 输出）。服务端校验 DevicePublicKey 与签名私钥配对。
type deviceInfo struct {
	DeviceID        string `json:"DeviceID"`
	MachineID       string `json:"MachineID"`
	PlatformCode    string `json:"PlatformCode"` // IDE 模式 = "IDE_PC"
	DeviceType      string `json:"DeviceType"`
	DeviceName      string `json:"DeviceName"`
	DeviceModel     string `json:"DeviceModel"`
	ClientVersion   string `json:"ClientVersion"`
	DevicePublicKey string `json:"DevicePublicKey"`
	DeviceBrand     string `json:"DeviceBrand"`
	DeviceCPU       string `json:"DeviceCPU"`
	OSInfo          string `json:"OSInfo"`
	OSVersion       string `json:"OSVersion"`
}

type deviceProof struct {
	Signature string `json:"Signature"`
	Timestamp int64  `json:"Timestamp"`
	Nonce     string `json:"Nonce"`
}

// ecdsaSignSHA256 用 PKCS8 PEM 私钥对 payload 做 SHA256 ECDSA 签名，返回 base64(DER)。
// 等价于客户端 Node crypto.sign("sha256", data, privateKey)（E_e 函数）。
func ecdsaSignSHA256(privPEM string, payload []byte) (string, error) {
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return "", fmt.Errorf("bad private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return "", fmt.Errorf("key is not a crypto.Signer")
	}
	h := sha256.Sum256(payload)
	sigDER, err := signer.Sign(rand.Reader, h[:], crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sigDER), nil
}

// GenerateDeviceKeyPair 生成 ECDSA P-256 设备签名密钥对，返回 PKCS8 私钥 PEM
// 与 SPKI 公钥 PEM。等价于客户端 Node crypto.generateKeyPairSync("ec",{namedCurve:"P-256"})。
// 服务端仅在 ExchangeToken 时校验公钥与签名配对，无需单独注册设备。
func GenerateDeviceKeyPair() (privateKeyPEM, publicKeyPEM string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ec key: %w", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("marshal pkcs8: %w", err)
	}
	privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal spki: %w", err)
	}
	publicKeyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privateKeyPEM, publicKeyPEM, nil
}

// NewDeviceID 生成 16 位随机纯数字设备 ID（对齐客户端 AHA 设备服务注册的格式）。
// hex 格式会被签到接口风控拒绝（9074）；实测随机数字 ID 部分可正常签到
// （9074 按设备 ID 分桶，换 ID 即重新抽签），DoCheckin 遇 9074 自动轮换。
func NewDeviceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = '0' + b[i]%10
	}
	return string(b)
}

// RefreshToken 通过官方 ExchangeToken 强制刷新 access token。
// 与被限签到的 en1oxy7wnw8j9n(旧端点 ClientSecret="-" 无设备签名)不同，
// 本端点要求 ClientID + DeviceInfo + DeviceProof(ECDSA 设备签名)，
// 被服务端识别为正规客户端 → 签到不受 9074 限制（已验证返回 9095 已签到）。
// 材料(PrivateKeyPEM/PublicKeyPEM/MachineID/DeviceID) 由导入工具从 TRAE 客户端实例写入。
func (c *Client) RefreshToken(a *auth.Auth) error {
	if strings.TrimSpace(a.RefreshToken) == "" {
		return fmt.Errorf("no refreshToken")
	}
	if strings.TrimSpace(a.PrivateKeyPEM) == "" || strings.TrimSpace(a.PublicKeyPEM) == "" {
		return fmt.Errorf("NO_DEVICE_KEY: 账号缺少设备签名密钥，请从 TRAE 客户端实例导入")
	}
	host := a.ApiHost
	if host == "" {
		host = OAuthHost
	}
	ts := time.Now().Unix()
	nonce := hex.EncodeToString(randNonce(16))
	// 签名串：POST\n/path\nClientID\nrefreshToken\ntimestamp\nnonce
	payload := strings.Join([]string{"POST", EpExchange, ClientID, a.RefreshToken, fmt.Sprintf("%d", ts), nonce}, "\n")
	sig, err := ecdsaSignSHA256(a.PrivateKeyPEM, []byte(payload))
	if err != nil {
		return err
	}
	body := map[string]any{
		"ClientID":     ClientID,
		"ClientSecret": "",
		"RefreshToken": a.RefreshToken,
		"DeviceInfo": deviceInfo{
			DeviceID:        a.DeviceID,
			MachineID:       a.MachineID,
			PlatformCode:    "IDE_PC",
			DeviceType:      "PC",
			DeviceName:      "trae-signin-web",
			ClientVersion:   IdeVersion,
			DevicePublicKey: a.PublicKeyPEM,
			OSInfo:          "Windows",
		},
		"DeviceProof": deviceProof{Signature: sig, Timestamp: ts, Nonce: nonce},
		"IDEVersion":  IdeVersion,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, host+EpExchange, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", clientUA)

	data, _, err := c.doJSON(req)
	if err != nil {
		return err
	}
	var resp struct {
		Result struct {
			Token               string `json:"Token"`
			TokenExpireAt       int64  `json:"TokenExpireAt"`
			TokenExpireDuration int64  `json:"TokenExpireDuration"`
			RefreshToken        string `json:"RefreshToken"`
		} `json:"Result"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("exchange parse: %w", err)
	}
	if resp.Result.Token == "" {
		return fmt.Errorf("refresh_failed: no token — re-login required")
	}
	a.AccessToken = resp.Result.Token
	if resp.Result.RefreshToken != "" {
		a.RefreshToken = resp.Result.RefreshToken
	}
	if resp.Result.TokenExpireAt > 0 {
		exp := resp.Result.TokenExpireAt
		if exp > 1e12 { // 毫秒转秒
			exp /= 1000
		}
		a.ExpiresAt = exp
	} else if resp.Result.TokenExpireDuration > 0 {
		a.ExpiresAt = time.Now().Add(time.Duration(resp.Result.TokenExpireDuration) * time.Second).Unix()
	}
	return nil
}

func randNonce(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

// CheckinStatus 查询签到状态。
func (c *Client) CheckinStatus(a *auth.Auth) (checkedIn bool, credits int64, enable bool, err error) {
	req, err := http.NewRequest(http.MethodPost, UgHost+EpCheckinStatus, bytes.NewReader([]byte("{}")))
	if err != nil {
		return false, 0, false, err
	}
	ugHeaders(req, a)
	data, status, err := c.doJSON(req)
	if err != nil {
		return false, 0, false, fmt.Errorf("%w (status=%d)", err, status)
	}
	var resp struct {
		CheckedIn bool    `json:"checked_in"`
		Credits   float64 `json:"credits"`
		Enable    bool    `json:"enable"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return false, 0, false, fmt.Errorf("checkin status parse: %w", err)
	}
	return resp.CheckedIn, int64(resp.Credits), resp.Enable, nil
}

// CheckinClaim 执行签到，返回本次获得积分（从响应 data.credits/points 等字段取）。
func (c *Client) CheckinClaim(a *auth.Auth) (earned int64, err error) {
	req, err := http.NewRequest(http.MethodPost, UgHost+EpCheckinClaim, bytes.NewReader([]byte("{}")))
	if err != nil {
		return 0, err
	}
	ugHeaders(req, a)
	data, status, err := c.doJSON(req)
	if err != nil {
		return 0, fmt.Errorf("%w (status=%d)", err, status)
	}
	// 解析业务码与积分
	var resp struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
		Msg     string `json:"msg"`
		Data    struct {
			Credits float64 `json:"credits"`
			Points  float64 `json:"points"`
			Credit  float64 `json:"credit"`
			Amount  float64 `json:"amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, fmt.Errorf("claim parse: %w", err)
	}
	bizMsg := resp.Message
	if bizMsg == "" {
		bizMsg = resp.Msg
	}
	// 业务码（float64 是 JSON number 默认解码类型）
	var bizCode int64
	if v, ok := resp.Code.(float64); ok {
		bizCode = int64(v)
	}
	// 已签到（9095 当前设备今日已经签到）视为业务提示而非错误
	if strings.Contains(bizMsg, "已签到") || strings.Contains(bizMsg, "已经签到") || bizCode == 9095 {
		return 0, fmt.Errorf("ALREADY: %s", bizMsg)
	}
	// 频控（9074 当前参与用户太多）单独标记，便于调度器冷却后重试
	if strings.Contains(bizMsg, "参与用户太多") || strings.Contains(bizMsg, "稍后再试") || bizCode == 9074 {
		return 0, fmt.Errorf("RATE_LIMIT: code=%d %s", bizCode, bizMsg)
	}
	// 其他非成功业务码一律判失败（防止兜底 200 把失败当成功）
	if bizCode != 0 && bizCode != 200 {
		return 0, fmt.Errorf("BIZ_FAIL: code=%d msg=%s", bizCode, bizMsg)
	}
	// 成功则取积分
	if d := resp.Data; d.Credits > 0 {
		return int64(d.Credits), nil
	}
	if d := resp.Data; d.Points > 0 {
		return int64(d.Points), nil
	}
	if d := resp.Data; d.Credit > 0 {
		return int64(d.Credit), nil
	}
	if d := resp.Data; d.Amount > 0 {
		return int64(d.Amount), nil
	}
	// 取不到积分则默认 200（与 trae-mate 一致）
	return 200, nil
}

// UserEntUsage 查询积分余额（剩余可用）。
func (c *Client) UserEntUsage(a *auth.Auth) (int64, error) {
	req, err := http.NewRequest(http.MethodPost, UgHost+EpEntUsage, bytes.NewReader([]byte("{}")))
	if err != nil {
		return 0, err
	}
	ugHeaders(req, a)
	data, status, err := c.doJSON(req)
	if err != nil {
		return 0, fmt.Errorf("%w (status=%d)", err, status)
	}
	var resp struct {
		UserEntitlementPackList []struct {
			EntitlementBaseInfo struct {
				Quota struct {
					CreditsLimit float64 `json:"credits_limit"`
				} `json:"quota"`
			} `json:"entitlement_base_info"`
			Usage struct {
				CreditsAmount float64 `json:"credits_amount"`
			} `json:"usage"`
		} `json:"user_entitlement_pack_list"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, fmt.Errorf("ent usage parse: %w", err)
	}
	var total float64
	for _, p := range resp.UserEntitlementPackList {
		if r := p.EntitlementBaseInfo.Quota.CreditsLimit - p.Usage.CreditsAmount; r > 0 {
			total += r
		}
	}
	return int64(total), nil
}

// GetUserInfo 用 access token 查询用户信息（登录回调后补全 uid/nickname 用）。
func (c *Client) GetUserInfo(accessToken string) (uid, nickname string, err error) {
	body, _ := json.Marshal(map[string]any{"ReqSource": "IDE", "IDEVersion": IdeVersion})
	req, err := http.NewRequest(http.MethodPost, OAuthHost+EpGetUserInfo, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-cloudide-token", accessToken)
	req.Header.Set("User-Agent", clientUA)
	data, _, err := c.doJSON(req)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		Result struct {
			UserID      string `json:"UserID"`
			ScreenName  string `json:"ScreenName"`
		} `json:"Result"`
	}
	_ = json.Unmarshal(data, &resp)
	return resp.Result.UserID, resp.Result.ScreenName, nil
}

// CheckinResult 单次签到完整结果。
type CheckinResult struct {
	Status    string // "OK" / "ALREADY" / "FAIL"
	Earned    int64  // 本次获得积分
	Remain    int64  // 签到后剩余积分
	Detail    string
	Refreshed bool // 是否刷新过 token（调用方据此回写凭证）
}

// claimMaxAttempts 单次签到内的设备轮换尝试上限（含首次）。
// 9074 实测是按设备 ID 分桶的容量/风控：同一 ID 重试结果不变，
// 换新数字 ID 即重新"抽签"（命中率约 40%），6 次累计成功率 >95%。
const claimMaxAttempts = 6

// DoCheckin 完整签到流程：刷新 token → 预查 status → claim（9074/9095 轮换设备重试）→ 查余额。
// - 预查 status：积分发放按账号每天幂等（重复 claim 返回 success 但不加积分），
//   已签到直接返回 ALREADY，省 claim 请求且避免幂等重复被误报为"签到成功"
// - 9074/9095 均为设备级门禁（配额按设备计、跨账号），换新数字设备 ID 重新尝试
// 轮换会修改 a.DeviceID，调用方需回写持久化。401 时强制刷新一次重试。
func (c *Client) DoCheckin(a *auth.Auth) CheckinResult {
	// 1. 主动刷新（2h 缓冲）
	refreshed := false
	if a.RefreshToken != "" && a.NeedsRefresh(2*3600) {
		if err := c.RefreshToken(a); err != nil {
			return CheckinResult{Status: "FAIL", Detail: "refresh: " + err.Error()}
		}
		refreshed = true
	}

	// 2. 预查签到状态：今日已领取 → 直接 ALREADY（status 不受设备分桶限制）
	if checked, _, _, serr := c.CheckinStatus(a); serr == nil && checked {
		remain, _ := c.UserEntUsage(a)
		return CheckinResult{Status: "ALREADY", Remain: remain, Detail: "今日已签到", Refreshed: refreshed}
	}

	// 3. claim，9074/9095 时轮换设备 ID 重试
	for attempt := 1; ; attempt++ {
		earned, err := c.CheckinClaim(a)
		if err == nil {
			remain, _ := c.UserEntUsage(a)
			return CheckinResult{Status: "OK", Earned: earned, Remain: remain, Detail: "签到成功", Refreshed: refreshed}
		}
		msg := err.Error()
		if !strings.HasPrefix(msg, "ALREADY") && !strings.HasPrefix(msg, "RATE_LIMIT") {
			if is401(err) && a.RefreshToken != "" && !refreshed {
				if err2 := c.RefreshToken(a); err2 == nil {
					refreshed = true
					continue
				}
			}
			return CheckinResult{Status: "FAIL", Detail: "claim: " + msg, Refreshed: refreshed}
		}
		if attempt >= claimMaxAttempts {
			if strings.HasPrefix(msg, "RATE_LIMIT") {
				return CheckinResult{Status: "RATE_LIMIT", Detail: "今日签到配额已用尽（" + msg + "），明日自动恢复", Refreshed: refreshed}
			}
			return CheckinResult{Status: "FAIL", Detail: msg + "（设备配额被占用，轮换未成功）", Refreshed: refreshed}
		}
		// 换新数字设备 ID 重试（a.DeviceID 由调用方回写持久化）
		a.DeviceID = NewDeviceID()
		time.Sleep(1500 * time.Millisecond)
	}
}

// PushPlus 推送签到结果到 PushPlus。
func (c *Client) PushPlus(token, title, content string) error {
	if token == "" {
		return fmt.Errorf("no pushplus token")
	}
	body, _ := json.Marshal(map[string]any{
		"token":    token,
		"title":    title,
		"content":  content,
		"template": "txt",
	})
	req, err := http.NewRequest(http.MethodPost, PushPlusURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	data, _, err := c.doJSON(req)
	if err != nil {
		return err
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	_ = json.Unmarshal(data, &resp)
	if resp.Code != 200 {
		return fmt.Errorf("pushplus: %s", resp.Msg)
	}
	return nil
}

// ugHeaders 对齐 ql 版：只带最必要的几个头。
// 多余的伪装头（如 X-User-Region）没有收益，反而增加风控特征。
func ugHeaders(req *http.Request, a *auth.Auth) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", clientUA)
	req.Header.Set("Authorization", "Cloud-IDE-JWT "+a.AccessToken)
	if a.DeviceID != "" {
		req.Header.Set("x-device-id", a.DeviceID)
	}
}

func is401(err error) bool {
	return err != nil && strings.Contains(err.Error(), "401")
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
