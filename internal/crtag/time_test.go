package crtag

import (
	"testing"
	"time"
)

func TestParseOlderThan(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"12h", 12 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0h", 0, false},
		{"100d", 100 * 24 * time.Hour, false},
		{"30", 0, true},
		{"d", 0, true},
		{"30m", 0, true},
		{"30w", 0, true},
		{"-1d", 0, true},
		{"", 0, true},
		{"3.5d", 0, true},
	}
	for _, c := range cases {
		got, err := ParseOlderThan(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseOlderThan(%q) 应报错, got %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseOlderThan(%q) 意外报错: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseOlderThan(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParsePushTime(t *testing.T) {
	cases := []struct {
		in   string
		want time.Time
		ok   bool
	}{
		// RFC3339
		{"2026-09-01T10:00:00Z", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), true},
		{"2026-09-01T10:00:00+08:00", time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC), true},
		// 无时区空格分隔, 按 UTC
		{"2026-09-01 10:00:00", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), true},
		// Unix 秒 / 毫秒
		{"1760000000", time.Unix(1760000000, 0).UTC(), true},
		{"1760000000123", time.Unix(1760000000, 123000000).UTC(), true},
		// 失败
		{"", time.Time{}, false},
		{"not-a-time", time.Time{}, false},
		{"2026-09-01", time.Time{}, false},
	}
	for _, c := range cases {
		got, ok := ParsePushTime(c.in)
		if ok != c.ok {
			t.Errorf("ParsePushTime(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && !got.Equal(c.want) {
			t.Errorf("ParsePushTime(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
