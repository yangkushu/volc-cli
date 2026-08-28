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
		Long:  "ListPipelines: 列出工作区全部流水线.\n需配置 --workspace-id 或环境变量 VOLC_CP_WORKSPACE_ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			client, err := newClient()
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
