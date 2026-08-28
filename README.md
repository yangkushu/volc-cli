# volc-cli

火山云(Volcengine) CLI 工具. 命令结构与官方 OpenAPI 对齐:

    volc-cli <模块> <命令>

- 模块名 = SDK `service/` 包名小写(如 `codepipeline` = service/cp, 持续交付)
- 命令名 = OpenAPI Action 转 kebab-case(如 `ListPipelines` → `list-pipelines`)
- flag 名 = API 请求字段转 kebab-case(如 `--workspace-id`)
- `--json` 输出字段名与 SDK 结构一致

持续交付模块基于 **V2 OpenAPI(API 版本 2023-05-01, 官方 Go SDK `volcengine-go-sdk`)**. 旧版 V1 API 已于 2025-07-31 下线.

完整命名规范与开发约定见 [CONTRIBUTING.md](CONTRIBUTING.md).

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

脚本幂等: 重复运行会重新下载最新版覆盖旧版, 不会残留多余文件. 下载失败会明确报错并提示检查 <https://github.com/yangkushu/volc-cli/releases/latest>.

### 手动下载

GitHub Release 页: <https://github.com/yangkushu/volc-cli/releases>, 按平台下载 `volc-cli-linux-amd64` / `volc-cli-windows-amd64.exe`, 放到 PATH 目录即可.

### 手动安装 skills(可选, 不装不影响 CLI 使用)

```bash
# linux / mac: 一条命令装到 Claude Code 与 Codex
mkdir -p ~/.claude/skills/volc-cli ~/.codex/skills/volc-cli \
  && curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/skills/volc-cli/SKILL.md -o ~/.claude/skills/volc-cli/SKILL.md \
  && cp ~/.claude/skills/volc-cli/SKILL.md ~/.codex/skills/volc-cli/SKILL.md
```

### 源码编译

```bash
git clone git@github.com:yangkushu/volc-cli.git
cd volc-cli && go build -o volc-cli .
```

开发规范(命名规则、目录结构、扩展新模块)见 [CONTRIBUTING.md](CONTRIBUTING.md).

## 凭证

优先级: flag > 环境变量.

    export VOLC_ACCESSKEY=AKxxx
    export VOLC_SECRETKEY=SKxxx

WorkspaceId 无需手动配置: 命令会通过 `ListWorkspaces` 自动解析(账户下唯一工作区直接用; 多个工作区时按提示用 `--workspace-id` 指定).

    volc-cli codepipeline list-workspaces      # 手动查看/确认工作区

验证:

    volc-cli check-credentials

region 固定 cn-north-1(持续交付仅北京 region).

**安全提示**: AK/SK 是敏感信息. 工具只显示是否已设置与长度, 不会打印密钥内容; 请勿把密钥写进脚本或提交到仓库.

## 持续交付(codepipeline)

    # 工作区列表(获取 workspace-id, 无需凭证以外的配置)
    volc-cli codepipeline list-workspaces

    # 流水线列表
    volc-cli codepipeline list-pipelines

    # 执行记录(含每次发布的参数, 秘密参数脱敏)
    volc-cli codepipeline list-pipeline-runs <流水线名或ID> [--page-size N] [--status Failed]

    # 查询某次运行的 step 日志(--run-id 从执行记录拿, 默认自动定位失败 step)
    volc-cli codepipeline get-task-run-log <流水线名或ID> --run-id <运行ID> [--tail N]

    # 最近一次失败(默认)/最近 N 次失败: stage→task→step 定位 + 失败日志尾部 + 控制台 URL
    volc-cli codepipeline failures <流水线名或ID> [--page-size N] [--tail N]

所有命令加 `--json` 切结构化输出.

## AI skills(Claude Code / Codex)

安装脚本会自动把 `skills/volc-cli` 安装到:

- Claude Code: `~/.claude/skills/volc-cli/`
- Codex: `~/.codex/skills/volc-cli/`

装好后, 直接对 AI 说"查一下 oss 流水线最近为什么失败", AI 会调用 volc-cli 完成排查. 两者共用同一份 SKILL.md, 无需分别维护. 更新 skill 重跑安装脚本即可.

## 扩展新模块

1. `internal/<module>client/` 封装对应 SDK service 包(参照 cpclient)
2. `internal/cmd/<module>/` 建子命令组, 命令名/flag 名按上述对齐规则
3. `internal/cmd/root.go` AddCommand 一行挂载

完整步骤见 [CONTRIBUTING.md](CONTRIBUTING.md)「扩展新模块」.

## 维护约定

- 修改命令行为时同步更新其 Short/Long(flag 用法), --help 由 cobra 自动生成, 与代码同源.
- 新增 SDK 方法封装时命令名与 flag 名照 SDK 抄, 不自造名.
- 新增模块/命令后, 同步更新 `skills/volc-cli/SKILL.md` 的操作流程与 README 用法.
- 详细开发规范(命名规则、目录结构、发布流程)见 [CONTRIBUTING.md](CONTRIBUTING.md).
