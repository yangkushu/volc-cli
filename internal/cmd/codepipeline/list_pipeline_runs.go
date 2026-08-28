package codepipeline

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListPipelineRunsCmd() *cobra.Command {
	var pageSize int64
	var status string
	c := &cobra.Command{
		Use:   "list-pipeline-runs",
		Short: "ListPipelineRuns: 列出流水线执行记录(含发布参数)",
		Long: "ListPipelineRuns: 按名称或 ID 定位流水线(需先 ListPipelines), 列出执行记录.\n" +
			"记录含 Status(运行状态)与 Parameters(每次发布的参数, 秘密参数脱敏显示).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			p, err := findPipelineByName(client, context.Background(), ws, args[0])
			if err != nil {
				return err
			}
			resp, err := client.ListPipelineRuns(context.Background(), ws, ptrValue(p.Id), pageSize, status)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintRuns(resp.Items)
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 10, "返回记录条数, 默认 10")
	c.Flags().StringVar(&status, "status", "", "按状态过滤, 如 Failed/Succeeded, 默认不过滤")
	return c
}

// ptrValue 返回指针的值, nil 时返回空串.
func ptrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
