# volc-cli

火山云(Volcengine) CLI 工具. 命令结构与官方 OpenAPI 对齐:

    volc-cli <模块> <命令>

- 模块名 = SDK `service/` 包名小写(如 `codepipeline` = service/cp, 持续交付)
- 镜像仓库(cr)模块对应 SDK `service/cr`(API 版本 2022-05-12), 命令如 `ListRegistries` → `list-registries`
- 命令名 = OpenAPI Action 转 kebab-case(如 `ListPipelines` → `list-pipelines`)
- flag 名 = API 请求字段转 kebab-case(如 `--workspace-id`)
- `--json` 输出字段名与 SDK 结构一致

持续交付模块基于 **V2 OpenAPI(API 版本 2023-05-01, 官方 Go SDK `volcengine-go-sdk`)**. 旧版 V1 API 已于 2025-07-31 下线.

完整命名规范与开发约定见 [CONTRIBUTING.md](CONTRIBUTING.md).

## 安装

### 安装 CLI（Release 脚本，推荐）

自动下载最新 Release 的可执行文件。脚本默认只安装 CLI，不修改 Claude Code、Codex 或 Cursor 的 Skill 配置。

```bash
# linux
curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh | bash

# windows(powershell)
powershell -ExecutionPolicy Bypass -Command "irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1 | iex"
```

安装位置：Linux/macOS 默认 `~/bin/volc-cli`（可用 `VOLC_CLI_INSTALL_DIR` 覆盖），Windows 默认 `%LOCALAPPDATA%\volc-cli\volc-cli.exe`。安装后请确认该目录在 PATH 中。

支持的平台和架构：Linux、macOS、Windows 的 `amd64` 与 `arm64`。脚本会检测架构；如果对应 Release 资产不存在会明确失败。重复运行会升级到最新 Release。

