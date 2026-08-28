# volc-cli (火山云 CLI) 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 独立 CLI 工具 `volc-cli`, 通过火山云 Go SDK 查询持续交付(CodePipeline)流水线的发布记录与构建失败错误信息. 发布到 GitHub(git@github.com:yangkushu/volc-cli.git), tag 推送自动构建 win/linux x64 可执行文件, 提供一键安装脚本(自动下载对应平台最新版本 + 安装 Claude Code/Codex skills).

**Architecture:** cobra 两级命令结构 `volc-cli <模块> <命令>`, 模块名与 SDK `service/` 包名对齐(`codepipeline`), 命令名与 OpenAPI Action 对齐(kebab-case). 内部按 `client`(SDK 封装)/`output`(渲染)/`cmd`(cobra 命令) 分层, 互不耦合.

**Tech Stack:** Go 1.26, cobra v1.9.x, github.com/volcengine/volc-sdk-golang v1.0.248

## Global Constraints

- 项目目录: `~/workspaces/kuopin/volc-cli`, 独立 git 仓库, 与 oss 服务完全解耦.
- 模块名 = SDK 包名小写: `codepipeline` (对应 `service/codePipeline`).
- 子命令名 = API Action 转 kebab-case: `ListPipelines` → `list-pipelines`.
- flag 名 = SDK 请求字段转 kebab-case: `WorkspaceId` → `--workspace-id`.
- JSON 输出字段沿用 SDK json tag 原样大写: `"Id"`, `"Status"`, `"StartTime"`.
- region 固定 SDK 默认 `cn-north-1`(CodePipeline 仅北京), 不加 --region flag.
- 凭证读取优先级: `--ak/--sk` flag > 环境变量 `VOLC_ACCESSKEY`/`VOLC_SECRETKEY`(SDK base client 默认读取的名字).
- WorkspaceId 读取优先级: `--workspace-id` flag > 环境变量 `VOLC_CP_WORKSPACE_ID`.
- 每个 cobra 命令必须写 Short/Long 描述与全部 flag 用法说明(--help 自动生成, 与代码同源).
- AK/SK 值禁止打印到终端/日志/错误信息(只显示是否已设置与长度).
- Go 源文件中文注释用半角标点.
- commit message 中文, 简短明确.
- GitHub 仓库: `git@github.com:yangkushu/volc-cli.git`, 工作流和安装脚本中的下载地址固定指向该仓库, 平台/二进制命名见 Task 10.
- 本机没有 gh CLI, GitHub 操作(建 repo/默认分支)由人工完成; 推送与 PR 用 git 直接做.
- 安装脚本幂等(重复运行不报错), 下载失败时给出明确错误与重试指引; 脚本内 curl 必须带 -fSsl 且失败即退出.
- skills 目录名必须为 `volc-cli`(与 SKILL.md 的 name 一致, Claude Code/Codex 按目录发现 skill).

## SDK 关键事实(实现时直接引用, 无需再翻源码)

- `codePipeline.DefaultInstance` 单例; 通过 `instance.Client.SetAccessKey(ak)` / `SetSecretKey(sk)` 设凭证(base/client.go:131,138).
- 凭证也可由 SDK init 从 `VOLC_ACCESSKEY`/`VOLC_SECRETKEY` 自动读取(client.go:81-82), 所以 env 已设时无需显式 Set.
- SDK 默认 region: `base.RegionCnNorth1`, Service: `"cp"`, Host: `open.volcengineapi.com`, Timeout 5s, 内置一次重试.
- wrapper 签名:
  - `ListPipelines(req *ListPipelinesRequest) (*ListPipelinesResponse, error)`
  - `ListPipelineRecords(req *ListPipelineRecordsRequest) (*ListPipelineRecordsResponse, error)`
  - `GetPipelineRecord(req *GetPipelineRecordRequest) (*GetPipelineRecordResponse, error)`
- 请求字段:
  - `ListPipelinesRequest{WorkspaceId string; Filter *PipelineFilter{Name string}; Desc bool; OrderBy string}` — 注意: 无分页字段.
  - `ListPipelineRecordsRequest{WorkspaceId; PipelineId; Page{PageNumber,PageSize int64}; Desc bool; Filter *PipelineRecordFilter{Statuses string /*json:"Name,omitempty"*/}}`
  - `GetPipelineRecordRequest{WorkspaceId; PipelineId; Id string}`
- 响应字段:
  - `ListPipelinesResponse{Total int64; Items []Pipeline{Id,Name,ClusterPool,LastStatus,UpdateTime,Triggerer,...}}`
  - `ListPipelineRecordsResponse{Total int64; Items []PipelineRecord{Id,Status,Creator,StartTime,EndTime,TriggerMode,DynamicEnvs []*KVPair,Description,Stages []PipelineRecordStage,...}}`
  - `GetPipelineRecordResponse` 嵌 `PipelineRecord`(同上, 带 Stages 详情).
- 层级: `PipelineRecord.Stages[].Name/Status` → `.Tasks[].Name/Status/Type` → `.Steps[].Name/Status/Type/Result []KVPair{Key,Value}`.
- 失败状态字符串: Status 为 `"Failed"`(记录/阶段/任务/步骤同名枚举). Filter 按状态筛选时 `PipelineRecordFilter.Statuses` 字段 json tag 实际是 `"Name"`(SDK 模型如此, 直接用).
- 控制台 URL 格式: `https://console.volcengine.com/cp/workspace/{workspaceId}/pipeline/{pipelineId}/record/{recordId}`.

## File Structure

```
volc-cli/
├── go.mod                          # module kuopin/volc-cli
├── main.go                         # 入口: 调 cmd.Execute()
├── README.md                       # 安装/用法/WorkspaceId 获取指引
├── .github/workflows/release.yml   # tag 推送自动构建 win/linux x64 并发布
├── skills/
│   └── volc-cli/SKILL.md           # Claude Code + Codex 共用 skill(--json + 错误自愈工作流)
├── scripts/
│   ├── install.sh                  # 自动下载指定平台最新可执行文件 + 安装 skills(幂等)
│   └── install.ps1                 # 同上, Windows PowerShell 版
├── internal/
│   ├── config/config.go            # 凭证与 workspace 解析(flag+env, 含敏感信息脱敏)
│   ├── cpclient/client.go          # codePipeline SDK 封装(认证, API 调用)
│   ├── failure/extract.go          # 从 PipelineRecord 提取失败信息(stage/task/step 路径+错误KV)
│   ├── output/
│   │   ├── json.go                 # --json 输出
│   │   └── text.go                 # 人类可读输出(表格/树形)
│   └── cmd/
│       ├── root.go                 # 根命令+全局 flag(--ak/--sk/--workspace-id/--json)
│       ├── check_credentials.go    # volc-cli check-credentials
│       └── codepipeline/
│           ├── codepipeline.go     # volc-cli codepipeline 父命令
│           ├── list_pipelines.go   # list-pipelines
│           ├── list_pipeline_records.go  # list-pipeline-records
│           ├── get_pipeline_record.go    # get-pipeline-record
│           └── failures.go         # failures(便捷: 最近失败+错误提取)
└── internal/.../*_test.go          # failure/extract, config, output 的单测
```

各文件职责单一: config 只管参数解析, cpclient 只管 SDK 调用, failure 只管数据提取, output 只管渲染, cmd 只管参数接线. 未来加 `volc-cli tos ...` 时平行新增 `internal/cmd/tos/` 与对应 client 包, root.go 加一行 AddCommand.

---

### Task 1: 项目脚手架 + go.mod + root 命令

