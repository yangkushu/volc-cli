package crtag

import (
	"strings"
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
)

func sp(s string) *string { return &s }

func mkTag(name, pushTime string) *cr.ItemForListTagsOutput {
	return &cr.ItemForListTagsOutput{Name: sp(name), PushTime: sp(pushTime), Type: sp("Image")}
}

var now = time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)

func names(vs []TagView) []string {
	var out []string
	for _, v := range vs {
		out = append(out, *v.Tag.Name)
	}
	return out
}

func TestFilterOlderThan(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("old", "2026-07-01T00:00:00Z"),    // 73 天前
		mkTag("recent", "2026-09-10T00:00:00Z"), // 2 天前
		mkTag("edge", "2026-08-13T00:00:00Z"),   // 恰好 30 天前(不早于阈值, 不候选)
	}
	got := names(Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, Now: now}))
	if len(got) != 1 || got[0] != "old" {
		t.Errorf("older-than 30d 应只圈定 old, got %v", got)
	}
	// Reason 应含规则说明
	vs := Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, Now: now})
	if !strings.Contains(vs[0].Reason, "older-than") {
		t.Errorf("Reason 应说明 older-than 命中, got %q", vs[0].Reason)
	}
}

func TestFilterKeepLast(t *testing.T) {
	// 直接构造明确时间, 避免拼接出错
	var tags []*cr.ItemForListTagsOutput
	for i := 0; i < 12; i++ {
		tags = append(tags, mkTag(string(rune('a'+i)), time.Date(2026, 9, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)))
	}
	got := names(Filter(tags, Criteria{KeepLast: 10, Now: now}))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("keep-last 10 应圈定最早的 a,b, got %v", got)
	}
}

func TestFilterKeepLastUnknownPushTimeNeverCandidate(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("a", "2026-09-01T00:00:00Z"),
		mkTag("b", "2026-09-02T00:00:00Z"),
		mkTag("bad", "garbage"), // PushTime 无法解析
	}
	got := names(Filter(tags, Criteria{KeepLast: 1, Now: now}))
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("PushTime 未知的 bad 永不候选, keep-last 1 只圈定 a, got %v", got)
	}
}

func TestFilterIntersection(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("ci-old1", "2026-07-01T00:00:00Z"),
		mkTag("ci-old2", "2026-07-02T00:00:00Z"),
		mkTag("ci-new", "2026-09-10T00:00:00Z"),
		mkTag("rel-old", "2026-07-03T00:00:00Z"), // 老但无 ci- 前缀
	}
	got := names(Filter(tags, Criteria{
		OlderThan: 30 * 24 * time.Hour,
		TagPrefix: "ci-",
		Now:       now,
	}))
	if len(got) != 2 || got[0] != "ci-old1" || got[1] != "ci-old2" {
		t.Errorf("older-than+prefix 交集应圈定 ci-old1/ci-old2, got %v", got)
	}
}

func TestFilterKeepLastAndOlderThanCombined(t *testing.T) {
	// 12 个老 tag(全部早于 30d) + keep-last 10 → 只圈定最早 2 个
	var tags []*cr.ItemForListTagsOutput
	for i := 0; i < 12; i++ {
		tags = append(tags, mkTag(string(rune('a'+i)), time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)))
	}
	got := names(Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, KeepLast: 10, Now: now}))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("组合条件应只圈定最早 2 个, got %v", got)
	}
}

func TestFilterTagNames(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("v1", "2026-09-10T00:00:00Z"),
		mkTag("v2", "2026-09-11T00:00:00Z"),
	}
	got := names(Filter(tags, Criteria{TagNames: []string{"v2"}, Now: now}))
	if len(got) != 1 || got[0] != "v2" {
		t.Errorf("tag-names 精确匹配应只圈定 v2, got %v", got)
	}
}

func TestFilterZeroCriteriaReturnsAll(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("a", "2026-09-10T00:00:00Z"),
		mkTag("b", "garbage"),
	}
	vs := Filter(tags, Criteria{Now: now})
	if len(vs) != 2 {
		t.Fatalf("零值条件应返回全部, got %d", len(vs))
	}
	for _, v := range vs {
		if v.Reason != "" {
			t.Errorf("零值条件下 Reason 应为空, got %q", v.Reason)
		}
	}
}
