package failure

import (
	"strings"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/cp"
)

func sptr(s string) *string { return &s }

func TestExtractFindsFailedStep(t *testing.T) {
	run := &cp.ItemForListPipelineRunsOutput{
		Status: sptr("Failed"),
		Stages: []*cp.StageForListPipelineRunsOutput{{
			Name:   sptr("构建"),
			Status: sptr("Failed"),
			Tasks: []*cp.TaskForListPipelineRunsOutput{{
				Name:   sptr("build-image"),
				Status: sptr("Failed"),
				Id:     sptr("task-1"),
			}},
		}},
	}
	taskRuns := map[string][]*cp.ItemForListTaskRunsOutput{
		"task-1": {{
			Status: sptr("Failed"),
			Steps: []*cp.StepForListTaskRunsOutput{
				{Name: sptr("checkout"), Status: sptr("Succeeded")},
				{Name: sptr("docker-build"), Status: sptr("Failed")},
			},
		}},
	}
	got := Extract(run, taskRuns)
	if len(got) != 1 {
		t.Fatalf("应提取 1 条失败, got %d", len(got))
	}
	f := got[0]
	if f.Stage != "构建" || f.Task != "build-image" || f.Step != "docker-build" {
		t.Errorf("路径错误: %+v", f)
	}
}

func TestExtractNoFailure(t *testing.T) {
	run := &cp.ItemForListPipelineRunsOutput{Status: sptr("Succeeded")}
	if got := Extract(run, nil); len(got) != 0 {
		t.Errorf("成功运行不应有失败项, got %d", len(got))
	}
}

func TestExtractNilRun(t *testing.T) {
	if got := Extract(nil, nil); len(got) != 0 {
		t.Errorf("nil run 应返回空, got %d", len(got))
	}
}

func TestExtractTaskFailedNoTaskRuns(t *testing.T) {
	run := &cp.ItemForListPipelineRunsOutput{
		Stages: []*cp.StageForListPipelineRunsOutput{{
			Name: sptr("部署"),
			Tasks: []*cp.TaskForListPipelineRunsOutput{{
				Name:   sptr("deploy"),
				Status: sptr("Failed"),
				Id:     sptr("task-2"),
			}},
		}},
	}
	got := Extract(run, nil)
	if len(got) != 1 {
		t.Fatalf("无 taskRuns 时应按 task 级提取 1 条, got %d", len(got))
	}
	if got[0].Task != "deploy" || got[0].Step != "" {
		t.Errorf("task 级失败不应有 step, got %+v", got[0])
	}
}

func TestExtractTaskFailedNoFailedSteps(t *testing.T) {
	run := &cp.ItemForListPipelineRunsOutput{
		Stages: []*cp.StageForListPipelineRunsOutput{{
			Name: sptr("构建"),
			Tasks: []*cp.TaskForListPipelineRunsOutput{{
				Name:   sptr("build"),
				Status: sptr("Failed"),
				Id:     sptr("task-3"),
			}},
		}},
	}
	// taskRun 存在但所有 step 都成功(失败信息在 task 级)
	taskRuns := map[string][]*cp.ItemForListTaskRunsOutput{
		"task-3": {{
			Status: sptr("Failed"),
			Steps:  []*cp.StepForListTaskRunsOutput{{Name: sptr("s1"), Status: sptr("Succeeded")}},
		}},
	}
	got := Extract(run, taskRuns)
	if len(got) != 1 || got[0].Step != "" {
		t.Errorf("应回退 task 级失败, got %+v", got)
	}
}

func TestTailLogLines(t *testing.T) {
	lines := []*string{
		sptr("line1\n"),
		sptr(""),
		sptr("line2\n"),
		sptr("   \n"),
		sptr("line3"),
	}
	got := TailLogLines(lines, 3)
	if len(got) != 3 {
		t.Fatalf("应取 3 行, got %d: %v", len(got), got)
	}
	if strings.Join(got, "|") != "line1|line2|line3" {
		t.Errorf("顺序或内容错误: %v", got)
	}
	if got2 := TailLogLines(lines, 10); len(got2) != 3 {
		t.Errorf("行数不足时应返回全部, got %d", len(got2))
	}
}