**Files:**
- Create: `go.mod`, `main.go`, `internal/cmd/root.go`, `internal/cmd/root_test.go`

**Interfaces:**
- Produces: `cmd.NewRootCommand() *cobra.Command`(供 main.go 调用); 全局 flag `--ak/--sk/--workspace-id/--json` 及 viper 化的 `cmd.GlobalOpts` 结构. 后续所有任务通过 `rootCmd.AddCommand` 挂载子命令.

- [ ] **Step 1: 初始化 module 与依赖**

```bash
cd ~/workspaces/kuopin/volc-cli
go mod init kuopin/volc-cli
go get github.com/spf13/cobra@v1.9.1
go get github.com/volcengine/volc-sdk-golang@v1.0.248
```

- [ ] **Step 2: 写 main.go**

```go
package main

import (
	"os"

	"kuopin/volc-cli/internal/cmd"
)

func main() {
	if err := cmd.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 3: 写 root.go(全局 flag 与 PersistentPreRun 校验)**

```go
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
	envAK    = "VOLC_ACCESSKEY"
	envSK    = "VOLC_SECRETKEY"
	envWs    = "VOLC_CP_WORKSPACE_ID"
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
	}
	pf := root.PersistentFlags()
	pf.StringVar(&globalOpts.AccessKey, "ak", "", "访问密钥 ID, 默认读环境变量 "+envAK)
	pf.StringVar(&globalOpts.SecretKey, "sk", "", "访问密钥 Secret, 默认读环境变量 "+envSK)
	pf.StringVar(&globalOpts.WorkspaceId, "workspace-id", "", "持续交付工作区 ID, 默认读环境变量 "+envWs)
	pf.BoolVar(&globalOpts.JSON, "json", false, "以 JSON 格式输出(字段名与官方 SDK 一致)")
	return root
}

// resolveWorkspaceId 返回生效的 workspace id, 未配置时报错退出.
func resolveWorkspaceId() (string, error) {
	if globalOpts.WorkspaceId != "" {
		return globalOpts.WorkspaceId, ""
	}
	// errcheck 由调用方处理
	return "", fmt.Errorf("workspace-id 未设置: 请用 --workspace-id 指定或导出环境变量 %s(可从控制台流水线 URL /cp/workspace/{id}/pipeline/... 中获取)", envWs)
}
```

注意: `resolveWorkspaceId` 返回 `(string, error)`, 上面占位的 `return "", ""` 应为 `return "", nil`, 以最终编译通过为准.

- [ ] **Step 4: 写 root_test.go 验证 help 可用**

```go
package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// executeInTest 执行命令并捕获输出, 参考 cobra 官方测试模式.
func executeInTest(t *testing.T, root *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestRootHelp(t *testingtestingT) {} // 占位, 实际实现见下

func TestRootHelp(t *testing.T) {
	out, err := executeInTest(t, NewRootCommand(), "--help")
	if err != nil {
		t.Fatalf("help 不应报错: %v", err)
	}
	for _, want := range []string{"codepipeline 未挂载也无妨", "volc-cli"} {
		if !strings.Contains(out, want) {
			t.Errorf("help 输出应包含 %q, 实际: %s", want, out)
		}
	}
}
```

测试中的断言以最终实现为准(初版 root 只有自身描述), 删除占位 func, 断言 `volc-cli` 与 `--json` 出现即可.

- [ ] **Step 5: 编译 + 测试**

```bash
go build ./... && go test ./...
```

Expected: 编译通过, TestRootHelp PASS.

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: 项目脚手架与 root 命令"
```

---

### Task 2: config 包(凭证/workspace 解析与脱敏)

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`

**Interfaces:**
- Consumes: root.go 中的 `globalOpts`(cmd.Opts) 与环境变量名常量.
- Produces: `config.ResolveCredentials(akFlag, skFlag string) (ak, sk string)`, `config.ResolveWorkspaceId(flagVal string) (string, error)`, `config.MaskSecret(s string) string`. Task 3 的 client 构造与各命令的参数校验依赖这些函数.

- [ ] **Step 1: 写失败测试 config_test.go**

```go
package config

import (
	"testing"
)

func TestResolveCredentialsFlagWins(t *testingtestingT) {} // 见下实现

func TestResolveCredentialsFlagWins(t *testing.T) {
	t.Setenv("VOLC_ACCESSKEY", "env-ak")
	t.Setenv("VOLC_SECRETKEY", "env-sk")
	ak, sk := ResolveCredentials("flag-ak", "flag-sk")
	if ak != "flag-ak" || sk != "flag-sk" {
		t.Errorf("flag 应优先, got ak=%q sk=%q", ak, sk)
	}
}

func TestResolveCredentialsEnvFallback(t *testing.T) {
	t.Setenv("VOLC_ACCESSKEY", "env-ak")
	t.Setenv("VOLC_SECRETKEY", "env-sk")
	ak, sk := ResolveCredentials("", "")
	if ak != "env-ak" || sk != "env-sk" {
		t.Errorf("env 应兜底, got ak=%q sk=%q", ak, sk)
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "未设置"},
		{"abc", "***"},
		{"AKLT1234567890abcdef", "AKLT...(len=20)"},
	}
	for _, c := range cases {
		if got := MaskSecret(c.in); got != c.want {
			t.Errorf("MaskSecret(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveWorkspaceIdMissing(t *testing.T) {
	if _, err := ResolveWorkspaceId(""); err == nil {
		t.Error("未设置 workspace 应报错")
	}
}
```

删除占位 func, 保留其余.

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/config/
```

Expected: FAIL, 函数未定义.

- [ ] **Step 3: 实现 config.go**

```go
// Package config 解析 CLI 凭证与工作区配置, flag 优先, 环境变量兜底.
package config

import (
	"fmt"
	"os"
)

const (
	EnvAccessKey   = "VOLC_ACCESSKEY"
	EnvSecretKey   = "VOLC_SECRETKEY"
	EnvWorkspaceId = "VOLC_CP_WORKSPACE_ID"
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

// MaskSecret 返回密钥脱敏描述, 只暴露长度不暴露内容.
func MaskSecret(s string) string {
	if s == "" {
		return "未设置"
	}
	return fmt.Sprintf("*** (len=%d)", len(s))
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/config/ -v
```

Expected: 全部 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config && git commit -m "feat: config 包凭证解析与脱敏"
```

---

### Task 3: cpclient 包(SDK 封装)

**Files:**
- Create: `internal/cpclient/client.go`

**Interfaces:**
- Consumes: SDK `codePipeline.DefaultInstance`, `models.*`(见"SDK 关键事实").
- Produces: 
  - `cpclient.New(ak, sk string) *Client`
  - `(c *Client) ListPipelines(workspaceId string) (*models.ListPipelinesResponse, error)`
  - `(c *Client) ListPipelineRecords(workspaceId, pipelineId string, pageSize int64, filterStatus string) (*models.ListPipelineRecordsResponse, error)`
  - `(c *Client) GetPipelineRecord(workspaceId, pipelineId, recordId string) (*models.GetPipelineRecordResponse, error)`
  - `(c *Client) ConsoleURL(workspaceId, pipelineId, recordId string) string`
  - Task 4+ 所有命令经由这些方法访问火山云.

- [ ] **Step 1: 实现 client.go**

```go
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
```

- [ ] **Step 2: 编译验证**

```bash
go build ./...
```

Expected: 通过(SDK 调用为薄封装, 不写单测; 真实连通性由 check-credentials 与 Task 8 手动验收覆盖).

- [ ] **Step 3: Commit**

```bash
git add internal/cpclient && git commit -m "feat: cpclient 封装 codePipeline SDK"
```

---

### Task 4: failure 包(失败信息提取)

**Files:**
- Create: `internal/failure/extract.go`, `internal/failure/extract_test.go`

**Interfaces:**
- Consumes: `models.PipelineRecord` / `PipelineRecordStage` / `PipelineRecordTask` / `PipelineRecordStep` / `KVPair`.
- Produces:
  - `failure.Failure{Stage, Task, Step, Message string; Details []models.KVPair}`
  - `failure.Extract(record *models.PipelineRecord) []Failure` — 遍历三层, 收集 Status=="Failed" 的 step; Message 取 Result KV 中第一个可读值(见 messageKeys).
  - `failure.ConsoleURL(workspaceId, pipelineId, recordId string) string` — 从 cpclient 移到这里(渲染层需要), cpclient 保留自己的但内部共用此函数; 若重复则只在 failure 提供, cpclient 删除自己的(二选一, 以"只在一处"为准).

- [ ] **Step 1: 写失败测试 extract_test.go**

```go
package failure

import (
	"testing"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
)

func failedStep(name string, kvs ...models.KVPair) models.PipelineRecordStep {
	return models.PipelineRecordStep{Name: name, Status: "Failed", Result: kvs}
}

func TestExtractFindsFailedStep(t *testingtestingT) {} // 删除此占位

func TestExtractFindsFailedStep(t *testing.T) {
	rec := &models.PipelineRecord{
		Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "构建", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "build-image", Status: "Failed", Type: "Build",
				Steps: []models.PipelineRecordStep{
					{Name: "checkout", Status: "Success"},
					failedStep("docker-build",
						models.KVPair{Key: "exit_code", Value: "1"},
						models.KVPair{Key: "error", Value: "exited with code 1"}),
				},
			}},
		}},
	}
	got := Extract(rec)
	if len(got) != 1 {
		t.Fatalf("应提取 1 条失败, got %d", len(got))
	}
	f := got[0]
	if f.Stage != "构建" || f.Task != "build-image" || f.Step != "docker-build" {
		t.Errorf("路径错误: %+v", f)
	}
	if f.Message != "exited with code 1" {
		t.Errorf("Message 应取 error KV, got %q", f.Message)
	}
	if len(f.Details) != 2 {
		t.Errorf("Details 应保留全部 Result KV, got %d", len(f.Details))
	}
}

