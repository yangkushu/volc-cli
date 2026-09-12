package cr

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListRegistriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-registries",
		Short: "ListRegistries: 列出当前 region 全部镜像仓库实例",
		Long:  "ListRegistries: 列出当前 region 全部镜像仓库实例(自动翻页拉全).\n实例是区域性资源, 无实例时请检查 --region 是否正确.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			items, err := client.ListRegistriesAll(context.Background())
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintRegistries(items)
		},
	}
}
