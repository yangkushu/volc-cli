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