func TestExtractNoFailure(t *testing.T) {
	rec := &models.PipelineRecord{Status: "Success"}
	if got := Extract(rec); len(got) != 0 {
		t.Errorf("成功记录不应有失败项, got %d", len(got))
	}
}

func TestExtractMessageFallback(t *testing.T) {
	rec := &models.PipelineRecord{
		Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "部署", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "deploy", Status: "Failed",
				Steps: []models.PipelineRecordStep{
					{Name: "kubectl", Status: "Failed", Result: []models.KVPair{{Key: "log", Value: "pod crash"}}},
				},
			}},
		}},
	}
	got := Extract(rec)
	if len(got) != 1 || got[0].Message != "pod crash" {
		t.Errorf("无 error/message KV 时应兜底取第一个非空 KV, got %+v", got)
	}
}
```

删除占位 func, 保留其余.

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/failure/
```

Expected: FAIL, Extract 未定义.

- [ ] **Step 3: 实现 extract.go**

```go
// Package failure 从流水线执行记录中提取失败定位信息.
package failure

import (
	"strings"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
)

// messageKeys Message 取值优先级: 命中这些 key 的 KV 优先作为 Message.
var messageKeys = []string{"error", "message", "errorMsg", "reason", "log"}

// Failure 一处失败的定位路径与错误信息.
type Failure struct {
	Stage   string          `json:"Stage"`
	Task    string          `json:"Task"`
	Step    string          `json:"Step"`
	Message string          `json:"Message"`
	Details []models.KVPair `json:"Details,omitempty"`
}

// Extract 遍历 Stage→Task→Step, 收集所有 Status==Failed 的 step 失败信息.
func Extract(record *models.PipelineRecord) []Failure {
	var out []Failure
	if record == nil {
		return out
	}
	for _, st := range record.Stages {
		for _, tk := range st.Tasks {
			for _, sp := range tk.Steps {
				if sp.Status != "Failed" {
					continue
				}
				out = append(out, Failure{
					Stage:   st.Name,
					Task:    tk.Name,
					Step:    sp.Name,
					Message: pickMessage(sp.Result),
					Details: sp.Result,
				})
			}
		}
	}
	return out
}

// pickMessage 按优先级选可读错误信息, 全部为空时回退第一个非空 KV.
func pickMessage(kvs []models.KVPair) string {
	for _, k := range messageKeys {
		for _, kv := range kvs {
			if strings.EqualFold(kv.Key, k) && strings.TrimSpace(kv.Value) != "" {
				return kv.Value
			}
		}
	}
	for _, kv := range kvs {
		if strings.TrimSpace(kv.Value) != "" {
			return kv.Value
		}
	}
	return "详见控制台日志"
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/failure/ -v
```

Expected: 全部 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/failure && git commit -m "feat: failure 失败信息提取"
```

---

### Task 5: output 包(text/json 渲染)

**Files:**
- Create: `internal/output/json.go`, `internal/output/text.go`, `internal/output/output_test.go`

**Interfaces:**
- Consumes: `models.ListPipelinesResponse`, `models.PipelineRecord`, `failure.Failure`.
- Produces:
  - `output.PrintJSON(v any) error` — MarshalIndent 到 stdout.
  - `output.PrintPipelines(items []models.Pipeline) error`
  - `output.PrintRecords(items []models.PipelineRecord) error` — 表格: Id/Status/TriggerMode/StartTime/EndTime/DynamicEnvs/Description.
  - `output.PrintRecordDetail(record *models.PipelineRecord, fs []failure.Failure, url string) error` — 树形: 失败分支展开, 成功折叠.
  - Task 6-9 的所有命令最终调用这些函数输出.

- [ ] **Step 1: 写失败测试 output_test.go**

```go
package output

import (
	"bytes"
	"testing"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
)

func TestPrintJSONUsesSDKTags(t *testingtestingT) {} // 删除占位

func TestPrintJSONUsesSDKTags(t *testing.T) {
	var buf bytes.Buffer
	if err := printTo(&buf, func() error { return PrintJSONTo(&buf, models.KVPair{Key: "k", Value: "v"}) }); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"Key"`)) {
		t.Errorf("JSON 字段名应保持 SDK 原样 Key/Value, got %s", buf.String())
	}
}

func TestPrintRecordsTable(t *testing.T) {
	var buf bytes.Buffer
	recs := []models.PipelineRecord{{
		Id: "rec-1", Status: "Failed", TriggerMode: "Manual",
		StartTime: "2026-08-27 14:32:10",
		DynamicEnvs: []*models.KVPair{{Key: "BRANCH", Value: "master"}},
	}}
	if err := PrintRecordsTo(&buf, recs); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"rec-1", "Failed", "Manual", "BRANCH=master"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("表格应包含 %q, got:\n%s", want, buf.String())
		}
	}
}

func TestPrintRecordDetailFailedPath(t *testing.T) {
	var buf bytes.Buffer
	rec := &models.PipelineRecord{
		Id: "rec-1", Status: "Failed",
		Stages: []models.PipelineRecordStage{{
			Name: "构建", Status: "Failed",
			Tasks: []models.PipelineRecordTask{{
				Name: "build", Status: "Failed",
				Steps: []models.PipelineRecordStep{{Name: "docker-build", Status: "Failed"}},
			}},
		}},
	}
	fs := []failure.Failure{{Stage: "构建", Task: "build", Step: "docker-build", Message: "exit 1"}}
	err := PrintRecordDetailTo(&buf, rec, fs, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"rec-1", "Failed", "构建", "docker-build", "exit 1", "https://example.com"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("详情应包含 %q, got:\n%s", want, out)
		}
	}
}
```

实现时统一采用 `PrintXxxTo(w io.Writer, ...)` 形式, stdout 版本是一行包装; 测试中的 `printTo` 辅助函数按此简化为直接调 `PrintJSONTo`. 删除占位 func.

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/output/
```

Expected: FAIL.

- [ ] **Step 3: 实现 json.go 与 text.go**

json.go:

```go
// Package output 负责终端渲染, text 为人类可读, json 为结构化输出.
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// PrintJSONTo 将 v 以缩进 JSON 写入 w.
func PrintJSONTo(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintJSON 输出到 stdout.
func PrintJSON(v any) error {
	return PrintJSONTo(stdout, v)
}
```

text.go(核心结构, fmt 排版可微调):

```go
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
)

var stdout io.Writer = os.Stdout

// PrintPipelinesTo 以表格输出流水线列表.
func PrintPipelinesTo(w io.Writer, items []models.Pipeline) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无流水线")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tLAST_STATUS\tUPDATE_TIME\tTRIGGERER")
	for _, p := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Id, p.Name, p.LastStatus, p.UpdateTime, p.Triggerer)
	}
	return tw.Flush()
}

