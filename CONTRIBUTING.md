# volc-cli 开发规范

> 面向维护者. 使用者请读 [README](README.md).

## 项目说明

`volc-cli` 是火山云(Volcengine)服务的命令行工具, 通过官方 Go SDK 访问火山云 OpenAPI.

- 独立仓库: <https://github.com/yangkushu/volc-cli>, 与业务服务完全解耦
- 技术栈: Go 1.26 + [cobra](https://github.com/spf13/cobra) + [volcengine-go-sdk](https://github.com/volcengine/volcengine-go-sdk)(锁定 v1.2.9)
- 首个模块: `codepipeline`(持续交付 V2, API 版本 2023-05-01), 查询流水线、运行记录与失败日志
- 镜像仓库模块: `cr`(镜像仓库, API 版本 2022-05-12), 实例/命名空间/制品仓库/版本查询与过期版本清理
- 分发: GitHub Actions 打 tag(`v*`)自动构建 win/linux x64 二进制发布 Release, 配合一键安装脚本
- AI 集成: `skills/volc-cli` 一份 SKILL.md 同时服务 Claude Code 与 Codex

> **历史坑**: 旧 SDK `volc-sdk-golang` 的 CP 封装写死 V1 API 版本(2021-03-03), 火山云已于 2025-07-31 下线 V1(官方公告 [持续交付 V1 版本下线](https://docs.volcengine.com/docs/6461/1607393)), 全部返回 502. 必须用 `volcengine-go-sdk` 的 `service/cp`(2023-05-01). 若以后加其他火山服务, 同样先确认 SDK 封装的 API 版本还在维护.

## 目录结构

```
volc-cli/
├── main.go                          # 入口, 只调 cmd.NewRootCommand().Execute()
├── internal/
│   ├── opts/opts.go                 # 全局 CLI 选项(全局 flag 绑定于此, 各命令只读)
│   ├── config/config.go             # 凭证/workspace 解析(flag > 环境变量)与脱敏
│   ├── cpclient/client.go           # service/cp 薄封装(会话/认证 + API 调用)
│   ├── crclient/client.go           # service/cr 薄封装(会话/认证 + API 调用)
│   ├── crtag/                       # cr 清理候选过滤与 PushTime 解析(纯函数, 含单测)
│   ├── failure/extract.go           # 从 PipelineRun/TaskRun 提取失败信息(纯数据提取)
│   ├── output/                      # 渲染层: text(表格/树形/日志) + json
│   └── cmd/
│       ├── root.go                  # 根命令 + 全局 flag + 子命令挂载
│       ├── check_credentials.go     # volc-cli check-credentials(全局命令)
│       ├── codepipeline/            # 模块子命令组(对应持续交付 V2 API)
│       └── cr/                      # 模块子命令组(对应镜像仓库 2022-05-12 API)
├── skills/volc-cli/SKILL.md        # Claude Code / Codex 共用 skill
├── scripts/install.sh               # linux 一键安装(幂等)
├── scripts/install.ps1              # windows 一键安装(幂等)
└── .github/workflows/release.yml    # tag v* 触发构建发布
```

分层依赖单向: `cmd` → `config`/`cpclient`/`crclient`/`crtag`/`failure`/`output`; `output` → `failure`(渲染失败信息); 底层包不反向依赖 `cmd`. `opts` 独立存放全局选项, 供 `cmd` 及子模块命令读取(避免 `cmd` ↔ 子包循环依赖).

## 命名规范

命令面与官方 SDK 逐层对齐, 看到 SDK 文档就知道 CLI 怎么敲, 不自造名:

| 层 | 规则 | 示例 |
|----|------|------|
| 模块名 | SDK `service/` 包名小写 | `service/cp` → `codepipeline` |
| 命令名 | OpenAPI Action 转 kebab-case | `ListPipelines` → `list-pipelines` |
| flag 名 | API 请求字段转 kebab-case | `WorkspaceId` → `--workspace-id`, `MaxResults` → `--page-size`(语义别名, 记录于命令 Long) |
| JSON 输出字段 | 沿用 SDK 结构字段(JSON tag 保持) | `"Id"`, `"Status"`, `"Parameters"`, `"Items"` |

便捷组合命令(非单个 Action)可用动词名, 如 `failures` = ListPipelineRuns(Filter=Failed) + ListTaskRuns + GetTaskRunLog + 失败提取.

## 全局约束

- Agent 涉及线上修改必须获得用户对具体目标和操作的明确确认, 包括触发/重跑发布等间接修改; 具体边界见 [Skill 线上修改确认规则](skills/volc-cli/SKILL.md#线上修改确认). 当前命令仅检查凭证或查询; 未来新增写操作时须落实确认流程, 不得把工具权限或 `--yes` 视为用户确认, 并同步 Skill 与 CLI 帮助.
- 凭证读取优先级: `--ak/--sk` flag > 环境变量 `VOLC_ACCESS_KEY`/`VOLC_SECRET_KEY`(SDK 会话默认凭证链读取的名字)
- WorkspaceId 优先级: `--workspace-id` flag > 环境变量(兼容旧配置); 两者都缺省时各命令自动调 ListWorkspaces 解析(唯一工作区直接用, 多个工作区报错列出候选)
- cr region 优先级: `--region` flag > 环境变量 `VOLC_CR_REGION` > 默认 `cn-north-1`; cr 删除命令须显式 `--tags` 列表 + `--yes`
- region 固定 `cn-north-1`(持续交付仅北京 region), 不加 `--region` flag
- **AK/SK 值禁止打印**到终端/日志/错误信息, 只显示是否已设置与长度(`config.MaskSecret`); 回显外部错误原文前须用 `scrubSecret` 过滤密钥值
- 每个 cobra 命令必须写 Short/Long 描述与全部 flag 用法说明(--help 由 cobra 自动生成, 与代码同源)
- Go 源文件中文注释用半角标点; commit message 中文、简短明确
- 秘密参数(Parameters 中 `Secret=true`)渲染时必须脱敏为 `***`(见 output.formatParams)

## 开发流程

```bash
# 构建与测试(核心逻辑 TDD: 先写测试确认失败, 再实现)
go build ./...
go test ./...

# 本地试跑(凭证见 README「凭证」节)
go run . --help
go run . codepipeline list-pipelines
```

新增代码须覆盖单测(config/failure/output 三包为核心逻辑层, cmd 层为薄接线不强制单测; SDK 薄封装不写单测, 真实连通性靠 `check-credentials` 手动验收).

## 扩展新模块

以接入 `service/vod` 为例:

1. `internal/vodclient/` 封装 SDK `service/vod` 包(参照 `internal/cpclient/` 的 session+client 构造)
2. `internal/cmd/vod/` 建子命令组, 命令名/flag 名按「命名规范」逐层对齐
3. `internal/cmd/root.go` 加一行 `root.AddCommand(vod.NewVodCmd())`; 若新模块需要独立的全局选项, 扩 `internal/opts` 并加对应 flag
4. 同步更新: `skills/volc-cli/SKILL.md` 操作流程、README 用法、本文档目录结构

> 接入前先确认该服务的 SDK 封装 API 版本仍在维护(参照 CP V1 下线教训).

## 发布流程

1. 确认 `go test ./...` 全绿、README 与 SKILL.md 已同步
2. 打 tag 并推送, Actions 自动构建 win/linux x64 并发布 Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

3. 到 <https://github.com/yangkushu/volc-cli/releases> 核对两个资产(`volc-cli-linux-amd64` / `volc-cli-windows-amd64.exe`)可下载
4. 验证安装脚本: 重跑一次安装应提示覆盖为新版本(幂等)

**命名约定**: Release 资产名固定 `volc-cli-{goos}-{goarch}[-.exe]`, 安装脚本按此下载; 新增平台须同步改 `.github/workflows/release.yml` 的 matrix 与 `files:` 列表、两个安装脚本.

## 维护约定

- 修改命令行为时同步更新其 Short/Long(flag 用法)
- 新增 SDK 方法封装时命令名与 flag 名照 API 字段抄
- 修改 CLI 能力后同步 `skills/volc-cli/SKILL.md`(操作流程)与 README(用法)
- 升级 `volcengine-go-sdk` 版本时, 核对 cp 服务的类型定义与行为(该 SDK 自动生成, 字段可能随版本变化)
