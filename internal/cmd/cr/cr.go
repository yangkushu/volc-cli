// Package cr 提供 volc-cli cr 子命令组, 对应火山云镜像仓库 OpenAPI(API 版本 2022-05-12).
package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/crclient"
	"kuopin/volc-cli/internal/opts"
)

// NewCRCmd 创建 cr 模块父命令.
func NewCRCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cr",
		Short: "镜像仓库(Container Registry)操作",
		Long: "对应镜像仓库 OpenAPI(API 版本 2022-05-12), 命令名与 Action 对齐:\n" +
			"  list-registries     ListRegistries   列出当前 region 全部实例\n" +
			"  list-namespaces     ListNamespaces   列出实例下全部命名空间\n" +
			"  list-repositories   ListRepositories 列出命名空间下全部 OCI 制品仓库\n" +
			"  list-tags           ListTags         列出制品仓库全部版本, 可按清理候选过滤(客户端语义)\n" +
			"  delete-tags         DeleteTags       删除指定版本, 需显式 --tags 列表与 --yes 二次确认\n\n" +
			"删除是不可恢复的线上修改: Agent 调用前必须向用户说明目标与影响并获得明确确认.",
	}
	pf := c.PersistentFlags()
	pf.StringVar(&opts.Global.Region, "region", "", "实例所在 region, 默认读环境变量 VOLC_CR_REGION 或 cn-beijing")
	pf.StringVar(&opts.Global.Registry, "registry", "", "镜像仓库实例名, 未指定时自动解析(唯一实例直接用, 多实例报错列出候选)")
	c.AddCommand(newListRegistriesCmd(), newListNamespacesCmd(), newListRepositoriesCmd(),
		newListTagsCmd(), newDeleteTagsCmd())
	return c
}

// newClient 用全局凭证与生效 region 构造 crclient, 失败时报错.
func newClient() (*crclient.Client, error) {
	ak, sk := config.ResolveCredentials(opts.Global.AccessKey, opts.Global.SecretKey)
	return crclient.New(ak, sk, config.ResolveRegion(opts.Global.Region))
}

// resolveRegistry 解析实例名: --registry 优先, 未配置时自动列出(唯一实例直接用).
func resolveRegistry(client *crclient.Client, ctx context.Context) (string, error) {
	if opts.Global.Registry != "" {
		return opts.Global.Registry, nil
	}
	items, err := client.ListRegistriesAll(ctx)
	if err != nil {
		return "", fmt.Errorf("registry 未配置且自动获取失败: %w(请用 --registry 指定或先跑 list-registries 查看)", err)
	}
	switch len(items) {
	case 0:
		return "", fmt.Errorf("当前 region(%s)下无镜像仓库实例, 请检查 --region", config.ResolveRegion(opts.Global.Region))
	case 1:
		if items[0].Name == nil {
			return "", fmt.Errorf("实例名为空, 请用 --registry 指定")
		}
		return *items[0].Name, nil
	default:
		names := make([]string, 0, len(items))
		for _, r := range items {
			names = append(names, volcengine.StringValue(r.Name))
		}
		return "", fmt.Errorf("当前 region 下有多个实例: %s, 请用 --registry 指定", strings.Join(names, ", "))
	}
}
