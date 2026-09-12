package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// deleteTagBatch 官方 DeleteTags 单次请求上限.
const deleteTagBatch = 20

var (
	dtNamespace  string
	dtRepository string
	dtTags       string
	dtYes        bool
)

func newDeleteTagsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete-tags",
		Short: "DeleteTags: 删除制品仓库中指定版本",
		Long: "DeleteTags: 删除制品仓库中指定的版本 tag, 不可恢复.\n" +
			"必须显式指定 --tags(逗号分隔的精确列表)并加 --yes 二次确认; 不做任何自动圈定.\n" +
			"超过 20 个自动分批调用; 请求级失败即止并报告进度, 批内逐条失败继续, 存在失败时以非零码退出.\n" +
			"Agent 调用前必须先向用户展示删除清单并获得明确确认.",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := parseTagList(dtTags)
			if len(names) == 0 {
				return fmt.Errorf("--tags 不能为空: 请用逗号分隔显式列出要删除的版本名")
			}
			if dtNamespace == "" || dtRepository == "" {
				return fmt.Errorf("--namespace 与 --repository 必填")
			}
			if !dtYes {
				return fmt.Errorf("删除是不可恢复操作, 请核对后加 --yes 重试:\n"+
					"  tags: %s", strings.Join(names, ","))
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
			var deleted, remaining []string
			var failures []*cr.FailureForDeleteTagsOutput
			for i := 0; i < len(names); i += deleteTagBatch {
				end := i + deleteTagBatch
				if end > len(names) {
					end = len(names)
				}
				batch := names[i:end]
				resp, err := client.DeleteTags(ctx, registry, dtNamespace, dtRepository, batch)
				if err != nil {
					// 失败批次本身未删除, 未执行清单从本批起算
					remaining = names[i:]
					_ = output.PrintDeleteTagsResult(deleted, remaining, failures)
					return fmt.Errorf("第 %d 批(共 %d 批)请求失败: %w", i/deleteTagBatch+1,
						(len(names)+deleteTagBatch-1)/deleteTagBatch, err)
				}
				for _, s := range resp.Successes {
					deleted = append(deleted, volcengine.StringValue(s.Name))
				}
				failures = append(failures, resp.Failures...)
			}
			if opts.Global.JSON {
				return output.PrintJSON(map[string]any{
					"Registry": registry, "Namespace": dtNamespace, "Repository": dtRepository,
					"Deleted": deleted, "Failures": failures, "Remaining": remaining,
				})
			}
			if err := output.PrintDeleteTagsResult(deleted, remaining, failures); err != nil {
				return err
			}
			if len(failures) > 0 {
				return fmt.Errorf("%d 个版本删除失败(见上方失败清单)", len(failures))
			}
			return nil
		},
	}
	f := c.Flags()
	f.StringVar(&dtNamespace, "namespace", "", "命名空间(必填)")
	f.StringVar(&dtRepository, "repository", "", "制品仓库名(必填)")
	f.StringVar(&dtTags, "tags", "", "要删除的版本名, 逗号分隔显式列表(必填)")
	f.BoolVar(&dtYes, "yes", false, "确认执行删除(不可恢复)")
	return c
}

// parseTagList 解析逗号分隔的 tag 名: 去空白/去空项/保序去重.
func parseTagList(s string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range strings.Split(s, ",") {
		if n = strings.TrimSpace(n); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
