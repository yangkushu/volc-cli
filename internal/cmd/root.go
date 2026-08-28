package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"kuopin/volc-cli/internal/cmd/codepipeline"
	"kuopin/volc-cli/internal/opts"
)

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
	pf.StringVar(&opts.Global.AccessKey, "ak", "", "访问密钥 ID, 默认读环境变量 "+envAK)
	pf.StringVar(&opts.Global.SecretKey, "sk", "", "访问密钥 Secret, 默认读环境变量 "+envSK)
	pf.StringVar(&opts.Global.WorkspaceId, "workspace-id", "", "持续交付工作区 ID, 默认读环境变量 "+envWs)
	pf.BoolVar(&opts.Global.JSON, "json", false, "以 JSON 格式输出(字段名与官方 SDK 一致)")
	root.AddCommand(NewCheckCredentialsCmd())
	root.AddCommand(codepipeline.NewCodePipelineCmd())
	return root
}

// resolveWorkspaceId 返回生效的 workspace id, 未配置时报错.
func resolveWorkspaceId() (string, error) {
	if opts.Global.WorkspaceId != "" {
		return opts.Global.WorkspaceId, nil
	}
	return "", fmt.Errorf("workspace-id 未设置: 请用 --workspace-id 指定或导出环境变量 %s(可从控制台流水线 URL /cp/workspace/{id}/pipeline/... 中获取)", envWs)
}
