// Package opts 存放全局 CLI 选项, 由 root 命令的 PersistentFlags 解析填充, 各子命令只读.
package opts

// Opts 全局选项, 子命令通过命令链上的 flag 读取.
type Opts struct {
	AccessKey   string
	SecretKey   string
	WorkspaceId string
	JSON        bool
}

// Global 由 root 命令绑定 flag 后填充, 各子模块命令在此读取.
var Global Opts
