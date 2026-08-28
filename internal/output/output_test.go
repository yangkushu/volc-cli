package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
)

func TestPrintJSONUsesSDKTags(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSONTo(&buf, models.KVPair{Key: "k", Value: "v"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"Key"`)) {
		t.Errorf("JSON 字段名应保持 SDK 原样 Key/Value, got %s", buf.String())
	}
}

func TestPrintRecordsTable(t *testing.T) {
	var buf bytes.Buffer
	recs := []models.PipelineRecord{{
		Id: "rec-1", Status: "Failed", TriggerMode: "Manual",
		StartTime: "2026-08-27 14:32:10",
		DynamicEnvs: []*models.KVPair{{Key: "BRANCH", Value: "master"}},
	}}
	if err := PrintRecordsTo(&buf, recs); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rec-1", "Failed", "Manual", "BRANCH=master"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("表格应包含 %q, got:\n%s", want, buf.String())
		}
	}
}

func TestPrintRecordDetailFailedPath(t *testing.T) {
	var buf bytes.Buffer
	rec := &models.PipelineRecord{
		Id: "rec-1", Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "构建", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "build", Status: "Failed",
				Steps: []models.PipelineRecordStep{{Name: "docker-build", Status: "Failed"}},
			}},
		}},
	}
	fs := []failure.Failure{{Stage: "构建", Task: "build", Step: "docker-build", Message: "exit 1"}}
	err := PrintRecordDetailTo(&buf, rec, fs, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"rec-1", "Failed", "构建", "docker-build", "exit 1", "https://example.com"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("详情应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintRecordDetailSameStepNameDifferentStage(t *testing.T) {
	var buf bytes.Buffer
	rec := &models.PipelineRecord{
		Id: "rec-1", Status: "Failed",
		Stages: []models.PipelineRecordStage{
			{
				Name: "构建", Status: "Failed",
				Tasks: []models.PipelineRecordTask{{
					Name: "build-a", Status: "Failed",
					Steps: []models.PipelineRecordStep{{Name: "docker-build", Status: "Failed"}},
				}},
			},
			{
				Name: "部署", Status: "Failed",
				Tasks: []models.PipelineRecordTask{{
					Name: "deploy-b", Status: "Failed",
					Steps: []models.PipelineRecordStep{{Name: "docker-build", Status: "Failed"}},
				}},
			},
		},
	}
	fs := []failure.Failure{
		{Stage: "构建", Task: "build-a", Step: "docker-build", Message: "exit 1"},
		{Stage: "部署", Task: "deploy-b", Step: "docker-build", Message: "exit 2"},
	}
	if err := PrintRecordDetailTo(&buf, rec, fs, "https://example.com"); err != nil {
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
