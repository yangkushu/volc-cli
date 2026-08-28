package codepipeline

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListWorkspacesCmd() *cobra.Command {
	var pageNumber, pageSize int64
	c := &cobra.Command{
		Use:   "list-workspaces",
		Short: "ListWorkspaces: 列出账户下全部工作区",
		Long: "ListWorkspaces: 列出账户下全部工作区(无需 WorkspaceId).\n" +
			"用于获取 workspace-id: 从输出 ID 列取值, 配置到 --workspace-id 或环境变量 VOLC_CP_WORKSPACE_ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			resp, err := client.ListWorkspaces(context.Background(), pageNumber, pageSize)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintWorkspaces(resp.Items)
		},
	}
	c.Flags().Int64Var(&pageNumber, "page-number", 1, "页码, 默认 1")
	c.Flags().Int64Var(&pageSize, "page-size", 20, "每页条数, 默认 20")
	return c
}
