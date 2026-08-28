package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/cpclient"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// checkResult check-credentials 的 JSON 输出结构(字段名遵循 SDK 风格大写).
type checkResult struct {
	Configured bool   `json:"Configured"`
	Valid      bool   `json:"Valid"`
	Region     string `json:"Region"`
	Account    string `json:"Account,omitempty"`
	Error      string `json:"Error,omitempty"`
}

// NewCheckCredentialsCmd 创建 check-credentials 命令.
func NewCheckCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-credentials",
		Short: "检查 AK/SK 是否设置且有效",
		Long: "检查访问凭证:\n" +
			"1. 是否已通过 --ak/--sk 或环境变量 " + envAK + "/" + envSK + " 设置;\n" +
			"2. 调用 ListWorkspaces 验证凭证真实有效(只读操作, 无需 workspace-id).\n" +
			"region 固定 cn-north-1(持续交付仅北京 region).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ak, sk := config.ResolveCredentials(opts.Global.AccessKey, opts.Global.SecretKey)
			res := checkResult{Region: "cn-north-1"}
			if ak == "" || sk == "" {
				res.Error = fmt.Sprintf("AK/SK 未设置: 请用 --ak/--sk 指定或导出环境变量 %s/%s", envAK, envSK)
				return finish(cmd, res)
			}
			res.Configured = true
			// 真实探针: ListWorkspaces 无需 WorkspaceId, 不会因缺参数误报
			c, err := cpclient.New(ak, sk)
			if err != nil {
				res.Error = "创建客户端失败: " + scrubSecret(err.Error(), ak, sk)
				return finish(cmd, res)
			}
			if _, err := c.ListWorkspaces(context.Background(), 1, 1); err != nil {
				res.Error = classifyErr(err) + " (原文: " + scrubSecret(err.Error(), ak, sk) + ")"
				return finish(cmd, res)
			}
			res.Valid = true
			res.Account = config.MaskSecret(ak) // 脱敏展示已配置的 AK
			return finish(cmd, res)
		},
	}
}

// scrubSecret 将错误信息中出现的密钥值替换为 ***, 防止泄露.
func scrubSecret(s, ak, sk string) string {
	for _, secret := range []string{ak, sk} {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "***")
		}
	}
	return s
}

// classifyErr 将 SDK 错误归类为可读建议.
func classifyErr(err error) string {
	s := err.Error()
	switch {
	case contains(s, "InvalidAccessKey", "SignatureDoesNotMatch", "403"):
		return "凭证无效或无权限: 检查 AK/SK 是否正确/已禁用, 以及是否开通持续交付(CP)权限"
	case contains(s, "timeout", "Timeout", "dial tcp"):
		return "网络超时: 检查本机网络与 open.volcengineapi.com 连通性"
	default:
		return "调用失败"
	}
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// finish 按 --json 或文本输出结果; Configured=false 或 Valid=false 时返回错误使退出码非 0.
func finish(cmd *cobra.Command, res checkResult) error {
	if opts.Global.JSON {
		if err := output.PrintJSON(res); err != nil {
			return err
		}
	} else {
		printTextCheck(cmd.OutOrStdout(), res)
	}
	if !res.Valid {
		return fmt.Errorf("凭证检查未通过")
	}
	return nil
}

func printTextCheck(w io.Writer, res checkResult) {
	fmt.Fprintln(w, "凭证检查 (region:", res.Region+")")
	if res.Configured {
		fmt.Fprintln(w, "  ✔ AK/SK 已设置:", res.Account)
	} else {
		fmt.Fprintln(w, "  ✘ AK/SK 未设置")
	}
	if res.Error != "" {
		fmt.Fprintln(w, "  ✘", res.Error)
	} else if res.Valid {
		fmt.Fprintln(w, "  ✔ 调用 ListWorkspaces 验证通过, 凭证有效")
	}
}
