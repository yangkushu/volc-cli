package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListPipelineRecordsCmd() *cobra.Command {
	var pageSize int64
	var status string
	c := &cobra.Command{
		Use:   "list-pipeline-records",
		Short: "ListPipelineRecords: 列出流水线执行记录(含发布参数)",
		Long: "按名称或 ID 定位流水线(需先 ListPipelines), 倒序列出执行记录.\n" +
			"记录含 TriggerMode(触发方式)与 DynamicEnvs(每次发布的参数).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			c := newClient()
			p, err := findPipelineByName(c, ws, args[0])
			if err != nil {
				return err
			}
			resp, err := c.ListPipelineRecords(ws, p.Id, pageSize, status)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintRecords(resp.Items)
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 10, "返回记录条数, 默认 10")
	c.Flags().StringVar(&status, "status", "", "按状态过滤, 如 Failed/Success, 默认不过滤")
	return c
}