// PrintRecordsTo 以表格输出执行记录(含发布参数 DynamicEnvs).
func PrintRecordsTo(w io.Writer, items []models.PipelineRecord) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无执行记录")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tTRIGGER_MODE\tSTART_TIME\tEND_TIME\tDYNAMIC_ENVS\tDESCRIPTION")
	for _, r := range items {
		envs := make([]string, 0, len(r.DynamicEnvs))
		for _, kv := range r.DynamicEnvs {
			envs = append(envs, kv.Key+"="+kv.Value)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Id, r.Status, r.TriggerMode, r.StartTime, endTimeOr(r.EndTime), strings.Join(envs, ","), r.Description)
	}
	return tw.Flush()
}

// PrintRecordDetailTo 树形输出单条记录: 失败分支展开错误信息, 成功分支折叠.
func PrintRecordDetailTo(w io.Writer, record *models.PipelineRecord, fs []failure.Failure, url string) error {
	fmt.Fprintf(w, "记录: %s  状态: %s  %s ~ %s\n", record.Id, record.Status, record.StartTime, endTimeOr(record.EndTime))
	if len(record.DynamicEnvs) > 0 {
		envs := make([]string, 0, len(record.DynamicEnvs))
		for _, kv := range record.DynamicEnvs {
			envs = append(envs, kv.Key+"="+kv.Value)
		}
		fmt.Fprintf(w, "发布参数: %s\n", strings.Join(envs, ", "))
	}
	for _, st := range record.Stages {
		fmt.Fprintf(w, "Stage: %s [%s]\n", st.Name, st.Status)
		for _, tk := range st.Tasks {
			fmt.Fprintf(w, "  Task: %s [%s]\n", tk.Name, tk.Status)
			for _, sp := range tk.Steps {
				if sp.Status == "Failed" {
					fmt.Fprintf(w, "    ✘ Step: %s [%s]\n", sp.Name, sp.Status)
					for _, f := range fs {
						if f.Step == sp.Name {
							fmt.Fprintf(w, "      错误: %s\n", f.Message)
							for _, kv := range f.Details {
								fmt.Fprintf(w, "        %s: %s\n", kv.Key, truncate(kv.Value, 200))
							}
						}
					}
				} else {
					fmt.Fprintf(w, "    ✔ Step: %s [%s]\n", sp.Name, sp.Status)
				}
			}
		}
	}
	fmt.Fprintf(w, "控制台: %s\n", url)
	return nil
}

// endTimeOr 空结束时间显示运行中.
func endTimeOr(s string) string {
	if s == "" {
		return "(运行中)"
	}
	return s
}

// truncate 截断长字符串, 避免 Result KV 单值刷屏.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "...(截断)"
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/output/ -v
```

Expected: 全部 PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output && git commit -m "feat: output 渲染层"
```

---

### Task 6: check-credentials 命令

**Files:**
- Create: `internal/cmd/check_credentials.go`, `internal/cmd/check_credentials_test.go`

**Interfaces:**
- Consumes: `config.ResolveCredentials/MaskSecret`, `cpclient.New().ListPipelines`(作为有效性的真实探针).
- Produces: `volc-cli check-credentials` 命令; `cmd.NewCheckCredentialsCmd() *cobra.Command` 由 root AddCommand.

- [ ] **Step 1: 写失败测试**

```go
package cmd

import (
	"strings"
	"testing"
)

func TestCheckCredentialsMissing(t *testing.T) {
	out, err := executeInTest(t, NewCheckCredentialsCmd(), "--ak", "", "--sk", "")
	// --ak "" 显式传空且环境变量被清空时应报错提示; 具体断言以实现为准:
	if err == nil && !strings.Contains(out, "未设置") {
		t.Errorf("AK/SK 未设置时应有明确提示, out=%s err=%v", out, err)
	}
}
```

注意: 测试需先清空 VOLC_ACCESSKEY/VOLC_SECRETKEY 环境变量(`t.Setenv(envAK, "")`), 断言按最终实现调整.

- [ ] **Step 2: 跑测试确认失败**

```bash
go test ./internal/cmd/ -run TestCheckCredentials
```

Expected: FAIL, NewCheckCredentialsCmd 未定义.

- [ ] **Step 3: 实现 check_credentials.go**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/cpclient"
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
			"2. 调用 ListPipelines 验证凭证真实有效(只读操作).\n" +
			"region 固定 cn-north-1(持续交付仅北京 region).",
		RunE: func(cmd *cobra.Command, args []string) error {
			ak, sk := config.ResolveCredentials(globalOpts.AccessKey, globalOpts.SecretKey)
			res := checkResult{Region: "cn-north-1"}
			if ak == "" || sk == "" {
				res.Error = fmt.Sprintf("AK/SK 未设置: 请用 --ak/--sk 指定或导出环境变量 %s/%s", envAK, envSK)
				return finish(cmd, res)
			}
			res.Configured = true
			// 真实探针: ListPipelines 不需要 workspace 时可传空, 失败再区分鉴权错/参数错
			c := cpclient.New(ak, sk)
			if _, err := c.ListPipelines(""); err != nil {
				res.Error = classifyErr(err) + " (原文: " + err.Error() + ")"
				return finish(cmd, res)
			}
			res.Valid = true
			res.Account = config.MaskSecret(ak) // 脱敏展示已配置的 AK
			return finish(cmd, res)
		},
	}
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
	if globalOpts.JSON {
		return output.PrintJSON(res)
	}
	printTextCheck(cmd.OutOrStdout(), res)
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
		fmt.Fprintln(w, "  ✔ 调用 ListPipelines 验证通过, 凭证有效")
	}
}
```

import 补齐 `io`, `os`, `strings`(printTextCheck 用 io.Writer, cmd.OutOrStdout 需 cobra context). 编译报错时按提示补.

- [ ] **Step 4: 跑测试确认通过**

```bash
go test ./internal/cmd/ -run TestCheckCredentials -v
```

Expected: PASS(未设置分支).

- [ ] **Step 5: 挂载到 root 并全量测试**

root.go 的 `NewRootCommand` 中加:

```go
root.AddCommand(NewCheckCredentialsCmd())
```

```bash
go build ./... && go test ./...
```

Expected: 全部 PASS.

- [ ] **Step 6: 真实凭证手动验收(需要用户环境)**

```bash
export VOLC_ACCESSKEY=... VOLC_SECRETKEY=...
go run . check-credentials
go run . check-credentials --json
```

Expected: 文本模式显示两行 ✔; JSON 模式 `"Valid": true`. 若报权限错误, 检查 key 的 CP 权限.

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat: check-credentials 凭证检查命令"
```

