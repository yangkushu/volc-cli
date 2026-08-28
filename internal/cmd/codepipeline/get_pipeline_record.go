package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newGetPipelineRecordCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "get-pipeline-record",
		Short: "GetPipelineRecord: 查询单条执行记录详情(含失败信息)",
		Long: "定位流水线后查询指定 record 详情, 展示 Stage/Task/Step 三层结构,\n" +
			"失败步骤展开错误信息, 并附控制台 URL.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			recordId, _ := cmd.Flags().GetString("id")
			if recordId == "" {
				return fmt.Errorf("必须用 --id 指定记录 ID(可先用 list-pipeline-records 查看)")
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
			resp, err := c.GetPipelineRecord(ws, p.Id, recordId)
			if err != nil {
				return err
			}
			var rec *models.PipelineRecord
			if resp != nil {
				rec = &resp.PipelineRecord
			}
			fs := failure.Extract(rec)
			if opts.Global.JSON {
				return output.PrintJSON(map[string]any{
					"Record":     rec,
					"Failures":   fs,
					"ConsoleURL": c.ConsoleURL(ws, p.Id, recordId),
				})
			}
			return output.PrintRecordDetail(rec, fs, c.ConsoleURL(ws, p.Id, recordId))
		},
	}
	c.Flags().String("id", "", "执行记录 ID(必填)")
	return c
}
