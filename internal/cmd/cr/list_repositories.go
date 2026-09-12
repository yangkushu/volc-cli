package cr

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListRepositoriesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list-repositories",
		Short: "ListRepositories: 列出制品仓库",
		Long: "ListRepositories: 列出实例下全部 OCI 制品仓库(自动翻页拉全).\n" +
			"--namespace 过滤指定命名空间(Filter.Namespaces, 可逗号分隔多个); 省略时列出全部.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			registry, err := resolveRegistry(client, context.Background())
			if err != nil {
				return err
			}
			var namespaces []string
			if namespacesFlag != "" {
				for _, n := range strings.Split(namespacesFlag, ",") {
					if n = strings.TrimSpace(n); n != "" {
						namespaces = append(namespaces, n)
					}
				}
			}
			items, err := client.ListRepositoriesAll(context.Background(), registry, namespaces)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintRepositories(items)
		},
	}
	c.Flags().StringVar(&namespacesFlag, "namespace", "", "按命名空间过滤, 逗号分隔多个")
	return c
}

// namespacesFlag 由 list-repositories 的 --namespace flag 绑定(包级变量, 各命令文件独立持有自己的 flag 变量).
var namespacesFlag string