也可以从 [GitHub Releases](https://github.com/yangkushu/volc-cli/releases) 手动下载。资产名为 `volc-cli-<linux|darwin|windows>-<amd64|arm64>`；Windows 文件带 `.exe` 后缀。

### 安装 CLI + Skill（可选）

如需方便地一次安装 CLI 与 Claude Code、Codex、Cursor 的 Skill，可传入显式参数。Skill 从与二进制相同的 Release tag 提取，因此两者版本一致。

```bash
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.sh \
  | bash -s -- --with-skill
```

```powershell
# Windows PowerShell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/yangkushu/volc-cli/master/scripts/install.ps1))) -WithSkill
```

这是便捷安装方式。若已使用 `npx skills` 管理 Skill，请不要混用此方式更新 Skill。

### 仅安装 Skill

仅安装 Skill 不会安装 `volc-cli` 二进制。AI 在调用前仍要求 `volc-cli` 已在 `PATH` 中；请先完成「安装 CLI」。

推荐使用 [skills CLI](https://github.com/vercel-labs/skills) 受管安装，它会维护 canonical copy 和 Agent 目录的链接：

```bash
npx skills add https://github.com/yangkushu/volc-cli \
  --skill volc-cli \
  --agent claude-code --agent codex --agent cursor \
  --global --yes
```

此命令跟随仓库的最新 Skill。若要与已安装的某个 CLI Release 严格对应，请将 `<tag>` 替换为该 Release tag（例如 `v0.1.1`）：

```bash
npx skills add https://github.com/yangkushu/volc-cli/tree/<tag>/skills/volc-cli \
  --skill volc-cli \
  --agent claude-code --agent codex --agent cursor \
  --global --yes
```

受管安装后请用 `npx skills update --global` 更新 Skill，不要再以手工 `curl` 覆盖该 Skill 目录。若链接异常，先运行 `npx skills list --global` 确认安装状态，再按其输出修复。

### 安装模式速览

| 场景 | CLI 二进制 | Claude Code / Codex / Cursor Skill |
| --- | --- | --- |
| Release 安装脚本（默认） | 安装 | 不安装 |
| Release 安装脚本 + `--with-skill` / `-WithSkill` | 安装 | 安装，且与 Release tag 一致 |
| `npx skills add ...` | 不安装 | 安装 |

### 源码编译

```bash
git clone git@github.com:yangkushu/volc-cli.git
cd volc-cli && go build -o volc-cli .
```

开发规范(命名规则、目录结构、扩展新模块)见 [CONTRIBUTING.md](CONTRIBUTING.md).

## 凭证

优先级: flag > 环境变量.

    export VOLC_ACCESS_KEY=AKxxx
    export VOLC_SECRET_KEY=SKxxx

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

## 镜像仓库(cr)

对应镜像仓库 OpenAPI(API 版本 2022-05-12). 模块级 flag 对全部子命令生效:

    --region    实例所在 region. 优先级: --region flag > 环境变量 VOLC_CR_REGION > 默认 cn-north-1
    --registry  镜像仓库实例名. 未指定时自动解析: 当前 region 唯一实例直接用; 多实例报错列出候选, 可先跑 list-registries 查看

    # 列出当前 region 全部镜像仓库实例(无实例时检查 --region)
    volc-cli cr list-registries

    # 列出实例下全部命名空间
    volc-cli cr list-namespaces

    # 列出命名空间下全部 OCI 制品仓库(--namespace 支持逗号分隔多个, 省略时列出全部)
    volc-cli cr list-repositories [--namespace N]

    # 列出制品仓库全部版本(--repository 省略时遍历该命名空间下全部仓库, 只读聚合)
    volc-cli cr list-tags --namespace N [--repository X]

    # 删除指定版本(必须显式列表 + --yes, 不可恢复)
    volc-cli cr delete-tags --namespace N --repository X --tags t1,t2 --yes

清理候选过滤(list-tags 客户端交集过滤, 非 API 字段; 命中项带 REASON 列):

    --older-than 30d   PushTime 早于 N 天前(支持 Nd/Nh)
    --keep-last 10     每个仓库按 PushTime 降序保留最近 N 个, 其余为候选
    --tag-prefix ci-   tag 名前缀匹配
    --tag-names t1,t2  tag 名精确匹配

清理过期版本是两步流, CLI 不做自动批量删除:

    # 第一步: 圈候选(只读)
    volc-cli cr list-tags --namespace N --repository X --older-than 30d --keep-last 10

    # 第二步: 人工确认候选清单后, 对显式列表执行删除
    volc-cli cr delete-tags --namespace N --repository X --tags t1,t2 --yes

删除安全说明:

- `delete-tags` 必须显式给出逗号分隔的 `--tags` 精确列表并加 `--yes`, 缺一即报错退出, 不做任何自动圈定
- 删除不可恢复; 超过 20 个自动分批调用, 请求级失败即止并报告进度, 逐条失败不影响其余, 存在失败时以非零码退出并输出已删除/失败/未执行清单
- PushTime 无法解析(表格显示"(未知)")的版本永不进入清理候选(宁漏删不错删), 需人工单独处置

所有命令加 `--json` 切结构化输出(list-tags 多仓库聚合时 JSON 按仓库分组).

## AI Skill（Claude Code / Codex / Cursor）

Agent 使用 Skill 或 CLI 涉及线上/生产环境修改时, 必须先说明目标、操作和预期影响, 并获得用户明确确认. 发布、重跑发布流水线、回滚等间接修改也适用; 环境不明时先核实, 仍不明确则暂停修改并询问. 笼统的“修好它”、工具权限或 `--yes` 不代替确认. 已明确确认的同一操作无需重复询问, 范围变化须重新确认. 只读查询无需此类确认. 完整规则见 [SKILL.md](skills/volc-cli/SKILL.md#线上修改确认).

当前 CLI 的写操作仅 cr delete-tags(删除镜像版本), 执行前必须先把删除清单展示给用户并获得明确确认; 上述规则是 Agent 操作约束, CLI 帮助会提示该要求, 并非运行时审批机制.

Release 安装脚本传入 `--with-skill`（PowerShell 为 `-WithSkill`）时，会把完整的 `skills/volc-cli` 目录安装到：

- Claude Code: `~/.claude/skills/volc-cli/`
- Codex: `~/.codex/skills/volc-cli/`
- Cursor: `~/.cursor/skills/volc-cli/`

通过 `npx skills` 安装时，它会维护共享副本与各 Agent 目录的链接；请使用该工具更新，避免混用两套安装方式。装好 CLI 与 Skill 后，直接对 AI 说“查一下 oss 流水线最近为什么失败”，AI 会调用 `volc-cli` 完成排查。

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
