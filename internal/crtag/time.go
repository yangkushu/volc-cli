// Package crtag 实现镜像 Tag 清理候选的圈定逻辑(纯数据处理, 不做 IO).
package crtag

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var olderThanRe = regexp.MustCompile(`^(\d+)([dh])$`)

// ParseOlderThan 解析 --older-than 参数, 仅支持 Nd(天)/Nh(小时), 如 30d/12h.
func ParseOlderThan(s string) (time.Duration, error) {
	m := olderThanRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("--older-than 格式非法: %q, 仅支持如 30d(天)/12h(小时)", s)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("--older-than 数字非法: %q", s)
	}
	d := time.Duration(n) * time.Hour
	if m[2] == "d" {
		d *= 24
	}
	return d, nil
}

// ParsePushTime 容错解析 SDK 返回的 PushTime 字符串, 依次尝试:
// RFC3339 / "2006-01-02 15:04:05"(按 UTC) / Unix 秒(13 位按毫秒).
// 全部失败返回 ok=false; 该类 tag 永不进入清理候选(宁可漏删不可错删).
func ParsePushTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), true
	}
	if sec, err := strconv.ParseInt(s, 10, 64); err == nil && sec > 0 {
		if sec > 1e12 { // 13 位毫秒时间戳
			return time.Unix(sec/1e3, (sec%1e3)*1e6).UTC(), true
		}
		return time.Unix(sec, 0).UTC(), true
	}
	return time.Time{}, false
}
