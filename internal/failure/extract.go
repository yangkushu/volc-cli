// Package failure 从 V2 流水线执行记录(PipelineRun/TaskRun)中提取失败定位信息.
package failure

import (
	"strings"

	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
)

// Failure 一处失败的定位路径与错误信息.
type Failure struct {
	Stage   string   `json:"Stage"`
	Task    string   `json:"Task"`
	Step    string   `json:"Step"`
	Message string   `json:"Message"`
	LogTail []string `json:"LogTail,omitempty"`
}

// Extract 从 Run 与 TaskRuns 数据中收集全部失败 step.
// taskRuns 为 nil 时仅基于 Run.Stages 的 task 级状态提取(不细分 step).
func Extract(run *cp.ItemForListPipelineRunsOutput, taskRuns map[string][]*cp.ItemForListTaskRunsOutput) []Failure {
	if run == nil {
		return nil
	}
	var out []Failure
	for _, st := range run.Stages {
		if st == nil || st.Tasks == nil {
			continue
		}
		for _, tk := range st.Tasks {
			if tk == nil || tk.Status == nil {
				continue
			}
			stage := volcengine.StringValue(st.Name)
			task := volcengine.StringValue(tk.Name)
			taskId := volcengine.StringValue(tk.Id)
			if volcengine.StringValue(tk.Status) == "Failed" {
				if runs, ok := taskRuns[taskId]; ok {
					// 细分 step: 收集 taskRun 中的失败 step
					found := false
					for _, tr := range runs {
						if tr == nil {
							continue
						}
						for _, sp := range tr.Steps {
							if sp == nil || volcengine.StringValue(sp.Status) != "Failed" {
								continue
							}
							found = true
							out = append(out, Failure{
								Stage:   stage,
								Task:    task,
								Step:    volcengine.StringValue(sp.Name),
								Message: "step 失败, 详见日志与控制台",
							})
						}
					}
					if !found {
						out = append(out, Failure{
							Stage:   stage,
							Task:    task,
							Message: "task 失败(无 step 级信息), 详见控制台",
						})
					}
				} else {
					out = append(out, Failure{
						Stage:   stage,
						Task:    task,
						Message: "task 失败, 详见控制台",
					})
				}
			}
		}
	}
	return out
}

// BuildTaskRunMap 将 TaskRuns 列表按 TaskId 分组(供 Extract 使用).
func BuildTaskRunMap(taskRuns []*cp.ItemForListTaskRunsOutput) map[string][]*cp.ItemForListTaskRunsOutput {
	m := make(map[string][]*cp.ItemForListTaskRunsOutput)
	for _, tr := range taskRuns {
		if tr == nil {
			continue
		}
		tid := volcengine.StringValue(tr.TaskId)
		m[tid] = append(m[tid], tr)
	}
	return m
}

// TailLogLines 取日志尾部 n 行(正序), 去掉空行.
func TailLogLines(lines []*string, n int) []string {
	out := make([]string, 0, n)
	for i := len(lines) - 1; i >= 0 && len(out) < n; i-- {
		s := strings.TrimRight(volcengine.StringValue(lines[i]), "\n")
		if strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	// 反转回正序
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
