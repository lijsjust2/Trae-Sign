package store

import (
	"os"
	"testing"
	"time"
)

// 验证：TodayEarned 从今日日志里的 success 条目累加，无视 account.LastEarned
// 场景：真实签到 earned=200，接着一次 ALREADY 尝试把 LastEarned 覆盖成 0。
// 卡片显示时应依然展示 200。
func TestTodayEarnedFromLogs(t *testing.T) {
	path := filepathTemp(t)
	s, _ := New(path)
	defer os.Remove(path)

	// 造账号（LastEarned 被 ALREADY 覆盖成 0 的典型场景）
	acc := Account{
		ID: "a1", UID: "u1", Nickname: "41015",
		LastCheckinAt:      time.Now().Add(-time.Minute).UnixMilli(),
		LastCheckinResult:  "success",
		LastCheckinMessage: "今日已签到",
		LastEarned:         0, // 被 ALREADY 覆盖，脏数据
		TotalCredits:       2874,
	}
	s.d.Accounts = append(s.d.Accounts, acc)

	// 日志：今日真实获得 200（earned>0）
	todayStart := time.Now().Truncate(24 * time.Hour).Add(-time.Hour) // 今日内
	s.d.Logs = []CheckinLog{
		{
			ID: "l1", AccountID: "a1", AccountName: "41015",
			Time: todayStart.Add(30 * time.Minute).UnixMilli(),
			Result: "success", Earned: 200, Remain: 2973,
			Message: "签到成功",
		},
		{ // 第二遍手动点签到 → ALREADY，earned=0
			ID: "l2", AccountID: "a1", AccountName: "41015",
			Time: todayStart.Add(2 * time.Hour).UnixMilli(),
			Result: "already", Earned: 0, Remain: 2874, // 命名将在下一步改成 "already"
			Message: "今日已签到",
		},
	}
	_ = s.save()

	got := s.TodayEarned("a1")
	if got != 200 {
		t.Errorf("TodayEarned = %d, want 200 (from success log entry)", got)
	}

	// 另一账号今天没日志 → 0
	if got := s.TodayEarned("not-exist"); got != 0 {
		t.Errorf("no log: TodayEarned = %d, want 0", got)
	}

	// 只有 already 的 0 积分条目 → 0
	noEarn := filepathTemp(t)
	s2, _ := New(noEarn)
	defer os.Remove(noEarn)
	s2.d.Accounts = append(s2.d.Accounts, Account{ID: "a2", UID: "u2"})
	s2.d.Logs = []CheckinLog{
		{ID: "l3", AccountID: "a2", Time: time.Now().UnixMilli(), Result: "already", Earned: 0},
	}
	_ = s2.save()
	if got := s2.TodayEarned("a2"); got != 0 {
		t.Errorf("only already entries: TodayEarned = %d, want 0", got)
	}
}

// 验证：今日签到状态按日志成功条目判断，不因 later ALREADY 覆盖成 success 而混淆。
func TestTodayCheckinStatusFromLogs(t *testing.T) {
	path := filepathTemp(t)
	s, _ := New(path)
	defer os.Remove(path)

	s.d.Accounts = []Account{{ID: "a1", UID: "u1"}}
	// 今日内只有 success +200 → 应返回已签到
	todayStart := time.Now().Truncate(24 * time.Hour).Add(time.Hour)
	s.d.Logs = []CheckinLog{
		{ID: "l1", AccountID: "a1", Time: todayStart.UnixMilli(), Result: "success", Earned: 200},
	}
	_ = s.save()
	if status := s.TodayCheckinStatus("a1"); status != "已签到" {
		t.Errorf("has success log today: TodayCheckinStatus = %q, want 已签到", status)
	}

	// 今天没日志 → 未签到
	s.d.Logs = nil
	_ = s.save()
	if status := s.TodayCheckinStatus("a1"); status != "未签到" {
		t.Errorf("no log today: TodayCheckinStatus = %q, want 未签到", status)
	}

	// 昨天有成功日志，今天没有 → 未签到
	yesterday := time.Now().Add(-25 * time.Hour)
	s.d.Logs = []CheckinLog{
		{ID: "l2", AccountID: "a1", Time: yesterday.UnixMilli(), Result: "success", Earned: 200},
	}
	_ = s.save()
	if status := s.TodayCheckinStatus("a1"); status != "未签到" {
		t.Errorf("only yesterday log: TodayCheckinStatus = %q, want 未签到", status)
	}
}

// ListAccounts 返回的 PublicAccount.todayEarned / todayStatus 必须来自今日 success 日志聚合，
// 不能用 LastEarned（会被 ALREADY 覆盖）。这是端到端集成测试：模拟真实数据结构。
func TestListAccountsTodayEarnedFromLogs(t *testing.T) {
	path := filepathTemp(t)
	s, _ := New(path)
	defer os.Remove(path)

	s.d.Accounts = []Account{
		{ // 场景1：今日 success +200，随后 ALREADY LastEarned 覆盖为 0
			ID: "a1", UID: "u1", Nickname: "41015",
			LastEarned: 0, // 被 ALREADY 污染
			LastCheckinResult: "already",
		},
		{ // 场景2：今日两条 success +200 (同账号不同时段) → 累加？实际上每天只能 +200，但防御性编程：应累加
			ID: "a2", UID: "u2", Nickname: "1044", LastEarned: 999,
		},
		{ // 场景3：今天没日志
			ID: "a3", UID: "u3", Nickname: "xxx", LastEarned: 100,
		},
	}

	todayM := time.Now().UnixMilli() // 今日内时间戳
	s.d.Logs = []CheckinLog{
		{ID: "l1", AccountID: "a1", Time: todayM, Result: "success", Earned: 200},
		{ID: "l2", AccountID: "a1", Time: todayM + 1000, Result: "already", Earned: 0}, // ALREADY 不应计入
		{ID: "l3", AccountID: "a2", Time: todayM, Result: "success", Earned: 200},
		{ID: "l4", AccountID: "a3", Time: todayM - 25*3600*1000, Result: "success", Earned: 200}, // 昨天，不计
	}
	_ = s.save()

	accs := s.ListAccounts()
	byID := map[string]PublicAccount{}
	for _, pa := range accs { byID[pa.ID] = pa }

	if got := byID["a1"].TodayEarned; got != 200 {
		t.Errorf("a1 TodayEarned = %d, want 200 (从 success 日志，无视 LastEarned=0 与 already 条目)", got)
	}
	if got := byID["a1"].TodayStatus; got != "已签到" {
		t.Errorf("a1 TodayStatus = %q, want 已签到", got)
	}
	if got := byID["a2"].TodayEarned; got != 200 {
		t.Errorf("a2 TodayEarned = %d, want 200", got)
	}
	if got := byID["a2"].TodayStatus; got != "已签到" {
		t.Errorf("a2 TodayStatus = %q, want 已签到", got)
	}
	if got := byID["a3"].TodayEarned; got != 0 {
		t.Errorf("a3 TodayEarned = %d, want 0 (今天没日志)", got)
	}
	if got := byID["a3"].TodayStatus; got != "未签到" {
		t.Errorf("a3 TodayStatus = %q, want 未签到", got)
	}

	// 关键：hasRefreshToken 等旧字段也正确返回，嵌入 Account 字段不能缺失
	if !byID["a1"].HasRefreshToken && byID["a1"].UID != "u1" {
		t.Errorf("PublicAccount 嵌入字段丢失: %+v", byID["a1"])
	}
}

func filepathTemp(t *testing.T) string {
	f, err := os.CreateTemp("", "store-*.json")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)
	return path
}