---

### Task 7: codepipeline 模块命令(list-pipelines / list-pipeline-records / get-pipeline-record)

**Files:**
- Create: `internal/cmd/codepipeline/codepipeline.go`, `internal/cmd/codepipeline/list_pipelines.go`, `internal/cmd/codepipeline/list_pipeline_records.go`, `internal/cmd/codepipeline/get_pipeline_record.go`
- Modify: `internal/cmd/root.go`(AddCommand 挂载模块)

**Interfaces:**
- Consumes: `cmd.GlobalOpts`(改为导出 `cmd.Global` 或提供 getter, 由 root.go 决定), `config.ResolveWorkspaceId`, `cpclient` 三方法, `output.PrintXxx`.
- Produces: 三个命令; `codepipeline.NewCodePipelineCmd() *cobra.Command` 挂到 root; 共享辅助 `resolveWs()`(模块内 workspace 解析+报错), `findPipelineByName(ws, name) (*models.Pipeline, error)`(ListPipelines 后按名称或 ID 前缀匹配). Task 8 的 failures 复用这两辅助.

- [ ] **Step 1: 实现模块父命令与共享辅助 codepipeline.go**

```go
// Package codepipeline 提供 volc-cli codepipeline 子命令组, 对应 SDK service/codePipeline.
package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	cmdroot "kuopin/volc-cli/internal/cmd"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/cpclient"
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
	return config.ResolveWorkspaceId(cmdroot.Global.WorkspaceId)
}

// newClient 用全局凭证构造 cpclient.
func newClient() *cpclient.Client {
	ak, sk := config.ResolveCredentials(cmdroot.Global.AccessKey, cmdroot.Global.SecretKey)
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
```

补 `strings` import. `cmdroot.Global` 要求 Task 1 的 root.go 把 `globalOpts` 导出为 `Global`(或提供访问函数), 本步顺带完成该修改.

- [ ] **Step 2: 实现 list_pipelines.go**

```go
package codepipeline

import (
	"github.com/spf13/cobra"
	cmdroot "kuopin/volc-cli/internal/cmd"
	"kuopin/volc-cli/internal/output"
)

func newListPipelinesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-pipelines",
		Short: "ListPipelines: 列出工作区全部流水线",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			resp, err := newClient().ListPipelines(ws)
			if err != nil {
				return err
			}
			if cmdroot.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintPipelines(resp.Items)
		},
	}
}
```

- [ ] **Step 3: 实现 list_pipeline_records.go**

```go
package codepipeline

import (
	"github.com/spf13/cobra"
	cmdroot "kuopin/volc-cli/internal/cmd"
	"kuopin/volc-cli/internal/output"
)

func newListPipelineRecordsCmd() *cobra.Command {
	var pageSize int64
	var status string
	c := &cobra.Command{
		Use:   "list-pipeline-records",
		Short: "ListPipelineRecords: 列出流水线执行记录(含发布参数)",
		Long: "按名称或 ID 定位流水线(需先 ListPipelines), 倒序列出执行记录.\n" +
			"记录含 TriggerMode(触发方式)与 DynamicEnvs(每次发布的参数).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			c := newClient()
			p, err := findPipelineByName(c, ws, args[0])
			if err != nil {
				return err
			}
			resp, err := c.ListPipelineRecords(ws, p.Id, pageSize, status)
			if err != nil {
				return err
			}
			if cmdroot.Global.JSON {
				return output.PrintJSON(resp)
			}
			return output.PrintRecords(resp.Items)
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 10, "返回记录条数, 默认 10")
	c.Flags().StringVar(&status, "status", "", "按状态过滤, 如 Failed/Success, 默认不过滤")
	return c
}
```

补 `fmt` import.

- [ ] **Step 4: 实现 get_pipeline_record.go**

```go
package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	cmdroot "kuopin/volc-cli/internal/cmd"
	"kuopin/volc-cli/internal/failure"
	"kuopin/volc-cli/internal/output"
)

func newGetPipelineRecordCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get-pipeline-record",
		Short: "GetPipelineRecord: 查询单条执行记录详情(含失败信息)",
		Long: "定位流水线后查询指定 record 详情, 展示 Stage/Task/Step 三层结构,\n" +
			"失败步骤展开错误信息, 并附控制台 URL.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			recordId, _ := cmd.Flags().GetString("id")
			if recordId == "" {
				return fmt.Errorf("必须用 --id 指定记录 ID(可先用 list-pipeline-records 查看)")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			c := newClient()
			p, err := findPipelineByName(c, ws, args[0])
			if err != nil {
				return err
			}
			resp, err := c.GetPipelineRecord(ws, p.Id, recordId)
			if err != nil {
				return err
			}
			var rec *models.PipelineRecord
			if resp != nil {
				rec = &resp.Record
			}
			fs := failure.Extract(rec)
			if cmdroot.Global.JSON {
				return output.PrintJSON(map[string]any{
					"Record":   rec,
					"Failures": fs,
					"ConsoleURL": c.ConsoleURL(ws, p.Id, recordId),
				})
			}
			return output.PrintRecordDetail(rec, fs, c.ConsoleURL(ws, p.Id, recordId))
		},
	}
}
```

命令上需注册 `c.Flags().String("id", "", "执行记录 ID(必填)")`. 验证 `GetPipelineRecordResponse` 是否有 `Record` 字段(SDK 源码 pipeline_record.go:23-24 处定义, 若字段名不同以 SDK 为准).

- [ ] **Step 5: 挂载到 root, 编译全测**

root.go 增加:

```go
root.AddCommand(codepipeline.NewCodePipelineCmd())
```

注意 newFailuresCmd 在 Task 8 实现, 本任务先在 codepipeline.go 的 AddCommand 里注释掉该行或创建空壳, Task 8 补齐. 推荐本任务先写空壳:

```go
// newFailuresCmd Task 8 实现.
func newFailuresCmd() *cobra.Command {
	return &cobra.Command{Use: "failures", Short: "最近失败记录(待实现)", Hidden: true}
}
```

```bash
go build ./... && go test ./... && go run . codepipeline --help
```

Expected: 编译测试通过, help 列出 4 个命令(failures 标记 Hidden).

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: codepipeline 三查询命令"
```

---

### Task 8: failures 命令 + 真实流水线验收

**Files:**
- Create: `internal/cmd/codepipeline/failures.go`
- Modify: `internal/cmd/codepipeline/codepipeline.go`(空壳替换为真实命令, 移除 Hidden)

**Interfaces:**
- Consumes: Task 7 的 `resolveWs/newClient/findPipelineByName`, `cpclient.ListPipelineRecords/GetPipelineRecord/ConsoleURL`, `failure.Extract`, `output.PrintRecordDetail`.
- Produces: `volc-cli codepipeline failures <流水线> [--page-size N]`, 默认 page-size=1 即"最后一次失败记录".

- [ ] **Step 1: 实现 failures.go**

```go
package codepipeline

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	cmdroot "kuopin/volc-cli/internal/cmd"
	"kuopin/volc-cli/internal/failure"
	"kuopin/volc-cli/internal/output"
)

