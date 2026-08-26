// Package checkin 签到服务：封装"刷新→签到→回写凭证/状态→写日志→推送"完整流程，
// 供 API 与定时调度复用。
package checkin

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"trae-signin-web/internal/store"
	"trae-signin-web/internal/upstream"
)

// Result 单账号签到结果。
type Result struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Status    string `json:"status"` // "OK" / "ALREADY" / "FAIL"
	Earned    int64  `json:"earned"`
	Remain    int64  `json:"remain"`
	Detail    string `json:"detail"`
}

// Service 签到服务。
type Service struct {
	Store *store.Store
	Up    *upstream.Client
}

// New 创建签到服务。
func New(s *store.Store, up *upstream.Client) *Service {
	return &Service{Store: s, Up: up}
}

// displayName 显示名：备注 > 昵称 > uid。
func displayName(a store.Account) string {
	if a.Remark != "" {
		return a.Remark
	}
	if a.Nickname != "" {
		return a.Nickname
	}
	if a.UID != "" {
		return a.UID
	}
	return a.ID
}

func nowMs() int64 { return time.Now().UnixMilli() }

func statusToResult(s string) string {
	switch s {
	case "OK", "ALREADY":
		return "success"
	case "RATE_LIMIT":
		return "rate_limited" // 9074 今日配额用尽：当天不重试，次日自动恢复
	default:
		return "failed"
	}
}

// CheckinAccount 对单个账号签到。pushSelf 为 true 时用该账号 pushplus 推送结果。
func (s *Service) CheckinAccount(id string, pushSelf bool) (Result, error) {
	acc, ok := s.Store.GetAccount(id)
	if !ok {
		return Result{}, fmt.Errorf("account not found: %s", id)
	}
	name := displayName(acc)
	res := Result{AccountID: id, Name: name}

	// 签到（DoCheckin 内部会按需刷新 token 并修改 a）
	a := acc.ToAuth()
	cr := s.Up.DoCheckin(a)
	res.Status = cr.Status
	res.Earned = cr.Earned
	res.Remain = cr.Remain
	res.Detail = cr.Detail

	// 回写凭证与状态（DeviceID 可能在 DoCheckin 内因 9074/9095 轮换过，需一并回写）
	back := acc
	back.AccessToken = a.AccessToken
	back.RefreshToken = a.RefreshToken
	back.ExpiresAt = a.ExpiresAt
	back.DeviceID = a.DeviceID
	back.LastCheckinAt = nowMs()
	back.LastCheckinResult = statusToResult(cr.Status)
	back.LastCheckinMessage = cr.Detail
	back.LastEarned = cr.Earned
	if cr.Remain > 0 {
		back.TotalCredits = cr.Remain
	}
	s.Store.UpdateCredentials(id, back)

	// 写日志
	s.Store.AddLog(store.CheckinLog{
		AccountID:   id,
		AccountName: name,
		Time:        nowMs(),
		Result:      statusToResult(cr.Status),
		Message:     cr.Detail,
		Earned:      cr.Earned,
		Remain:      cr.Remain,
	})

	// 推送该账号结果
	if pushSelf {
		token := acc.PushPlusToken
		if token == "" {
			token = s.Store.GetSettings().DefaultPushPlusToken
		}
		if token != "" {
			title := "TRAE 签到：" + name
			content := fmt.Sprintf("账号：%s\n状态：%s\n获得积分：%d\n累计积分：%d",
				name, statusText(cr.Status), cr.Earned, cr.Remain)
			go func() { _ = s.Up.PushPlus(token, title, content) }()
		}
	}
	return res, nil
}

func statusText(s string) string {
	switch s {
	case "OK":
		return "签到成功"
	case "ALREADY":
		return "今日已签到"
	case "RATE_LIMIT":
		return "今日配额已用尽"
	default:
		return "签到失败"
	}
}

// CheckinAll 对所有启用账号签到，每个账号推送自身结果，并汇总推送到默认 pushplus。
func (s *Service) CheckinAll() []Result {
	accounts := s.Store.ListAccounts()
	var enabled []store.PublicAccount
	for _, a := range accounts {
		if a.Enabled {
			enabled = append(enabled, a)
		}
	}

	var (
		results []Result
		mu      sync.Mutex
	)
	for i, a := range enabled {
		if i > 0 {
			time.Sleep(2 * time.Second) // 账号间间隔，降低风控
		}
		r, err := s.CheckinAccount(a.ID, true)
		if err != nil {
			r = Result{AccountID: a.ID, Name: displayName(a.Account), Status: "FAIL", Detail: err.Error()}
		}
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	}

	// 汇总推送（默认 pushplus）
	if def := s.Store.GetSettings().DefaultPushPlusToken; def != "" && len(results) > 0 {
		ok, already, fail := 0, 0, 0
		for _, r := range results {
			switch r.Status {
			case "OK":
				ok++
			case "ALREADY":
				already++
			default:
				fail++
			}
		}
		var b strings.Builder
		fmt.Fprintf(&b, "TRAE 签到汇总\n总计 %d  成功 %d  已签 %d  失败 %d\n\n", len(results), ok, already, fail)
		for _, r := range results {
			fmt.Fprintf(&b, "%s：%s  + %d  累计 %d\n", r.Name, statusText(r.Status), r.Earned, r.Remain)
		}
		go func() { _ = s.Up.PushPlus(def, "TRAE 签到汇总", b.String()) }()
	}
	return results
}
