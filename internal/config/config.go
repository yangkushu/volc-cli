// Package config 解析 CLI 凭证与工作区配置, flag 优先, 环境变量兜底.
package config

import (
	"fmt"
	"os"
)

const (
	EnvAccessKey   = "VOLC_ACCESS_KEY"
	EnvSecretKey   = "VOLC_SECRET_KEY"
	EnvWorkspaceId = "VOLC_CP_WORKSPACE_ID"
	EnvCRRegion    = "VOLC_CR_REGION"
)

// ResolveCredentials 返回生效的 AK/SK: flag 非空优先, 否则读环境变量.
func ResolveCredentials(akFlag, skFlag string) (ak, sk string) {
	if akFlag != "" {
		ak = akFlag
	} else {
		ak = os.Getenv(EnvAccessKey)
	}
	if skFlag != "" {
		sk = skFlag
	} else {
		sk = os.Getenv(EnvSecretKey)
	}
	return ak, sk
}

// ResolveWorkspaceId 返回生效的 workspace id, 未设置时返回错误.
func ResolveWorkspaceId(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if v := os.Getenv(EnvWorkspaceId); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("workspace-id 未设置: 请用 --workspace-id 指定或导出环境变量 %s(可从控制台流水线 URL /cp/workspace/{id}/pipeline/... 中获取)", EnvWorkspaceId)
}

// MaskSecret 返回密钥脱敏描述: 空值返回"未设置"; 非空只暴露长度, 不暴露任何内容(含前缀).
func MaskSecret(s string) string {
	if s == "" {
		return "未设置"
	}
	return fmt.Sprintf("*** (len=%d)", len(s))
}

// DefaultCRRegion cr 模块默认 region.
const DefaultCRRegion = "cn-beijing"

// ResolveRegion 返回 cr 模块生效的 region: flag 非空优先, 其次环境变量, 默认 cn-beijing.
// 注意 CR 服务不接受 cn-north-1 别名(2026-09-14 实测 InvalidRegion), 必须用新命名.
func ResolveRegion(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(EnvCRRegion); v != "" {
		return v
	}
	return DefaultCRRegion
}
