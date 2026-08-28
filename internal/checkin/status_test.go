package checkin

import "testing"

// 要求：statusToResult 要区分 OK 真正得积分（success）与 ALREADY 今日已签（already），
// 这样日志显示和聚合今日积分都能正确区分，不会被 ALREADY 的 0 积分覆盖真实结果。
func TestStatusToResult(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"OK", "success"},
		{"ALREADY", "already"}, // 关键：ALREADY 不能再映射为 success，否则覆盖真实积分
		{"RATE_LIMIT", "rate_limited"},
		{"FAIL", "failed"},
		{"其他未知", "failed"},
		{"", "failed"},
	}
	for _, c := range cases {
		got := statusToResult(c.status)
		if got != c.want {
			t.Errorf("statusToResult(%q) = %q, want %q", c.status, got, c.want)
		}
	}
}
