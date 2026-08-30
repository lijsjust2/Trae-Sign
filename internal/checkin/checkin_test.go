package checkin

import (
	"strings"
	"testing"
)

func TestBuildMergedMessage(t *testing.T) {
	results := []Result{
		{Name: "账号1", Earned: 200, Remain: 642},
		{Name: "账号2", Earned: 200, Remain: 3404},
		{Name: "账号3", Earned: 200, Remain: 2901},
	}
	title, content := buildMergedMessage(results)

	wantTitle := "TRAE签到：获得积分600"
	if title != wantTitle {
		t.Errorf("title = %q, want %q", title, wantTitle)
	}

	wantSubs := []string{
		"最新总积分 6947，签到获得 600 积分",
		"明细如下：",
		"1、账号账号1，签到+200积分 当前总积分 642",
		"2、账号账号2，签到+200积分 当前总积分 3404",
		"3、账号账号3，签到+200积分 当前总积分 2901",
	}
	for _, s := range wantSubs {
		if !strings.Contains(content, s) {
			t.Errorf("content missing %q\nGot:\n%s", s, content)
		}
	}
}

func TestBuildMergedMessageSingle(t *testing.T) {
	results := []Result{{Name: "仅一个", Earned: 50, Remain: 100}}
	title, content := buildMergedMessage(results)
	if title != "TRAE签到：获得积分50" {
		t.Errorf("title = %q", title)
	}
	if !strings.Contains(content, "1、账号仅一个，签到+50积分 当前总积分 100") {
		t.Errorf("content missing detail:\n%s", content)
	}
}
