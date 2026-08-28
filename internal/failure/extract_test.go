package failure

import (
	"testing"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
)

func failedStep(name string, kvs ...models.KVPair) models.PipelineRecordStep {
	return models.PipelineRecordStep{Name: name, Status: "Failed", Result: kvs}
}

func TestExtractFindsFailedStep(t *testing.T) {
	rec := &models.PipelineRecord{
		Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "构建", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "build-image", Status: "Failed", Type: "Build",
				Steps: []models.PipelineRecordStep{
					{Name: "checkout", Status: "Success"},
					failedStep("docker-build",
						models.KVPair{Key: "exit_code", Value: "1"},
						models.KVPair{Key: "error", Value: "exited with code 1"}),
				},
			}},
		}},
	}
	got := Extract(rec)
	if len(got) != 1 {
		t.Fatalf("应提取 1 条失败, got %d", len(got))
	}
	f := got[0]
	if f.Stage != "构建" || f.Task != "build-image" || f.Step != "docker-build" {
		t.Errorf("路径错误: %+v", f)
	}
	if f.Message != "exited with code 1" {
		t.Errorf("Message 应取 error KV, got %q", f.Message)
	}
	if len(f.Details) != 2 {
		t.Errorf("Details 应保留全部 Result KV, got %d", len(f.Details))
	}
}

func TestExtractNoFailure(t *testing.T) {
	rec := &models.PipelineRecord{Status: "Success"}
	if got := Extract(rec); len(got) != 0 {
		t.Errorf("成功记录不应有失败项, got %d", len(got))
	}
}

func TestExtractMessageFallback(t *testing.T) {
	rec := &models.PipelineRecord{
		Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "部署", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "deploy", Status: "Failed",
				Steps: []models.PipelineRecordStep{
					{Name: "kubectl", Status: "Failed", Result: []models.KVPair{{Key: "log", Value: "pod crash"}}},
				},
			}},
		}},
	}
	got := Extract(rec)
	if len(got) != 1 || got[0].Message != "pod crash" {
		t.Errorf("无 error/message KV 时应兜底取第一个非空 KV, got %+v", got)
	}
}