func newFailuresCmd() *cobra.Command {
	var pageSize int64
	c := &cobra.Command{
		Use:   "failures",
		Short: "最近 N 次失败记录及错误信息(默认 1 次)",
		Long: "组合命令: ListPipelineRecords(Filter=Failed) + GetPipelineRecord.\n" +
			"对每条失败记录提取 Stage→Task→Step 失败路径与错误信息, 附控制台 URL.\n" +
			"--page-size 1 时即'最后一次失败记录'.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("需要恰好 1 个参数: 流水线名称或 ID")
			}
			ws, err := resolveWs()
			if err != nil {
				return err
			}
			c := newClient()
			p, err := findPipelineByName(c, ws, args[0])
			if err != nil {
				return err
			}
			list, err := c.ListPipelineRecords(ws, p.Id, pageSize, "Failed")
			if err != nil {
				return err
			}
			if len(list.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "无失败记录")
				return nil
			}
			type item struct {
				Record     *models.PipelineRecord `json:"Record"`
				Failures   []failure.Failure      `json:"Failures"`
				ConsoleURL string                 `json:"ConsoleURL"`
			}
			items := make([]item, 0, len(list.Items))
			for _, brief := range list.Items {
				detail, err := c.GetPipelineRecord(ws, p.Id, brief.Id)
				if err != nil {
					return fmt.Errorf("查询记录 %s 详情失败: %w", brief.Id, err)
				}
				var rec *models.PipelineRecord
				if detail != nil {
					rec = &detail.Record
				}
				items = append(items, item{
					Record:     rec,
					Failures:   failure.Extract(rec),
					ConsoleURL: c.ConsoleURL(ws, p.Id, brief.Id),
				})
			}
			if cmdroot.Global.JSON {
				return output.PrintJSON(map[string]any{
					"PipelineId": p.Id, "PipelineName": p.Name, "Items": items,
				})
			}
			for _, it := range items {
				if err := output.PrintRecordDetail(it.Record, it.Failures, it.ConsoleURL); err != nil {
					return err
				}
				fmt.Println()
			}
			return nil
		},
	}
	c.Flags().Int64Var(&pageSize, "page-size", 1, "返回失败记录条数, 默认 1(最后一次失败)")
	return c
}
```

同时删除 codepipeline.go 中 Task 7 的空壳定义, AddCommand 保持不变.

- [ ] **Step 2: 编译全测 + help 检查**

```bash
go build ./... && go test ./... && go run . --help && go run . codepipeline failures --help
```

Expected: 全过; failures help 展示用途与 --page-size.

- [ ] **Step 3: 真实流水线手动验收(需要用户环境: VOLC_ACCESSKEY/VOLC_SECRETKEY/VOLC_CP_WORKSPACE_ID)**

```bash
go run . codepipeline list-pipelines
go run . codepipeline list-pipeline-records <流水线名> --page-size 5
go run . codepipeline failures <流水线名>
go run . codepipeline failures <流水线名> --json
```

Expected: 
1. list-pipelines 列出真实流水线;
2. records 表格含 TriggerMode 与 DynamicEnvs(发布参数);
3. failures 默认输出最后一次失败的 stage/task/step + 错误信息 + 控制台 URL, 与控制台页面对照一致;
4. --json 输出可被 jq 解析.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: failures 最近失败记录命令"
```

---

### Task 9: README

**Files:**
- Create: `README.md`

**Interfaces:**
- Consumes: 全部已实现命令.
- Produces: 使用文档; 后续维护 help 的约定.

- [ ] **Step 1: 写 README.md**

```markdown
# volc-cli

火山云(Volcengine) CLI 工具. 命令结构与官方 Go SDK 对齐:

    volc-cli <模块> <命令>

- 模块名 = SDK `service/` 包名小写(如 `codepipeline`)
- 命令名 = OpenAPI Action 转 kebab-case(如 `ListPipelines` → `list-pipelines`)
- flag 名 = SDK 请求字段转 kebab-case(如 `--workspace-id`)
- `--json` 输出字段名与 SDK json tag 一致(保持大写)

## 安装

    go install kuopin/volc-cli@latest   # 或 go build -o volc-cli .

## 凭证

优先级: flag > 环境变量.

    export VOLC_ACCESSKEY=AKxxx
    export VOLC_SECRETKEY=SKxxx
    export VOLC_CP_WORKSPACE_ID=从控制台流水线 URL 获取(/cp/workspace/{id}/pipeline/...)

验证:

    volc-cli check-credentials

region 固定 cn-north-1(持续交付仅北京 region).

## 持续交付(codepipeline)

    # 流水线列表
    volc-cli codepipeline list-pipelines

    # 发布记录(含每次发布的参数 DynamicEnvs)
    volc-cli codepipeline list-pipeline-records <流水线名或ID> --page-size 10

    # 单条记录详情(失败步骤展开错误信息)
    volc-cli codepipeline get-pipeline-record <流水线名或ID> --id <记录ID>

    # 最近一次失败(默认)/最近 N 次失败: stage→task→step 定位 + 错误信息 + 控制台 URL
    volc-cli codepipeline failures <流水线名或ID> [--page-size N]

所有命令加 `--json` 切结构化输出.

## 扩展新模块

1. `internal/<module>client/` 封装对应 SDK service 包(参照 cpclient)
2. `internal/cmd/<module>/` 建子命令组, 命令名/flag 名按上述对齐规则
3. `internal/cmd/root.go` AddCommand 一行挂载

## 维护约定

- 修改命令行为时同步更新其 Short/Long(flag 用法), --help 由 cobra 自动生成, 与代码同源.
- 新增 SDK 方法封装时命令名与 flag 名照 SDK 抄, 不自造名.
```

- [ ] **Step 2: Commit**

```bash
git add README.md && git commit -m "docs: README 用法与扩展约定"
```

---

### Task 10: GitHub 发布工作流(tag 触发构建 win/linux x64)

**Files:**
- Create: `.github/workflows/release.yml`

**Interfaces:**
- Consumes: 仓库 go.mod(module kuopin/volc-cli), 仓库地址 git@github.com:yangkushu/volc-cli.git.
- Produces: tag `v*` 推送触发的工作流; 产出 `volc-cli-linux-amd64` 与 `volc-cli-windows-amd64.exe` 两个 Release 资产. Task 11 安装脚本的下载 URL 依赖本任务的资产命名.
- 验收依赖: 需推 tag 到 GitHub 后由 Actions 实跑, 本机无法模拟; 静态校验(yaml 语法/资产名一致)完成后, 真实触发按"验证注意事项"交人工执行.

- [ ] **Step 1: 写 .github/workflows/release.yml**

```yaml
name: release

# tag 推送自动构建 win/linux x64 并发布 Release
on:
  push:
    tags:
      - "v*"
  workflow_dispatch:   # 手动兜底(默认按 push 时的 ref 构建)

permissions:
  contents: write      # 上传 Release 资产所需

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        include:
          - goos: linux
            goarch: amd64
            ext: ""
          - goos: windows
            goarch: amd64
            ext: ".exe"
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"
      - name: Build
        env:
          CGO_ENABLED: "0"
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          mkdir -p dist
          go build -trimpath -ldflags "-s -w" -o "dist/volc-cli-${{ matrix.goos }}-${{ matrix.goarch }}${{ matrix.ext }}" .
      - name: Upload binaries
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/volc-cli-linux-amd64
            dist/volc-cli-windows-amd64.exe
          generate_release_notes: true
          fail_on_unmatched_files: true
```

