package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"kuopin/volc-cli/internal/crtag"
)

func TestPrintRegistriesTable(t *testing.T) {
	var buf bytes.Buffer
	items := []*cr.ItemForListRegistriesOutput{
		{Name: sptr("reg-1"), Type: sptr("Standard"), CreateTime: sptr("2026-01-01T00:00:00Z"),
			Status: &cr.StatusForListRegistriesOutput{Phase: sptr("Running")}},
	}
	if err := PrintRegistriesTo(&buf, items); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"reg-1", "Standard", "Running"} {
		if !strings.Contains(out, want) {
			t.Errorf("实例表格应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintRegistriesEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintRegistriesTo(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "无镜像仓库实例") {
		t.Errorf("空列表应提示无实例(并提醒检查 --region), got: %s", buf.String())
	}
}

func TestPrintNamespacesAndRepositories(t *testing.T) {
	var buf bytes.Buffer
	ns := []*cr.ItemForListNamespacesOutput{{Name: sptr("team-a"), CreateTime: sptr("2026-01-01T00:00:00Z")}}
	if err := PrintNamespacesTo(&buf, ns); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "team-a") {
		t.Errorf("命名空间表格应包含 team-a, got:\n%s", buf.String())
	}

	buf.Reset()
	repos := []*cr.ItemForListRepositoriesOutput{
		{Name: sptr("svc-api"), Namespace: sptr("team-a"), AccessLevel: sptr("Private")},
	}
	if err := PrintRepositoriesTo(&buf, repos, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "svc-api") || !strings.Contains(buf.String(), "Private") {
		t.Errorf("制品仓库表格应包含 svc-api/Private, got:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), "TAG_COUNT") {
		t.Errorf("无计数时不应有 TAG_COUNT 列, got:\n%s", buf.String())
	}
}

func TestPrintRepositoriesWithTagCount(t *testing.T) {
	var buf bytes.Buffer
	repos := []*cr.ItemForListRepositoriesOutput{
		{Name: sptr("full"), Namespace: sptr("ns")},
		{Name: sptr("miss"), Namespace: sptr("ns")},
	}
	counts := map[string]int64{"ns/full": 100}
	if err := PrintRepositoriesTo(&buf, repos, counts); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"TAG_COUNT", "100", "miss", "-"} {
		if !strings.Contains(out, want) {
			t.Errorf("计数表格应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintTagsTable(t *testing.T) {
	var buf bytes.Buffer
	views := []crtag.TagView{
		{Tag: &cr.ItemForListTagsOutput{Name: sptr("v1"), PushTime: sptr("2026-09-01T00:00:00Z"),
			Type: sptr("Image"), Size: iptr(150 * 1024 * 1024), Digest: sptr("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")}},
		{Tag: &cr.ItemForListTagsOutput{Name: sptr("bad"), PushTime: sptr("garbage"), Type: sptr("Image")}},
	}
	// 带 REASON 列(过滤模式)
	if err := PrintTagsTo(&buf, views, false, true); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"v1", "150.0MB", "sha256:0123456789ab…", "未知"} {
		if !strings.Contains(out, want) {
			t.Errorf("tag 表格应包含 %q, got:\n%s", want, out)
		}
	}
	// 带 REPOSITORY 列(多 repo 模式)
	buf.Reset()
	views[0].Reason = "older-than:720h"
	if err := PrintTagsTo(&buf, views, true, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "older-than:720h") {
		t.Errorf("多 repo+过滤模式应含 REASON, got:\n%s", buf.String())
	}
}

func TestPrintDeleteTagsResult(t *testing.T) {
	var buf bytes.Buffer
	failures := []*cr.FailureForDeleteTagsOutput{{Name: sptr("t3"), Reason: sptr("immutable tag")}}
	if err := PrintDeleteTagsResultTo(&buf, []string{"t1", "t2"}, []string{"t4"}, failures); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"t1", "t2", "t3", "immutable tag", "t4", "失败 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("删除结果应包含 %q, got:\n%s", want, out)
		}
	}
}
