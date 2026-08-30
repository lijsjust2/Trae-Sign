// Package checkin 签到服务：封装"刷新→签到→回写凭证/状态→写日志→推送"完整流程，
// 供 API 与定时调度复用。
package checkin

import (
	"fmt"
	"strings"
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
	case "OK":
		return "success" // 真正签到并获得积分
	case "ALREADY":
		return "already" // 今日已签到（无积分），显示和聚合应区别于 success
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

	// 推送该账号结果（自定义 token 组与手动单账号签到走这里；默认 token 组由 CheckinBatch 合并推送）
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

// CheckinBatch 对一批账号串行签到，可选合并推送（用于默认组：均使用默认 pushplus token）。
// 合并推送时一条消息列出所有账号明细，标题用"TRAE 签到：<总积分>"。
func (s *Service) CheckinBatch(ids []string, mergedPush bool) []Result {
	var results []Result
	for i, id := range ids {
		if i > 0 {
			time.Sleep(2 * time.Second) // 账号间间隔，降低风控
		}
		r, err := s.CheckinAccount(id, false) // 不单独推
		if err != nil {
			acc, _ := s.Store.GetAccount(id)
			r = Result{AccountID: id, Name: displayName(acc), Status: "FAIL", Detail: err.Error()}
		}
		results = append(results, r)
	}

	if mergedPush && len(results) > 0 {
		if def := s.Store.GetSettings().DefaultPushPlusToken; def != "" {
			title, content := buildMergedMessage(results)
			go func() { _ = s.Up.PushPlus(def, title, content) }()
		}
	}
	return results
}

// buildMergedMessage 构造默认组合并推送的标题和内容。
// 标题"TRAE签到：获得积分X"（X 为本次签到获得总积分）；内容含总积分和每账号明细。
func buildMergedMessage(results []Result) (title, content string) {
	var totalRemain, totalEarned int64
	for _, r := range results {
		totalRemain += r.Remain
		totalEarned += r.Earned
	}
	var b strings.Builder
	fmt.Fprintf(&b, "最新总积分 %d，签到获得 %d 积分\n明细如下：\n", totalRemain, totalEarned)
	for i, r := range results {
		fmt.Fprintf(&b, "%d、账号%s，签到+%d积分 当前总积分 %d\n", i+1, r.Name, r.Earned, r.Remain)
	}
	return fmt.Sprintf("TRAE签到：获得积分%d", totalEarned), b.String()
}

// PushPreview 构造默认组合并推送的预览消息（用账号当前积分数据模拟，供设置页测试推送）。
func (s *Service) PushPreview() (title, content string) {
	var results []Result
	for _, a := range s.Store.ListAccounts() {
		if !a.Enabled || a.PushPlusToken != "" {
			continue
		}
		results = append(results, Result{Name: displayName(a.Account), Earned: a.TodayEarned, Remain: a.TotalCredits})
	}
	if len(results) == 0 {
		return "TRAE签到：测试推送",
			"【测试推送】当前没有使用默认 PushPlus Token 的启用账号。\n正式签到时，这类账号的汇总推送格式与此一致。"
	}
	_, detail := buildMergedMessage(results)
	return "TRAE签到：测试推送", "【测试推送】预览合并推送效果：\n" + detail
}

// isDefaultGroup 判断账号是否属于默认组（未单独设置 pushplus token，即使用默认 token）。
// 默认组合并一条推送；自定义组（单独设置 token）单独推送。
func isDefaultGroup(a store.PublicAccount) bool {
	return a.PushPlusToken == ""
}

// CheckinAll 对所有启用账号签到，按推送策略分组（仅看 pushplus token）：
//   - 默认组（未单独设置 token，用默认）：批量串行签到 + 合并一条推送
//   - 自定义组（单独设置 token）：单独签到 + 单独推送
func (s *Service) CheckinAll() []Result {
	accounts := s.Store.ListAccounts()

	var defaultBatch, customIDs []string
	for _, a := range accounts {
		if !a.Enabled {
			continue
		}
		if isDefaultGroup(a) {
			defaultBatch = append(defaultBatch, a.ID)
		} else {
			customIDs = append(customIDs, a.ID)
		}
	}

	var results []Result
	if len(defaultBatch) > 0 {
		results = append(results, s.CheckinBatch(defaultBatch, true)...)
	}
	for i, id := range customIDs {
		if len(results) > 0 || i > 0 {
			time.Sleep(2 * time.Second)
		}
		r, err := s.CheckinAccount(id, true)
		if err != nil {
			acc, _ := s.Store.GetAccount(id)
			r = Result{AccountID: id, Name: displayName(acc), Status: "FAIL", Detail: err.Error()}
		}
		results = append(results, r)
	}
	return results
}