关键点:
- 两平台同 job 分步构建(ubuntu runner 交叉编译, CGO_ENABLED=0 静态链接), 产物名 = `volc-cli-{goos}-{goarch}[-.exe]`, 与 Task 11 下载 URL 严格一致.
- 每个 matrix 腿都会跑一次 Upload 步骤, 第二次 Upload 对同一 Release 幂等(softprops/action-gh-release 会复用 tag 对应的 release), 这是该 action 的已知用法, 无需合并步骤.
- 不要加 checksum 步骤(安装脚本不校验, 避免多余机制).
- workflow_dispatch 为可选配置: 若希望手动也能触发, 保留; 不想要可去掉.

- [ ] **Step 2: 本地静态校验**

```bash
python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/release.yml'))" 2>/dev/null \
  || echo "pyyaml 不可用, 用 go 校验" && go run github.com/goccy/go-yaml/cmd/yq@latest . .github/workflows/release.yml >/dev/null 2>&1 && echo yaml OK
```

Expected: yaml 语法合法. 资产名 `volc-cli-linux-amd64` / `volc-cli-windows-amd64.exe` 与 Task 11 脚本一致.

- [ ] **Step 3: 验证注意事项(写进任务报告, 不实际执行)**

以下步骤需要推 tag 到 GitHub(本机无法模拟 Actions 运行), 留待合并后由人工执行:

1. `git tag v0.1.0 && git push origin v0.1.0`
2. Actions 绿后检查 Release 页面: 两个资产存在且可下载
3. Linux 下载后 `chmod +x && ./volc-cli-linux-amd64 --help` 可运行

- [ ] **Step 4: Commit**

```bash
git add .github && git commit -m "ci: tag 发布工作流(win/linux x64)"
```

---

### Task 11: skills + 安装脚本(自动下载最新版 + 装 Claude Code/Codex skills)

**Files:**
- Create: `skills/volc-cli/SKILL.md`, `scripts/install.sh`, `scripts/install.ps1`
- Modify: 无(README 由 Task 12 重写)

**Interfaces:**
- Consumes: Task 10 的 Release 资产名 `volc-cli-{os}-{arch}`, GitHub 仓库 yangkushu/volc-cli, CLI 的 `--json` 与命令面(见 Task 6-8).
- Produces:
  - `skills/volc-cli/SKILL.md` — Claude Code 与 Codex 共用的 skill(两者 skills 格式相同: SKILL.md + frontmatter).
  - `scripts/install.sh` / `scripts/install.ps1` — 自动检测平台(win/linux)下载最新 release 可执行文件, 放到 `~/bin`(linux)/`~/AppData/Local/volc-cli`(win) 并写入 PATH 提示, 同时把 skills 目录安装到 `~/.claude/skills/volc-cli` 与 `~/.codex/skills/volc-cli`(已存在同名目录时覆盖内容, 幂等).
- 安装脚本要求: curl 必须带 `-fSsl`, 下载失败即 `set -e` 退出并提示重试; 脚本可重复运行(幂等); Windows 版用 PowerShell 语法, 下载用 `Invoke-WebRequest`.

- [ ] **Step 1: 写 skills/volc-cli/SKILL.md**

```markdown
---
name: volc-cli
description: 查询火山云持续交付(CodePipeline)流水线的发布记录与失败原因. 用于排查流水线构建/发布失败、查看某次发布的参数(DynamicEnvs)、或列出工作区全部流水线.
---

# volc-cli 流水线排查

## 触发条件

- 用户提到流水线/构建失败, 要查失败原因或错误日志
- 用户要查某条流水线的发布记录、发布参数
- 用户要列出火山云持续交付工作区的流水线

## 前置要求

- 已安装 volc-cli(见仓库 README 安装章节)
- 环境变量已配置: `VOLC_ACCESSKEY`、`VOLC_SECRETKEY`、`VOLC_CP_WORKSPACE_ID`
  (WorkspaceId 从控制台流水线 URL `/cp/workspace/{id}/pipeline/...` 获取)
- 未配置时先跑 `volc-cli check-credentials` 看提示, 不要猜测

## 操作流程

1. 先看环境是否就绪: `volc-cli check-credentials`; 报错时按输出提示修复
2. 查失败原因: `volc-cli codepipeline failures <流水线名或ID> [--page-size N]`
   - 默认返回最后一次失败, 输出 stage→task→step 定位与错误信息、控制台 URL
3. 查发布记录与参数: `volc-cli codepipeline list-pipeline-records <流水线名或ID> [--page-size N]`
4. 列流水线(不知道名称/ID 时): `volc-cli codepipeline list-pipelines`
5. 单条记录详情: `volc-cli codepipeline get-pipeline-record <流水线名或ID> --id <记录ID>`

## 关键规则

- 流水线定位失败(提示"未找到流水线")时, 先 `list-pipelines` 拿真实名称, 不要猜
- 失败信息在 step 的 Result KV 里; 输出为空或只有"详见控制台日志"时, 把控制台 URL 给用户, 并说明完整日志在控制台
- AK/SK 是敏感信息: 永远不要把 key 内容写进命令输出、日志或对话
- 给用户排查结论时, 先给定位(stage/task/step), 再给错误原文, 最后给控制台链接
```

- [ ] **Step 2: 写 scripts/install.sh**

```bash
#!/usr/bin/env bash
# volc-cli 一键安装: 下载最新 release 可执行文件 + 安装 Claude Code/Codex skills
# 用法: curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh | bash
set -euo pipefail

REPO="yangkushu/volc-cli"
BIN="volc-cli"
API="https://api.github.com/repos/${REPO}/releases/latest"

# 1. 平台检测(只支持 linux/windows x64)
os="$(uname -s)"
case "$os" in
  Linux)  asset="volc-cli-linux-amd64" ;;
  MINGW*|MSYS*|CYGWIN*) asset="volc-cli-windows-amd64.exe" ;;
  *) echo "不支持的平台: $os (仅支持 linux/windows x64)" >&2; exit 1 ;;
esac

# 2. 下载最新 release
url="$(curl -fsSL "$API" | sed -n "s/.*\"browser_download_url\": *\"\([^\"]*${asset}[^\"]*\)\".*/\1/p" | head -1)"
if [ -z "$url" ]; then
  echo "未找到资产 ${asset}, 请检查 https://github.com/${REPO}/releases/latest" >&2
  exit 1
fi
install_dir="${VOLC_CLI_INSTALL_DIR:-$HOME/bin}"
mkdir -p "$install_dir"
tmp="$(mktemp)"
curl -fSL -o "$tmp" "$url"
chmod +x "$tmp"
mv "$tmp" "$install_dir/$BIN"

# 3. 安装 skills(claude code + codex 共用同一份 SKILL.md)
skills_src="https://raw.githubusercontent.com/${REPO}/master/skills/volc-cli/SKILL.md"
for dest in "$HOME/.claude/skills/volc-cli" "$HOME/.codex/skills/volc-cli"; do
  mkdir -p "$dest"
  curl -fSL -o "$dest/SKILL.md" "$skills_src"
  echo "skill 已安装: $dest"
done

echo "✔ 安装完成: $install_dir/$BIN"
echo "  请确认 $install_dir 在 PATH 中, 或执行: export PATH=\"$install_dir:\$PATH\""
echo "  凭证配置: export VOLC_ACCESSKEY=... VOLC_SECRETKEY=... VOLC_CP_WORKSPACE_ID=..."
echo "  验证: volc-cli check-credentials"
```

- [ ] **Step 3: 写 scripts/install.ps1**

