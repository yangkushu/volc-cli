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
			"各命令已支持自动解析 workspace; 本命令用于手动查看或确认多个工作区时用哪个.",
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
