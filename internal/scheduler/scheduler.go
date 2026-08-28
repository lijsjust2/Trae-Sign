// Package scheduler 定时调度：每分钟检查到点账号并签到。
package scheduler

import (
	"time"

	"trae-signin-web/internal/checkin"
	"trae-signin-web/internal/store"
)

// Scheduler 定时签到调度器。
type Scheduler struct {
	Store *store.Store
	Svc   *checkin.Service
	stop  chan struct{}
	done  chan struct{}
}

// New 创建调度器。
func New(s *store.Store, svc *checkin.Service) *Scheduler {
	return &Scheduler{Store: s, Svc: svc, stop: make(chan struct{}), done: make(chan struct{})}
}

// Start 启动调度（非阻塞）。
func (s *Scheduler) Start() {
	go s.loop()
}

// Stop 停止调度。
func (s *Scheduler) Stop() {
	close(s.stop)
	<-s.done
}

// loop 每分钟整点附近检查一次。
func (s *Scheduler) loop() {
	defer close(s.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.tick()
		}
	}
}

// tick 检查当前 HH:mm 是否匹配某账号签到时间，且今日未签到则签到。
// 推送分组：默认组（用默认签到时间 + 用默认 pushplus token）批量签到+合并推送；
// 自定义组（自定义时间或自定义 token）单独签到+单独推送。
func (s *Scheduler) tick() {
	settings := s.Store.GetSettings()
	if !settings.AutoCheckin {
		return
	}
	now := time.Now()
	nowMs := now.UnixMilli()
	hhmm := now.Format("15:04")

	// 收集这一分钟到点且需要签到的账号
	var defaultBatch []string // 默认组：合并推送
	var customIDs []string    // 自定义组：单独推送
	for _, a := range s.Store.ListAccounts() {
		if !a.Enabled {
			continue
		}
		// 今天已成功签到或今日配额已用尽(9074) → 跳过
		if isSameDayMs(a.LastCheckinAt) && (a.LastCheckinResult == "success" || a.LastCheckinResult == "already" || a.LastCheckinResult == "rate_limited") {
			continue
		}
		// 失败重试冷却：距上次尝试不足 10 分钟 → 跳过
		if a.LastCheckinAt > 0 && nowMs-a.LastCheckinAt < 10*60*1000 {
			continue
		}
		t := a.CheckinTime
		if t == "" {
			t = settings.DefaultCheckinTime
		}
		atPoint := t == hhmm
		needRetry := a.LastCheckinResult == "failed"
		if !atPoint && !needRetry {
			continue
		}

		// 分组：默认时间（空或等于默认）+ 用默认 pushplus（无自定义 token）→ 默认组
		isDefaultTime := a.CheckinTime == "" || a.CheckinTime == settings.DefaultCheckinTime
		isDefaultPush := a.PushPlusToken == ""
		if isDefaultTime && isDefaultPush && atPoint {
			defaultBatch = append(defaultBatch, a.ID)
		} else {
			customIDs = append(customIDs, a.ID)
		}
	}

	// 默认组：批量串行签到 + 合并一条推送
	if len(defaultBatch) > 0 {
		go func(ids []string) {
			delay := time.Duration(time.Now().UnixNano()%61) * time.Second
			time.Sleep(delay)
			_ = s.Svc.CheckinBatch(ids, true)
		}(defaultBatch)
	}
	// 自定义组：单独签到 + 单独推送
	for _, id := range customIDs {
		go func(id string) {
			delay := time.Duration(time.Now().UnixNano()%61) * time.Second
			time.Sleep(delay)
			_, _ = s.Svc.CheckinAccount(id, true)
		}(id)
	}
}

// isSameDayMs 判断毫秒戳是否是今天。
func isSameDayMs(ms int64) bool {
	if ms <= 0 {
		return false
	}
	t := time.UnixMilli(ms)
	n := time.Now()
	return t.Year() == n.Year() && t.Month() == n.Month() && t.Day() == n.Day()
}
