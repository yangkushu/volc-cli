package codepipeline

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListPipelinesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-pipelines",
		Short: "ListPipelines: 列出工作区全部流水线",
		Long:  "ListPipelines: 列出工作区全部流水线.\nWorkspace 未配置时自动解析(唯一工作区直接用, 多个工作区用 --workspace-id 指定).",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			ws, err := resolveWs(client, context.Background())
			if err != nil {
				return err
			}
			resp, err := client.ListPipelines(context.Background(), ws)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintPipelines(resp.Items)
		},
	}
}
