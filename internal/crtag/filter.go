package crtag

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
)

// Criteria 清理候选过滤条件, 多条件为交集; 零值字段不生效, 全零值返回全部(Reason 为空).
type Criteria struct {
	OlderThan time.Duration // >0: PushTime 早于 Now-OlderThan 才候选
	KeepLast  int           // >0: 按 PushTime 降序保留最近 KeepLast 个, 排名 >= KeepLast 才候选
	Oldest    int           // >0: 按 PushTime 升序取最早 Oldest 个才候选(与 KeepLast 语义对称)
	TagPrefix string        // 非空: tag 名需有该前缀
	TagNames  []string      // 非空: tag 名需精确命中
	Now       time.Time     // 时间基准, 零值取 time.Now()(测试注入)
}

// TagView 单个 tag 的过滤结果视图.
type TagView struct {
	Tag        *cr.ItemForListTagsOutput
	Repository string // 填充方为 cmd 层
	Reason     string // 命中规则串(如 "older-than:720h+beyond-keep-last:10"), 无过滤条件时为空
}

// Filter 按 Criteria 圈定候选 tag(交集语义), 保持输入顺序.
// PushTime 解析失败的 tag 永不进入候选(宁漏删不错删).
func Filter(tags []*cr.ItemForListTagsOutput, c Criteria) []TagView {
	now := c.Now
	if now.IsZero() {
		now = time.Now()
	}
	if c.OlderThan == 0 && c.KeepLast == 0 && c.Oldest == 0 && c.TagPrefix == "" && len(c.TagNames) == 0 {
		views := make([]TagView, 0, len(tags))
		for _, t := range tags {
			views = append(views, TagView{Tag: t})
		}
		return views
	}
	var rank map[string]int
	if c.KeepLast > 0 || c.Oldest > 0 {
		rank = keepLastRank(tags)
	}
	var out []TagView
	for _, t := range tags {
		name := ""
		if t.Name != nil {
			name = *t.Name
		}
		var reasons []string
		if c.TagPrefix != "" {
			if !strings.HasPrefix(name, c.TagPrefix) {
				continue
			}
			reasons = append(reasons, "prefix:"+c.TagPrefix)
		}
		if len(c.TagNames) > 0 {
			if !containsName(c.TagNames, name) {
				continue
			}
			reasons = append(reasons, "exact")
		}
		if c.OlderThan > 0 {
			pt, ok := parsePushTimeOf(t)
			if !ok || !pt.Before(now.Add(-c.OlderThan)) {
				continue
			}
			reasons = append(reasons, "older-than:"+formatDurationForReason(c.OlderThan))
		}
		if c.KeepLast > 0 {
			r, ok := rank[name]
			if !ok || r < c.KeepLast {
				continue
			}
			reasons = append(reasons, "beyond-keep-last:"+strconv.Itoa(c.KeepLast))
		}
		if c.Oldest > 0 {
			// rank 为降序名次(0=最新); 升序名次 = len(rank)-1-r, 候选需升序名次 < Oldest
			r, ok := rank[name]
			if !ok || r < len(rank)-c.Oldest {
				continue
			}
			reasons = append(reasons, "oldest:"+strconv.Itoa(c.Oldest))
		}
		out = append(out, TagView{Tag: t, Reason: strings.Join(reasons, "+")})
	}
	return out
}

// keepLastRank 返回按 PushTime 降序的名次(0=最新); PushTime 未知的 tag 不在结果中(永不淘汰).
func keepLastRank(tags []*cr.ItemForListTagsOutput) map[string]int {
	type named struct {
		name string
		t    time.Time
	}
	var known []named
	for _, t := range tags {
		if pt, ok := parsePushTimeOf(t); ok {
			name := ""
			if t.Name != nil {
				name = *t.Name
			}
			known = append(known, named{name, pt})
		}
	}
	sort.SliceStable(known, func(i, j int) bool { return known[i].t.After(known[j].t) })
	rank := make(map[string]int, len(known))
	for i, k := range known {
		rank[k.name] = i
	}
	return rank
}

func parsePushTimeOf(t *cr.ItemForListTagsOutput) (time.Time, bool) {
	if t.PushTime == nil {
		return time.Time{}, false
	}
	return ParsePushTime(*t.PushTime)
}

// formatDurationForReason 将 OlderThan 时长渲染为紧凑形式: 整天显示 Nd, 否则 Nh(输入仅支持 d/h).
func formatDurationForReason(d time.Duration) string {
	h := int64(d / time.Hour)
	if h > 0 && h%24 == 0 {
		return strconv.FormatInt(h/24, 10) + "d"
	}
	return strconv.FormatInt(h, 10) + "h"
}

func containsName(names []string, s string) bool {
	for _, n := range names {
		if n == s {
			return true
		}
	}
	return false
}
