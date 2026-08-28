// Package codepipeline 提供 volc-cli codepipeline 子命令组, 对应火山云持续交付 V2 API.
package codepipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/cpclient"
	"kuopin/volc-cli/internal/opts"
)

// NewCodePipelineCmd 创建 codepipeline 模块父命令.
func NewCodePipelineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "codepipeline",
		Short: "持续交付(CodePipeline)流水线操作",
		Long: "对应持续交付 V2 OpenAPI(API 版本 2023-05-01), 命令名与 Action 对齐:\n" +
			"  list-workspaces        ListWorkspaces    列出账户全部工作区(获取 workspace-id)\n" +
			"  list-pipelines         ListPipelines     列出工作区全部流水线\n" +
			"  list-pipeline-runs     ListPipelineRuns  列出流水线执行记录(含发布参数)\n" +
			"  get-task-run-log       GetTaskRunLog     查询失败 step 的日志\n" +
			"  failures               便捷命令          最近失败运行与错误日志",
	}
	c.AddCommand(newListWorkspacesCmd(), newListPipelinesCmd(), newListPipelineRunsCmd(), newGetTaskRunLogCmd(), newFailuresCmd())
	return c
}

// resolveWs 解析 workspace id, 未配置时报错.
func resolveWs() (string, error) {
	return config.ResolveWorkspaceId(opts.Global.WorkspaceId)
}

// newClient 用全局凭证构造 cpclient, 失败时报错.
func newClient() (*cpclient.Client, error) {
	ak, sk := config.ResolveCredentials(opts.Global.AccessKey, opts.Global.SecretKey)
	return cpclient.New(ak, sk)
}

// findPipelineByName 在工作区内按名称精确或 ID 匹配流水线, 未命中时报错并列出可用项.
func findPipelineByName(c *cpclient.Client, ctx context.Context, ws, nameOrId string) (*cp.ItemForListPipelinesOutput, error) {
	resp, err := c.ListPipelines(ctx, ws)
	if err != nil {
		return nil, fmt.Errorf("ListPipelines 失败: %w", err)
	}
	for _, p := range resp.Items {
		if volcengine.StringValue(p.Id) == nameOrId || volcengine.StringValue(p.Name) == nameOrId {
			return p, nil
		}
	}
	names := make([]string, 0, len(resp.Items))
	limit := 20
	for i, p := range resp.Items {
		if i >= limit {
			break
		}
		names = append(names, volcengine.StringValue(p.Name))
	}
	return nil, fmt.Errorf("未找到流水线 %q, 可用流水线(前 %d): %s", nameOrId, limit, strings.Join(names, ", "))
}
