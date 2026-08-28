// Package codepipeline 提供 volc-cli codepipeline 子命令组, 对应 SDK service/codePipeline.
package codepipeline

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/cpclient"
	"kuopin/volc-cli/internal/opts"
)

// NewCodePipelineCmd 创建 codepipeline 模块父命令.
func NewCodePipelineCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "codepipeline",
		Short: "持续交付(CodePipeline)流水线操作",
		Long: "对应官方 SDK service/codePipeline, 命令名与 OpenAPI Action 对齐:\n" +
			"  list-pipelines       ListPipelines       列出工作区全部流水线\n" +
			"  list-pipeline-records ListPipelineRecords  列出流水线执行记录(含发布参数)\n" +
			"  get-pipeline-record   GetPipelineRecord    查询单条执行记录详情\n" +
			"  failures              便捷命令            最近失败记录与错误信息",
	}
	c.AddCommand(newListPipelinesCmd(), newListPipelineRecordsCmd(), newGetPipelineRecordCmd(), newFailuresCmd())
	return c
}

// resolveWs 解析 workspace id, 未配置时报错.
func resolveWs() (string, error) {
	return config.ResolveWorkspaceId(opts.Global.WorkspaceId)
}

// newClient 用全局凭证构造 cpclient.
func newClient() *cpclient.Client {
	ak, sk := config.ResolveCredentials(opts.Global.AccessKey, opts.Global.SecretKey)
	return cpclient.New(ak, sk)
}

// findPipelineByName 在工作区内按名称精确或 ID 匹配流水线, 未命中时报错并列出可用项.
func findPipelineByName(c *cpclient.Client, ws, nameOrId string) (*models.Pipeline, error) {
	resp, err := c.ListPipelines(ws)
	if err != nil {
		return nil, fmt.Errorf("ListPipelines 失败: %w", err)
	}
	for i := range resp.Items {
		p := resp.Items[i]
		if p.Id == nameOrId || p.Name == nameOrId {
			return &p, nil
		}
	}
	names := make([]string, 0, len(resp.Items))
	limit := 20
	for i, p := range resp.Items {
		if i >= limit {
			break
		}
		names = append(names, p.Name)
	}
	return nil, fmt.Errorf("未找到流水线 %q, 可用流水线(前 %d): %s", nameOrId, limit, strings.Join(names, ", "))
}
