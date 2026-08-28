package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"kuopin/volc-cli/internal/failure"
)

func sptr(s string) *string { return &s }
func iptr(i int64) *int64  { return &i }

func TestPrintJSONUsesSDKTags(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSONTo(&buf, cp.ParameterForListPipelineRunsOutput{Key: sptr("k"), Value: sptr("v")}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"Key"`)) {
		t.Errorf("JSON 字段名应保持 SDK 原样 Key/Value, got %s", buf.String())
	}
}

func TestPrintWorkspacesTable(t *testing.T) {
	var buf bytes.Buffer
	items := []*cp.ItemForListWorkspacesOutput{
		{Id: sptr("ws-1"), Name: sptr("kuopin"), Visibility: sptr("Account"), CreateTime: sptr("2026-08-01 10:00:00")},
		{Id: sptr("ws-2"), Name: sptr("demo"), Visibility: sptr("Specified"), CreateTime: sptr("2026-08-02 11:00:00"), Description: sptr("演示用")},
	}
	if err := PrintWorkspacesTo(&buf, items); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"ws-1", "kuopin", "Account", "ws-2", "Specified", "演示用"} {
		if !strings.Contains(out, want) {
			t.Errorf("工作区表格应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintWorkspacesEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintWorkspacesTo(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "无工作区") {
		t.Errorf("空列表应输出无工作区, got: %s", buf.String())
	}
}

func TestPrintRunsTable(t *testing.T) {
	var buf bytes.Buffer
	runs := []*cp.ItemForListPipelineRunsOutput{{
		Id: sptr("run-1"), Index: iptr(787), Status: sptr("Failed"),
		StartTime: sptr("2026-08-28 16:55:58"),
		Trigger:   &cp.TriggerForListPipelineRunsOutput{Type: sptr("Webhook")},
		Parameters: []*cp.ParameterForListPipelineRunsOutput{
			{Key: sptr("BRANCH"), Value: sptr("master")},
			{Key: sptr("TOKEN"), Value: sptr("secret-value"), Secret: &[]bool{true}[0]},
		},
	}}
	if err := PrintRunsTo(&buf, runs); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"run-1", "787", "Failed", "BRANCH=master", "TOKEN=***"} {
		if !strings.Contains(out, want) {
			t.Errorf("记录表格应包含 %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secret-value") {
		t.Errorf("秘密参数值不应出现在输出中:\n%s", out)
	}
}

func TestPrintRunDetailFailedPath(t *testing.T) {
	var buf bytes.Buffer
	run := &cp.ItemForListPipelineRunsOutput{
		Id: sptr("run-1"), Index: iptr(787), Status: sptr("Failed"),
		Stages: []*cp.StageForListPipelineRunsOutput{{
			Name:   sptr("构建"),
			Status: sptr("Failed"),
			Tasks: []*cp.TaskForListPipelineRunsOutput{{
				Name:   sptr("build-image"),
				Status: sptr("Failed"),
			}},
		}},
	}
	fs := []failure.Failure{{
		Stage: "构建", Task: "build-image", Step: "docker-build",
		Message: "exit 1",
		LogTail: []string{"line1", "line2"},
	}}
	if err := PrintRunDetailTo(&buf, run, fs, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"run-1", "787", "Failed", "构建", "build-image", "docker-build", "exit 1", "line2", "https://example.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("详情应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintRunDetailSameStepNameDifferentStage(t *testing.T) {
	var buf bytes.Buffer
	run := &cp.ItemForListPipelineRunsOutput{
		Id: sptr("run-1"), Status: sptr("Failed"),
		Stages: []*cp.StageForListPipelineRunsOutput{
			{
				Name: sptr("构建"),
				Tasks: []*cp.TaskForListPipelineRunsOutput{{Name: sptr("build-a"), Status: sptr("Failed")}},
			},
			{
				Name: sptr("部署"),
				Tasks: []*cp.TaskForListPipelineRunsOutput{{Name: sptr("deploy-b"), Status: sptr("Failed")}},
			},
		},
	}
	fs := []failure.Failure{
		{Stage: "构建", Task: "build-a", Step: "docker-build", Message: "exit 1"},
		{Stage: "部署", Task: "deploy-b", Step: "docker-build", Message: "exit 2"},
	}
	if err := PrintRunDetailTo(&buf, run, fs, "https://example.com"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// 每条失败消息只应出现一次, 且绑定在各自 stage/task 之下.
	for _, want := range []string{"exit 1", "exit 2"} {
		if c := strings.Count(out, want); c != 1 {
			t.Errorf("失败消息 %q 应恰好出现 1 次, got %d:\n%s", want, c, out)
		}
	}
}

func TestTruncate(t *testing.T) {
	long := strings.Repeat("汉", 300)
	got := truncate(long, 200)
	if len([]rune(got)) != 200+len([]rune("...(截断)")) {
		t.Errorf("截断长度错误: got %d runes", len([]rune(got)))
	}
	if truncate("short", 200) != "short" {
		t.Errorf("短字符串不应截断")
	}
}
