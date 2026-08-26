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
func (s *Scheduler) tick() {
	settings := s.Store.GetSettings()
	if !settings.AutoCheckin {
		return
	}
	now := time.Now()
	nowMs := now.UnixMilli()
	hhmm := now.Format("15:04")

	for _, a := range s.Store.ListAccounts() {
		if !a.Enabled {
			continue
		}
		// 今天已成功签到或今日配额已用尽(9074) → 跳过
		// （rate_limited 当天不重试，重试只会多扣 claim 配额，次日自动恢复）
		if isSameDayMs(a.LastCheckinAt) && (a.LastCheckinResult == "success" || a.LastCheckinResult == "rate_limited") {
			continue
		}
		// 失败重试冷却：距上次尝试不足 10 分钟 → 跳过
		if a.LastCheckinAt > 0 && nowMs-a.LastCheckinAt < 10*60*1000 {
			continue
		}
		// 触发条件：到点，或上次失败需重试（rate_limited 不重试）
		t := a.CheckinTime
		if t == "" {
			t = settings.DefaultCheckinTime
		}
		atPoint := t == hhmm
		needRetry := a.LastCheckinResult == "failed"
		if !atPoint && !needRetry {
			continue
		}
		// 随机延迟 0~60s，错开整点风控
		go func(id string) {
			delay := time.Duration(now.UnixNano()%61) * time.Second
			time.Sleep(delay)
			_, _ = s.Svc.CheckinAccount(id, true)
		}(a.ID)
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
