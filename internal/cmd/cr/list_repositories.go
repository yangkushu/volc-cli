package cr

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// repoWithCount --tag-count 模式下的 JSON 输出结构(嵌入 SDK 字段原样平铺, 追加 TagCount).
type repoWithCount struct {
	*cr.ItemForListRepositoriesOutput
	TagCount int64 `json:"TagCount"`
}

func newListRepositoriesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list-repositories",
		Short: "ListRepositories: 列出制品仓库",
		Long: "ListRepositories: 列出实例下全部 OCI 制品仓库(自动翻页拉全).\n" +
			"--namespace 过滤指定命名空间(Filter.Namespaces, 可逗号分隔多个); 省略时列出全部.\n" +
			"--tag-count 追加 TAG_COUNT 列(JSON 输出追加 TagCount 字段), 用于配额水位排查;\n" +
			"会对每个仓库额外调用一次计数接口, 仓库多时较慢. 小微版实例单仓库 tag 上限 100.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			registry, err := resolveRegistry(client, ctx)
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
			items, err := client.ListRepositoriesAll(ctx, registry, namespaces)
			if err != nil {
				return err
			}
			if !tagCountFlag {
				if opts.Global.JSON {
					return output.PrintJSON(items)
				}
				return output.PrintRepositories(items, nil)
			}
			counts := make(map[string]int64, len(items))
			for _, r := range items {
				n, err := client.CountTags(ctx, registry, volcengine.StringValue(r.Namespace), volcengine.StringValue(r.Name))
				if err != nil {
					return err
				}
				counts[output.RepoKey(r)] = n
			}
			if opts.Global.JSON {
				out := make([]repoWithCount, 0, len(items))
				for _, r := range items {
					out = append(out, repoWithCount{ItemForListRepositoriesOutput: r, TagCount: counts[output.RepoKey(r)]})
				}
				return output.PrintJSON(out)
			}
			return output.PrintRepositories(items, counts)
		},
	}
	c.Flags().StringVar(&namespacesFlag, "namespace", "", "按命名空间过滤, 逗号分隔多个")
	c.Flags().BoolVar(&tagCountFlag, "tag-count", false, "追加每仓库 tag 总数列(逐仓库调计数接口, 较慢)")
	return c
}

// namespacesFlag/tagCountFlag 由 list-repositories 的 flag 绑定(包级变量, 各命令文件独立持有自己的 flag 变量).
var (
	namespacesFlag string
	tagCountFlag   bool
)
