package codepipeline

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/cpclient"
	"kuopin/volc-cli/internal/failure"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newFailuresCmd() *cobra.Command {
	var pageSize int64
	var tail int64
	c := &cobra.Command{
		Use:   "failures",
		Short: "最近 N 次失败运行及错误日志(默认 1 次)",
		Long: "组合命令: ListPipelineRuns(Filter=Failed) + ListTaskRuns + GetTaskRunLog.\n" +
			"对每条失败运行提取 Stage→Task→Step 失败路径, 拉取失败 step 日志尾部.\n" +
			"--page-size 1 时即'最后一次失败运行'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			ws, err := resolveWs(client, context.Background())
			if err != nil {
				return err
			}
			ctx := context.Background()
			p, err := findPipelineByName(client, ctx, ws, args[0])
			if err != nil {
				return err
			}
			pipeId := ptrValue(p.Id)

			list, err := client.ListPipelineRuns(ctx, ws, pipeId, pageSize, "Failed")
			if err != nil {
				return err
			}
			if len(list.Items) == 0 {
				if opts.Global.JSON {
					return output.PrintJSON(map[string]any{
						"PipelineId": pipeId, "PipelineName": ptrValue(p.Name), "Items": []any{},
					})
				}
				fmt.Fprintln(cmd.OutOrStdout(), "无失败运行")
				return nil
			}

			type item struct {
				Run        *cp.ItemForListPipelineRunsOutput `json:"Run"`
				Failures   []failure.Failure                 `json:"Failures"`
				ConsoleURL string                            `json:"ConsoleURL"`
			}
			items := make([]item, 0, len(list.Items))
			for _, run := range list.Items {
				fs := extractFailures(client, ctx, ws, pipeId, run, tail)
				items = append(items, item{
					Run:        run,
					Failures:   fs,
					ConsoleURL: client.ConsoleURL(ws, pipeId, volcengine.StringValue(run.Id)),
				})
			}
			if opts.Global.JSON {
				return output.PrintJSON(map[string]any{
					"PipelineId": pipeId, "PipelineName": ptrValue(p.Name), "Items": items,
				})
			}
			for _, it := range items {
				if err := output.PrintRunDetail(it.Run, it.Failures, it.ConsoleURL); err != nil {
					return err
				}
				fmt.Println()
			}
			return nil
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 1, "返回失败运行条数, 默认 1(最后一次失败)")
	c.Flags().Int64Var(&tail, "tail", 10, "每个失败 step 日志尾部行数, 0 表示不拉日志")
	return c
}

// extractFailures 提取单次运行的失败信息: 先按 task 聚合查 TaskRuns, 再拉失败 step 日志尾部.
func extractFailures(client *cpclient.Client, ctx context.Context, ws, pipeId string, run *cp.ItemForListPipelineRunsOutput, tail int64) []failure.Failure {
	var allTaskRuns []*cp.ItemForListTaskRunsOutput
	for _, st := range run.Stages {
		for _, tk := range st.Tasks {
			if volcengine.StringValue(tk.Status) != "Failed" {
				continue
			}
			taskId := volcengine.StringValue(tk.Id)
			if taskId == "" {
				continue
			}
			trs, err := client.ListTaskRuns(ctx, ws, pipeId, volcengine.StringValue(run.Id), taskId)
			if err != nil {
				continue // 单任务查询失败不阻塞整体提取
			}
			allTaskRuns = append(allTaskRuns, trs.Items...)
		}
	}
	fs := failure.Extract(run, failure.BuildTaskRunMap(allTaskRuns))
	if tail <= 0 {
		return fs
	}
	// 为每个失败 step 拉日志尾部
	trByTask := failure.BuildTaskRunMap(allTaskRuns)
	for i := range fs {
		f := &fs[i]
		if f.Step == "" {
			continue
		}
		taskId := taskIdOf(run, f)
		if taskId == "" {
			continue
		}
		// 找 TaskRunId
		var taskRunId string
		if runs, ok := trByTask[taskId]; ok && len(runs) > 0 && runs[0].Id != nil {
			taskRunId = volcengine.StringValue(runs[0].Id)
		}
		if taskRunId == "" {
			continue
		}
		log, err := client.GetTaskRunLog(ctx, ws, pipeId, volcengine.StringValue(run.Id), taskId, taskRunId, f.Step)
		if err != nil {
			continue // 单 step 日志失败不阻塞
		}
		f.LogTail = failure.TailLogLines(log.LogLines, int(tail))
	}
	return fs
}

// taskIdOf 返回失败项对应的 task Id(按 Stage+Task 名匹配).
func taskIdOf(run *cp.ItemForListPipelineRunsOutput, f *failure.Failure) string {
	for _, st := range run.Stages {
		if volcengine.StringValue(st.Name) != f.Stage {
			continue
		}
		for _, tk := range st.Tasks {
			if volcengine.StringValue(tk.Name) == f.Task {
				return volcengine.StringValue(tk.Id)
			}
		}
	}
	return ""
}