```powershell
# volc-cli 一键安装(Windows): 下载最新 release + 安装 skills
# 用法: powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1 | iex"
$ErrorActionPreference = "Stop"

$repo = "yangkushu/volc-cli"
$api = "https://api.github.com/repos/$repo/releases/latest"
$release = Invoke-RestMethod -Uri $api
$asset = $release.assets | Where-Object { $_.name -eq "volc-cli-windows-amd64.exe" } | Select-Object -First 1
if (-not $asset) {
    Write-Error "未找到资产 volc-cli-windows-amd64.exe, 请检查 https://github.com/$repo/releases/latest"
    exit 1
}
$installDir = if ($env:VOLC_CLI_INSTALL_DIR) { $env:VOLC_CLI_INSTALL_DIR } else { "$env:LOCALAPPDATA\volc-cli" }
New-Item -ItemType Directory -Force -Path $installDir | Out-Null
Invoke-WebRequest -Uri $asset.browser_download_url -OutFile "$installDir\volc-cli.exe"

$skillsSrc = "https://raw.githubusercontent.com/$repo/master/skills/volc-cli/SKILL.md"
foreach ($dest in @("$HOME\.claude\skills\volc-cli", "$HOME\.codex\skills\volc-cli")) {
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    Invoke-WebRequest -Uri $skillsSrc -OutFile "$dest\SKILL.md"
    Write-Host "skill 已安装: $dest"
}

Write-Host "✔ 安装完成: $installDir\volc-cli.exe"
Write-Host "  请将 $installDir 加入 PATH, 或使用完整路径调用"
Write-Host "  凭证配置: setx VOLC_ACCESSKEY ... / setx VOLC_SECRETKEY ... / setx VOLC_CP_WORKSPACE_ID ..."
Write-Host "  验证: volc-cli check-credentials"
```

- [ ] **Step 4: 本地校验(linux 部分, 不实际下载)**

```bash
bash -n scripts/install.sh && echo "install.sh 语法 OK"
# 静态核对: 资产名与 release.yml 一致; skills 目标路径与 .claude/.codex 实际目录一致
grep -c "volc-cli-linux-amd64" scripts/install.sh .github/workflows/release.yml
grep -c "volc-cli-windows-amd64.exe" scripts/install.ps1 .github/workflows/release.yml
```

Expected: bash 语法合法; 两个资产名在脚本与 workflow 中各出现且一致.

- [ ] **Step 5: Commit**

```bash
git add skills scripts && git commit -m "feat: skills 与一键安装脚本(claude code/codex)"
```

---

### Task 12: README(安装说明 + skills 说明)

**Files:**
- Modify: `README.md`(Task 9 已创建, 本任务全量重写: 安装章节改为安装脚本方式, 增加 skills 章节)

**Interfaces:**
- Consumes: Task 9 的命令用法内容(保留), Task 10 的 release 资产名, Task 11 的安装脚本.
- Produces: 面向使用者的 README: 一键安装(自动下载指定平台最新可执行文件 + 安装 AI skills)、凭证配置、命令用法、skills 使用方式.

- [ ] **Step 1: 重写 README.md**

```markdown
# volc-cli

火山云(Volcengine) CLI 工具. 命令结构与官方 Go SDK 对齐:

    volc-cli <模块> <命令>

- 模块名 = SDK `service/` 包名小写(如 `codepipeline`)
- 命令名 = OpenAPI Action 转 kebab-case(如 `ListPipelines` → `list-pipelines`)
- flag 名 = SDK 请求字段转 kebab-case(如 `--workspace-id`)
- `--json` 输出字段名与 SDK json tag 一致(保持大写)

## 安装

### 一键安装(推荐)

自动下载指定平台的最新可执行文件, 并安装 Claude Code / Codex 的 skills:

```bash
# linux
curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh | bash

# windows(powershell)
powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1 | iex"
```

安装位置: linux 默认 `~/bin/volc-cli`(可用 `VOLC_CLI_INSTALL_DIR` 覆盖), windows 默认 `%LOCALAPPDATA%\volc-cli\volc-cli.exe`. 安装后请确认该目录在 PATH 中.

### 手动下载

GitHub Release 页: <https://github.com/yangkushu/volc-cli/releases>, 按平台下载 `volc-cli-linux-amd64` / `volc-cli-windows-amd64.exe`, 放到 PATH 目录即可.

## 凭证

优先级: flag > 环境变量.

    export VOLC_ACCESSKEY=AKxxx
    export VOLC_SECRETKEY=SKxxx
    export VOLC_CP_WORKSPACE_ID=从控制台流水线 URL 获取(/cp/workspace/{id}/pipeline/...)

验证:

    volc-cli check-credentials

region 固定 cn-north-1(持续交付仅北京 region).

## 持续交付(codepipeline)

    # 流水线列表
    volc-cli codepipeline list-pipelines

    # 发布记录(含每次发布的参数 DynamicEnvs)
    volc-cli codepipeline list-pipeline-records <流水线名或ID> --page-size 10

    # 单条记录详情(失败步骤展开错误信息)
    volc-cli codepipeline get-pipeline-record <流水线名或ID> --id <记录ID>

    # 最近一次失败(默认)/最近 N 次失败: stage→task→step 定位 + 错误信息 + 控制台 URL
    volc-cli codepipeline failures <流水线名或ID> [--page-size N]

所有命令加 `--json` 切结构化输出.

## AI skills(Claude Code / Codex)

安装脚本会自动把 `skills/volc-cli` 安装到:

- Claude Code: `~/.claude/skills/volc-cli/`
- Codex: `~/.codex/skills/volc-cli/`

装好后, 直接对 AI 说"查一下 oss 流水线最近为什么失败", AI 会调用 volc-cli 完成排查. 两者共用同一份 SKILL.md, 无需分别维护.

## 扩展新模块

1. `internal/<module>client/` 封装对应 SDK service 包(参照 cpclient)
2. `internal/cmd/<module>/` 建子命令组, 命令名/flag 名按上述对齐规则
3. `internal/cmd/root.go` AddCommand 一行挂载

## 维护约定

- 修改命令行为时同步更新其 Short/Long(flag 用法), --help 由 cobra 自动生成, 与代码同源.
- 新增 SDK 方法封装时命令名与 flag 名照 SDK 抄, 不自造名.
- 新增模块/命令后, 同步更新 `skills/volc-cli/SKILL.md` 的操作流程与 README 用法.
```

- [ ] **Step 2: 链接检查**

```bash
# 仓库内引用的路径都应存在
test -f skills/volc-cli/SKILL.md && test -f scripts/install.sh && test -f scripts/install.ps1 && test -f .github/workflows/release.yml && echo OK
```

Expected: OK.

- [ ] **Step 3: Commit**

```bash
git add README.md && git commit -m "docs: README 安装说明与 skills 用法"
```

---

## 验收清单(整体)

- [ ] `volc-cli --help` / 各级 `--help` 可用且与行为一致
- [ ] AK/SK 未设置时所有命令有明确提示且不泄露密钥内容
- [ ] `check-credentials` 对无效 key 报"凭证无效或无权限"类建议
- [ ] `list-pipelines` / `list-pipeline-records`(含 DynamicEnvs) / `get-pipeline-record` / `failures` 在真实环境可用
- [ ] `--json` 输出字段名与 SDK json tag 一致
- [ ] `go test ./...` 全绿
- [ ] release.yml 资产名与安装脚本下载 URL 一致(win/linux x64)
- [ ] 安装脚本幂等且失败即退出(linux 本地可跑 bash -n + 静态核对; 真实下载需 GitHub 首个 release 后人工验证)
- [ ] skills 目录结构与 SKILL.md frontmatter 符合 Claude Code/Codex 约定
- [ ] README 安装说明覆盖 linux/windows 一键安装与 skills 说明
