package codepipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// logResult 日志查询的 JSON 输出结构(字段名大写, 与 SDK 风格一致).
type logResult struct {
	RunId    string   `json:"RunId"`
	TaskId   string   `json:"TaskId"`
	StepName string   `json:"StepName"`
	LogTail  []string `json:"LogTail"`
	More     bool     `json:"More"`
}

func newGetTaskRunLogCmd() *cobra.Command {
	var runId, task, step string
	var tail int64
	c := &cobra.Command{
		Use:   "get-task-run-log",
		Short: "GetTaskRunLog: 查询运行中某个 step 的日志",
		Long: "GetTaskRunLog: 定位流水线后查询某次运行(run)的 step 日志.\n" +
			"--task/--step 缺省时自动定位运行中的第一个失败 step.\n" +
			"输出日志尾部(默认 30 行), 完整日志用 --tail 0 或控制台查看.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			if runId == "" {
				return fmt.Errorf("必须用 --run-id 指定运行 ID(可先用 list-pipeline-runs 查看)")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			p, err := findPipelineByName(client, ctx, ws, args[0])
			if err != nil {
				return err
			}
			pipeId := ptrValue(p.Id)

			// 定位 run
			runs, err := client.ListPipelineRuns(ctx, ws, pipeId, 100, "")
			if err != nil {
				return err
			}
			var run *cp.ItemForListPipelineRunsOutput
			for _, r := range runs.Items {
				if volcengine.StringValue(r.Id) == runId {
					run = r
					break
				}
			}
			if run == nil {
				return fmt.Errorf("未找到运行 %q(最近 100 条记录内), 可用 list-pipeline-runs 查看", runId)
			}

			// 定位任务: 显式 --task 优先, 否则取第一个失败 task
			targetTask := task
			if targetTask == "" {
				targetTask = firstFailedTask(run)
				if targetTask == "" {
					return fmt.Errorf("该运行无失败任务, 请用 --task/--step 显式指定")
				}
			}

			// 查 TaskRuns: 拿 TaskRunId 与 step 列表
			taskRuns, err := client.ListTaskRuns(ctx, ws, pipeId, runId, targetTask)
			if err != nil {
				return err
			}
			var taskRunId string
			for _, tr := range taskRuns.Items {
				if tr.Id != nil {
					taskRunId = volcengine.StringValue(tr.Id)
					break
				}
			}
			if taskRunId == "" {
				return fmt.Errorf("未找到任务 %q 的执行记录", targetTask)
			}

			// 定位 step: 显式 --step 优先, 否则取该任务第一个失败 step
			targetStep := step
			if targetStep == "" {
				for _, tr := range taskRuns.Items {
					for _, sp := range tr.Steps {
						if volcengine.StringValue(sp.Status) == "Failed" {
							targetStep = volcengine.StringValue(sp.Name)
							break
						}
					}
					if targetStep != "" {
						break
					}
				}
			}
			if targetStep == "" {
				return fmt.Errorf("任务 %q 无失败 step, 请用 --step 显式指定", targetTask)
			}

			resp, err := client.GetTaskRunLog(ctx, ws, pipeId, runId, targetTask, taskRunId, targetStep)
			if err != nil {
				return err
			}

			if opts.Global.JSON {
				return output.PrintJSON(logResult{
					RunId:    runId,
					TaskId:   targetTask,
					StepName: targetStep,
					LogTail:  tailLines(resp.LogLines, tail),
					More:     volcengine.BoolValue(resp.More),
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "运行: %s  任务: %s  step: %s\n", runId, targetTask, targetStep)
			for _, line := range tailLines(resp.LogLines, tail) {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			if volcengine.BoolValue(resp.More) {
				fmt.Fprintln(cmd.OutOrStdout(), "...(还有更多日志, 控制台查看或 --tail 0 取全量)")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "控制台: %s\n", client.ConsoleURL(ws, pipeId, runId))
			return nil
		},
	}
	c.Flags().StringVar(&runId, "run-id", "", "运行 ID(必填)")
	c.Flags().StringVar(&task, "task", "", "任务 ID, 默认取第一个失败任务")
	c.Flags().StringVar(&step, "step", "", "step 名称, 默认取该任务第一个失败 step")
	c.Flags().Int64Var(&tail, "tail", 30, "日志尾部行数, 0 表示全量")
	return c
}

// firstFailedTask 返回运行中第一个失败 task 的 Id, 无失败返回空.
func firstFailedTask(run *cp.ItemForListPipelineRunsOutput) string {
	for _, st := range run.Stages {
		for _, tk := range st.Tasks {
			if volcengine.StringValue(tk.Status) == "Failed" {
				return volcengine.StringValue(tk.Id)
			}
		}
	}
	return ""
}

// tailLines 取日志尾部 n 行(跳过结尾空行), 0 表示全量.
func tailLines(lines []*string, n int64) []string {
	// 先去尾空行
	end := len(lines)
	for end > 0 && strings.TrimSpace(volcengine.StringValue(lines[end-1])) == "" {
		end--
	}
	lines = lines[:end]
	if n <= 0 || int64(len(lines)) <= n {
		out := make([]string, 0, len(lines))
		for _, l := range lines {
			out = append(out, strings.TrimRight(volcengine.StringValue(l), "\n"))
		}
		return out
	}
	out := make([]string, 0, n)
	for i := int64(len(lines)) - n; i < int64(len(lines)); i++ {
		out = append(out, strings.TrimRight(volcengine.StringValue(lines[i]), "\n"))
	}
	return out
}
