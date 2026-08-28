package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

type failureItem struct {
	Record     *models.PipelineRecord `json:"Record"`
	Failures   []failure.Failure      `json:"Failures"`
	ConsoleURL string                 `json:"ConsoleURL"`
}

// newFailuresCmd 组合命令: ListPipelineRecords(Filter=Failed) + GetPipelineRecord,
// 提取每条失败记录的 Stage→Task→Step 失败路径与错误信息, 附控制台 URL.
func newFailuresCmd() *cobra.Command {
	var pageSize int64
	c := &cobra.Command{
		Use:   "failures",
		Short: "最近 N 次失败记录及错误信息(默认 1 次)",
		Long: "组合命令: ListPipelineRecords(Filter=Failed) + GetPipelineRecord.\n" +
			"对每条失败记录提取 Stage→Task→Step 失败路径与错误信息, 附控制台 URL.\n" +
			"--page-size 1 时即'最后一次失败记录'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			client := newClient()
			p, err := findPipelineByName(client, ws, args[0])
			if err != nil {
				return err
			}
			list, err := client.ListPipelineRecords(ws, p.Id, pageSize, "Failed")
			if err != nil {
				return err
			}
			if len(list.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "无失败记录")
				return nil
			}
			items := make([]failureItem, 0, len(list.Items))
			for _, brief := range list.Items {
				detail, err := client.GetPipelineRecord(ws, p.Id, brief.Id)
				if err != nil {
					return fmt.Errorf("查询记录 %s 详情失败: %w", brief.Id, err)
				}
				var rec *models.PipelineRecord
				if detail != nil {
					rec = &detail.PipelineRecord
				}
				items = append(items, failureItem{
					Record:     rec,
					Failures:   failure.Extract(rec),
					ConsoleURL: client.ConsoleURL(ws, p.Id, brief.Id),
				})
			}
			if opts.Global.JSON {
				return output.PrintJSON(map[string]any{
					"PipelineId": p.Id, "PipelineName": p.Name, "Items": items,
				})
			}
			for _, it := range items {
				if err := output.PrintRecordDetail(it.Record, it.Failures, it.ConsoleURL); err != nil {
					return err
				}
				fmt.Println()
			}
			return nil
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 1, "返回失败记录条数, 默认 1(最后一次失败)")
	return c
}
