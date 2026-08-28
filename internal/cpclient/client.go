// Package cpclient 封装火山云持续交付(CodePipeline) SDK 访问.
// region 使用 SDK 默认 cn-north-1(该服务仅北京 region).
package cpclient

import (
	"fmt"

	cp "github.com/volcengine/volc-sdk-golang/service/codePipeline"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
)

// Client 持有已配置凭证的 codePipeline 实例.
type Client struct {
	svc *cp.CodePipeline
}

// New 创建 client. ak/sk 为空时依赖 SDK 从环境变量自动读取.
func New(ak, sk string) *Client {
	svc := cp.DefaultInstance
	if ak != "" {
		svc.Client.SetAccessKey(ak)
	}
	if sk != "" {
		svc.Client.SetSecretKey(sk)
	}
	return &Client{svc: svc}
}

// ListPipelines 列出工作区内全部流水线(SDK 该接口无分页参数).
func (c *Client) ListPipelines(workspaceId string) (*models.ListPipelinesResponse, error) {
	return c.svc.ListPipelines(&models.ListPipelinesRequest{WorkspaceId: workspaceId})
}

// ListPipelineRecords 按需倒序列出流水线执行记录, filterStatus 为空表示不过滤.
func (c *Client) ListPipelineRecords(workspaceId, pipelineId string, pageSize int64, filterStatus string) (*models.ListPipelineRecordsResponse, error) {
	req := &models.ListPipelineRecordsRequest{
		WorkspaceId: workspaceId,
		PipelineId:  pipelineId,
		Page:        models.Page{PageNumber: 1, PageSize: pageSize},
		Desc:        true,
	}
	if filterStatus != "" {
		req.Filter = &models.PipelineRecordFilter{Statuses: filterStatus}
	}
	return c.svc.ListPipelineRecords(req)
}

// GetPipelineRecord 查询单条执行记录详情(含 Stages/Tasks/Steps).
func (c *Client) GetPipelineRecord(workspaceId, pipelineId, recordId string) (*models.GetPipelineRecordResponse, error) {
	return c.svc.GetPipelineRecord(&models.GetPipelineRecordRequest{
		WorkspaceId: workspaceId,
		PipelineId:  pipelineId,
		Id:          recordId,
	})
}

// ConsoleURL 返回该执行记录的控制台页面地址.
func (c *Client) ConsoleURL(workspaceId, pipelineId, recordId string) string {
	return fmt.Sprintf("https://console.volcengine.com/cp/workspace/%s/pipeline/%s/record/%s",
		workspaceId, pipelineId, recordId)
}
