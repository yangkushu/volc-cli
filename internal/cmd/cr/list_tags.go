package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/crtag"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// repoTags --json 多 repo 聚合输出结构(字段名保持 SDK 风格).
type repoTags struct {
	Repository string                      `json:"Repository"`
	Items      []*cr.ItemForListTagsOutput `json:"Items"`
}

var (
	ltRepository    string
	ltNamespaceFlag string
	ltTagNames      string
	ltTagPrefix     string
	ltOlderThan     string
	ltKeepLast      int
	ltOldest        int
)

func newListTagsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list-tags",
		Short: "ListTags: 列出制品仓库全部版本",
		Long: "ListTags: 列出制品仓库全部版本 tag(自动翻页拉全).\n" +
			"--namespace 必填; --repository 可省略, 省略时遍历该命名空间下全部制品仓库(只读聚合).\n\n" +
			"以下为清理候选过滤参数(客户端语义, 非 API 字段, 多条件交集):\n" +
			"  --older-than 30d   PushTime 早于 N 天前(支持 Nd/Nh)\n" +
			"  --keep-last 10      按 PushTime 降序保留最近 N 个, 其余为候选\n" +
			"  --oldest 20         按 PushTime 升序取最早 N 个为候选(与 --keep-last 对称)\n" +
			"  --tag-prefix ci-    tag 名前缀匹配\n" +
			"  --tag-names t1,t2   tag 名精确匹配\n" +
			"带任一过滤参数时输出 REASON 列; PushTime 无法解析的 tag 永不进入候选.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ltNamespaceFlag == "" {
				return fmt.Errorf("--namespace 必填")
			}
			criteria := crtag.Criteria{TagPrefix: ltTagPrefix, KeepLast: ltKeepLast, Oldest: ltOldest}
			if ltTagNames != "" {
				for _, n := range strings.Split(ltTagNames, ",") {
					if n = strings.TrimSpace(n); n != "" {
						criteria.TagNames = append(criteria.TagNames, n)
					}
				}
			}
			if ltOlderThan != "" {
				d, err := crtag.ParseOlderThan(ltOlderThan)
				if err != nil {
					return err
				}
				criteria.OlderThan = d
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			registry, err := resolveRegistry(client, ctx)
			if err != nil {
				return err
			}
			ns := ltNamespaceFlag
			repos := []string{ltRepository}
			if ltRepository == "" {
				items, err := client.ListRepositoriesAll(ctx, registry, []string{ns})
				if err != nil {
					return err
				}
				repos = repos[:0]
				for _, r := range items {
					repos = append(repos, volcengine.StringValue(r.Name))
				}
			}
			multiRepo := len(repos) > 1
			var allViews []crtag.TagView
			var jsonOut []repoTags
			for _, repo := range repos {
				tags, err := client.ListTagsAll(ctx, registry, ns, repo)
				if err != nil {
					return err
				}
				views := crtag.Filter(tags, criteria)
				for i := range views {
					views[i].Repository = repo
				}
				allViews = append(allViews, views...)
				if opts.Global.JSON {
					items := make([]*cr.ItemForListTagsOutput, 0, len(views))
					for _, v := range views {
						items = append(items, v.Tag)
					}
					jsonOut = append(jsonOut, repoTags{Repository: repo, Items: items})
				}
			}
			if opts.Global.JSON {
				if !multiRepo {
					// 单 repo 输出 SDK 结构原样字段数组
					items := []*cr.ItemForListTagsOutput{}
					if len(jsonOut) == 1 {
						items = jsonOut[0].Items
					}
					return output.PrintJSON(items)
				}
				return output.PrintJSON(jsonOut)
			}
			withReason := criteria.OlderThan > 0 || criteria.KeepLast > 0 || criteria.Oldest > 0 || criteria.TagPrefix != "" || len(criteria.TagNames) > 0
			return output.PrintTags(allViews, multiRepo, withReason)
		},
	}
	f := c.Flags()
	f.StringVar(&ltRepository, "repository", "", "制品仓库名; 省略时遍历命名空间下全部仓库")
	f.StringVar(&ltNamespaceFlag, "namespace", "", "命名空间(必填)")
	f.StringVar(&ltTagNames, "tag-names", "", "tag 名精确匹配, 逗号分隔")
	f.StringVar(&ltTagPrefix, "tag-prefix", "", "tag 名前缀匹配")
	f.StringVar(&ltOlderThan, "older-than", "", "PushTime 早于该时长(如 30d/12h), 客户端过滤")
	f.IntVar(&ltKeepLast, "keep-last", 0, "每个仓库按 PushTime 降序保留最近 N 个, 其余为候选")
	f.IntVar(&ltOldest, "oldest", 0, "按 PushTime 升序取最早 N 个为候选(与 --keep-last 对称)")
	return c
}
