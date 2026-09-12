package cr

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListNamespacesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-namespaces",
		Short: "ListNamespaces: 列出实例下全部命名空间",
		Long:  "ListNamespaces: 列出实例下全部命名空间(自动翻页拉全).\nRegistry 未配置时自动解析(唯一实例直接用, 多实例用 --registry 指定).",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			registry, err := resolveRegistry(client, context.Background())
			if err != nil {
				return err
			}
			items, err := client.ListNamespacesAll(context.Background(), registry)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintNamespaces(items)
		},
	}
}
