// Package crclient 封装火山云镜像仓库(CR) OpenAPI 访问.
// 基于官方 SDK github.com/volcengine/volcengine-go-sdk/service/cr(API 版本 2022-05-12).
// CR 实例是区域性资源, region 由调用方传入.
package crclient

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

const (
	listPageSize = 100  // List* 接口 PageSize 上限
	maxPages     = 1000 // 翻页防御上限(10 万条), 避免异常响应导致死循环
)

// Client 持有已配置凭证的 cr 服务实例.
type Client struct {
	svc *cr.CR
}

// New 创建 client, ak/sk 为空时依赖 SDK 默认凭证链(环境变量等).
// 凭证优先级与 CLI 全局约定一致: flag > 环境变量.
func New(ak, sk, region string) (*Client, error) {
	cfg := volcengine.NewConfig().WithRegion(region)
	if ak != "" || sk != "" {
		cfg = cfg.WithAkSk(ak, sk)
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 session 失败: %w", err)
	}
	return &Client{svc: cr.New(sess, cfg)}, nil
}

// ListRegistriesAll 列出当前 region 全部镜像仓库实例(自动翻页拉全).
func (c *Client) ListRegistriesAll(ctx context.Context) ([]*cr.ItemForListRegistriesOutput, error) {
	var items []*cr.ItemForListRegistriesOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListRegistriesWithContext(ctx, &cr.ListRegistriesInput{
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListRegistries 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListNamespacesAll 列出实例下全部命名空间(自动翻页拉全).
func (c *Client) ListNamespacesAll(ctx context.Context, registry string) ([]*cr.ItemForListNamespacesOutput, error) {
	var items []*cr.ItemForListNamespacesOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListNamespacesWithContext(ctx, &cr.ListNamespacesInput{
			Registry:   &registry,
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListNamespaces 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListRepositoriesAll 列出实例全部制品仓库; namespaces 非空时按命名空间过滤(自动翻页拉全).
func (c *Client) ListRepositoriesAll(ctx context.Context, registry string, namespaces []string) ([]*cr.ItemForListRepositoriesOutput, error) {
	var items []*cr.ItemForListRepositoriesOutput
	in := &cr.ListRepositoriesInput{Registry: &registry}
	if len(namespaces) > 0 {
		nsPtrs := make([]*string, 0, len(namespaces))
		for _, ns := range namespaces {
			ns := ns
			nsPtrs = append(nsPtrs, &ns)
		}
		in.Filter = &cr.FilterForListRepositoriesInput{Namespaces: nsPtrs}
	}
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		in.PageNumber = &page
		in.PageSize = &size
		resp, err := c.svc.ListRepositoriesWithContext(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("ListRepositories 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListTagsAll 列出某制品仓库全部版本 tag(自动翻页拉全, 不带服务端 Filter, 过滤由客户端做).
func (c *Client) ListTagsAll(ctx context.Context, registry, namespace, repository string) ([]*cr.ItemForListTagsOutput, error) {
	var items []*cr.ItemForListTagsOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListTagsWithContext(ctx, &cr.ListTagsInput{
			Registry:   &registry,
			Namespace:  &namespace,
			Repository: &repository,
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListTags(%s/%s) 第 %d 页失败: %w", namespace, repository, page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// DeleteTags 删除指定版本(单批调用, 调用方负责按 20/批切分, 官方单次上限 20 个).
func (c *Client) DeleteTags(ctx context.Context, registry, namespace, repository string, names []string) (*cr.DeleteTagsOutput, error) {
	ptrs := make([]*string, 0, len(names))
	for _, n := range names {
		n := n
		ptrs = append(ptrs, &n)
	}
	return c.svc.DeleteTagsWithContext(ctx, &cr.DeleteTagsInput{
		Registry:   &registry,
		Namespace:  &namespace,
		Repository: &repository,
		Names:      ptrs,
	})
}
