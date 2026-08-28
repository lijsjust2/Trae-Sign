package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"trae-signin-web/internal/store"
	"trae-signin-web/internal/upstream"
	"trae-signin-web/internal/webauth"
)

// 端到端：GET /api/accounts 的 JSON 必须包含 todayEarned / todayStatus 字段，
// 且值来自今日 success 日志的聚合，不能用 account.LastEarned。
func TestListAccountsAPI_IncludesTodayEarnedFromLogs(t *testing.T) {
	// 准备空的 store
	dataDir := t.TempDir()
	dataPath := dataDir + "/data.json"
	authPath := dataDir + "/auth.json"
	st, err := store.New(dataPath)
	if err != nil { t.Fatal(err) }

	// 插入账号（模拟 LastEarned 被 ALREADY 覆盖的场景）
	acc := store.Account{
		ID: "a1", UID: "u1", Nickname: "41015",
		AccessToken: "x", RefreshToken: "y",
		LastEarned: 0, LastCheckinResult: "already", // 被污染
	}
	_, err = st.UpsertAccount(acc)
	if err != nil { t.Fatal(err) }

	// 插入今日 success +200 日志 （AddLog 是私有？直接塞进 d.Logs）
	// 注意 Upsert 后再 Load 一次然后通过 store.TestingXXX 不可行，所以手动注入
	_ = os.WriteFile(dataPath, []byte(`{
  "accounts":[{"id":"a1","uid":"u1","nickname":"41015","lastEarned":0,"lastCheckinResult":"already","accessToken":"x","refreshToken":"y"}],
  "logs":[{"id":"l1","accountId":"a1","accountName":"41015","time":`+strconv.FormatInt(time.Now().UnixMilli(),10)+`,"result":"success","message":"ok","earned":200,"remain":2973}],
  "settings":{"defaultCheckinTime":"08:00","defaultPushplusToken":"","autoCheckin":true}
}`), 0644)
	st2, err := store.New(dataPath)
	if err != nil { t.Fatal(err) }

	// 鉴权：Setup + Login 拿 token
	wa, err := webauth.New(authPath)
	if err != nil { t.Fatal(err) }
	if err := wa.Setup("admin", "test123456"); err != nil { t.Fatal(err) }
	token, err := wa.Login("admin", "test123456")
	if err != nil { t.Fatal(err) }

	// 构造 Handler
	h := NewHandler(Deps{
		Store: st2,
		Up:    upstream.New(),
		Auth:  wa,
	})

	// GET /api/accounts 带 token cookie
	req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
	req.AddCookie(&http.Cookie{Name: "ts_session", Value: token})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
	}
	var out []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil { t.Fatal(err) }
	if len(out) != 1 { t.Fatalf("返回账号数: %d, want 1", len(out)) }
	acc1 := out[0]

	// 关键字段断言
	_, has := acc1["todayEarned"]
	if !has { t.Fatal("缺少 todayEarned 字段! HTTP body:", w.Body.String()) }
	_, has2 := acc1["todayStatus"]
	if !has2 { t.Fatal("缺少 todayStatus 字段!") }
	te, _ := acc1["todayEarned"].(float64)
	ts, _ := acc1["todayStatus"].(string)
	if int(te) != 200 {
		t.Errorf("todayEarned = %v, want 200 (来自 success 日志)", te)
	}
	if ts != "已签到" {
		t.Errorf("todayStatus = %v, want 已签到", ts)
	}
	// 旧字段仍然存在
	if acc1["hasRefreshToken"] != true {
		t.Errorf("hasRefreshToken 丢失: %v", acc1["hasRefreshToken"])
	}
	if le, _ := acc1["lastEarned"].(float64); int(le) != 0 {
		t.Errorf("lastEarned 应保持 0（被 ALREADY 覆盖），todayEarned 才是正确显示值")
	}
}
