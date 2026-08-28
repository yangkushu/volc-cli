// Package cpclient 封装火山云持续交付(CodePipeline) V2 API 访问.
// 基于官方新 SDK github.com/volcengine/volcengine-go-sdk/service/cp(API 版本 2023-05-01).
// region 固定 cn-north-1(该服务仅北京 region).
package cpclient

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

// Client 持有已配置凭证的 cp 服务实例.
type Client struct {
	svc *cp.CP
}

// New 创建 client, ak/sk 为空时依赖 SDK 默认凭证链(环境变量等).
// 凭证优先级与 CLI 全局约定一致: flag > 环境变量.
func New(ak, sk string) (*Client, error) {
	cfg := volcengine.NewConfig().WithRegion("cn-north-1")
	if ak != "" || sk != "" {
		cfg = cfg.WithAkSk(ak, sk)
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 session 失败: %w", err)
	}
	return &Client{svc: cp.New(sess, cfg)}, nil
}

// ListWorkspaces 列出账户下全部工作区(无需 WorkspaceId, 可作凭证自举).
func (c *Client) ListWorkspaces(ctx context.Context, pageNumber, pageSize int64) (*cp.ListWorkspacesOutput, error) {
	return c.svc.ListWorkspacesWithContext(ctx, &cp.ListWorkspacesInput{
		PageNumber: &pageNumber,
		PageSize:   &pageSize,
	})
}

// ListPipelines 列出工作区内全部流水线.
func (c *Client) ListPipelines(ctx context.Context, workspaceId string) (*cp.ListPipelinesOutput, error) {
	return c.svc.ListPipelinesWithContext(ctx, &cp.ListPipelinesInput{
		WorkspaceId: &workspaceId,
	})
}

// ListPipelineRuns 列出流水线执行记录(按需过滤状态, 倒序).
func (c *Client) ListPipelineRuns(ctx context.Context, workspaceId, pipelineId string, maxResults int64, status string) (*cp.ListPipelineRunsOutput, error) {
	in := &cp.ListPipelineRunsInput{
		WorkspaceId: &workspaceId,
		PipelineId:  &pipelineId,
		MaxResults:  &maxResults,
	}
	if status != "" {
		in.Filter = &cp.FilterForListPipelineRunsInput{
			Statuses: []*string{&status},
		}
	}
	return c.svc.ListPipelineRunsWithContext(ctx, in)
}

// ListTaskRuns 列出单次运行中某个任务的执行步骤(含 step 级状态与耗时).
func (c *Client) ListTaskRuns(ctx context.Context, workspaceId, pipelineId, runId, taskId string) (*cp.ListTaskRunsOutput, error) {
	return c.svc.ListTaskRunsWithContext(ctx, &cp.ListTaskRunsInput{
		WorkspaceId:   &workspaceId,
		PipelineId:    &pipelineId,
		PipelineRunId: &runId,
		TaskId:        &taskId,
	})
}

// GetTaskRunLog 拉取某 step 的全量日志行(调用方本地截尾, 避免服务端分页语义差异).
func (c *Client) GetTaskRunLog(ctx context.Context, workspaceId, pipelineId, runId, taskId, taskRunId, stepName string) (*cp.GetTaskRunLogOutput, error) {
	return c.svc.GetTaskRunLogWithContext(ctx, &cp.GetTaskRunLogInput{
		WorkspaceId:   &workspaceId,
		PipelineId:    &pipelineId,
		PipelineRunId: &runId,
		TaskId:        &taskId,
		TaskRunId:     &taskRunId,
		StepName:      &stepName,
	})
}

// ConsoleURL 返回该执行记录的控制台页面地址(V2 控制台).
func (c *Client) ConsoleURL(workspaceId, pipelineId, runId string) string {
	return fmt.Sprintf("https://console.volcengine.com/cp/v2/workspace/%s/pipeline/%s/run/%s",
		workspaceId, pipelineId, runId)
}
