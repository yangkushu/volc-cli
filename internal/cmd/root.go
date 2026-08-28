package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Opts 全局选项, 子命令通过命令链上的 flag 读取.
type Opts struct {
	AccessKey   string
	SecretKey   string
	WorkspaceId string
	JSON        bool
}

// globalOpts 由 root 命令的 PersistentFlags 解析填充, 各子命令只读.
var globalOpts Opts

const (
	envAK = "VOLC_ACCESSKEY"
	envSK = "VOLC_SECRETKEY"
	envWs = "VOLC_CP_WORKSPACE_ID"
)

// NewRootCommand 创建根命令.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "volc-cli",
		Short: "火山云 CLI 工具",
		Long: "volc-cli 是火山云(Volcengine)服务的命令行工具.\n" +
			"命令结构: volc-cli <模块> <命令>, 模块名与官方 SDK service 包名对齐.\n" +
			"当前支持模块: codepipeline(持续交付).",
		SilenceUsage:  true,
		SilenceErrors: true,
		// 无参运行时打印帮助(含 flags), 也使根命令成为 Runnable 以输出 usage.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	pf := root.PersistentFlags()
	pf.StringVar(&globalOpts.AccessKey, "ak", "", "访问密钥 ID, 默认读环境变量 "+envAK)
	pf.StringVar(&globalOpts.SecretKey, "sk", "", "访问密钥 Secret, 默认读环境变量 "+envSK)
	pf.StringVar(&globalOpts.WorkspaceId, "workspace-id", "", "持续交付工作区 ID, 默认读环境变量 "+envWs)
	pf.BoolVar(&globalOpts.JSON, "json", false, "以 JSON 格式输出(字段名与官方 SDK 一致)")
	root.AddCommand(NewCheckCredentialsCmd())
	return root
}

// resolveWorkspaceId 返回生效的 workspace id, 未配置时报错.
func resolveWorkspaceId() (string, error) {
	if globalOpts.WorkspaceId != "" {
		return globalOpts.WorkspaceId, nil
	}
	return "", fmt.Errorf("workspace-id 未设置: 请用 --workspace-id 指定或导出环境变量 %s(可从控制台流水线 URL /cp/workspace/{id}/pipeline/... 中获取)", envWs)
}
