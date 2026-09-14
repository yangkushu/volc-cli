---
name: volc-cli
description: 查询火山云持续交付(CodePipeline)流水线的运行记录与失败原因. 用于排查流水线构建/发布失败、查看某次发布的参数、或列出工作区全部流水线. 也支持镜像仓库(cr)实例/命名空间/制品仓库/版本查询与过期版本清理(显式列表删除).
allowed-tools: Bash(volc-cli:*)
---

# volc-cli 流水线排查

## 触发条件

- 用户提到流水线/构建失败, 要查失败原因或错误日志
- 用户要查某条流水线的运行记录、发布参数
- 用户要列出火山云持续交付工作区的流水线
- 用户要查火山云镜像仓库的实例/命名空间/仓库/版本, 或要清理过期镜像版本

## 前置要求

- 已安装 volc-cli(见仓库 README 安装章节)
- 环境变量已配置: `VOLC_ACCESS_KEY`、`VOLC_SECRET_KEY`
- WorkspaceId 无需配置, 命令会自动解析(多工作区时按提示加 --workspace-id)
- 未配置凭证时先跑 `volc-cli check-credentials` 看提示, 不要猜测

## 线上修改确认

- 涉及线上/生产环境(production)的修改, 必须获得用户对具体操作的明确确认后才能执行. 包括直接修改资源、配置、权限或数据, 以及触发/重跑发布流水线、部署、回滚、重启等间接修改.
- 执行前先确定目标环境与影响范围; 无法确定是否影响线上时, 先做只读核实, 仍不明确则询问用户, 暂停修改.
- 请求确认时说明目标环境、资源/流水线、具体操作与预期影响, 等待用户明确同意. “排查一下”“修好它”等笼统请求、工具执行权限、`--yes` 参数或用户未回复都不算确认.
- 用户在当前对话中已明确确认同一环境、目标和操作时可直接执行; 目标、操作或影响范围发生变化须重新确认. 不得换用其他 CLI、API 或控制台绕过确认.
- 只读查询(包括线上日志、运行记录、流水线与工作区列表)无需此类确认. 当前 CLI 的写操作仅 cr delete-tags(删除镜像版本), 执行前必须先把删除清单展示给用户并获得明确确认.

## 操作流程

1. 先看环境是否就绪: `volc-cli check-credentials`; 报错时按输出提示修复
2. 查失败原因: `volc-cli codepipeline failures <流水线名或ID> [--page-size N]`
   - 默认返回最后一次失败运行, 输出 stage→task→step 定位、失败日志尾部、控制台 URL
3. 查运行记录与参数: `volc-cli codepipeline list-pipeline-runs <流水线名或ID> [--page-size N]`
4. 查指定运行的 step 日志: `volc-cli codepipeline get-task-run-log <流水线名或ID> --run-id <运行ID> [--tail N]`
5. 列流水线(不知道名称/ID 时): `volc-cli codepipeline list-pipelines`
6. 列工作区(不知道 WorkspaceId 时): `volc-cli codepipeline list-workspaces`
7. 镜像仓库查询: `volc-cli cr list-registries`(region 默认 cn-beijing, 实例如在其他 region 用 --region 或 VOLC_CR_REGION 切换)
8. 列命名空间/仓库/版本: `volc-cli cr list-namespaces` / `list-repositories --namespace N` / `list-tags --namespace N [--repository X]`
9. 清理过期版本(两步, 不自动批量删):
   - 圈候选: `volc-cli cr list-tags --namespace N [--repository X] --older-than 30d [--keep-last 10] [--tag-prefix ci-]`
   - 把候选清单展示给用户, 获得对具体 tag 列表的明确确认后执行:
     `volc-cli cr delete-tags --namespace N --repository X --tags t1,t2 --yes`

## 关键规则

- 流水线定位失败(提示"未找到流水线")时, 先 `list-pipelines` 拿真实名称, 不要猜
- workspace-id 未设置报错时, 先 `list-workspaces` 拿真实 ID, 不要猜
- 失败信息在失败 step 的日志尾部里; 日志为空时, 把控制台 URL 给用户, 并说明完整日志在控制台
- AK/SK 是敏感信息: 永远不要把 key 内容写进命令输出、日志或对话
- 给用户排查结论时, 先给定位(stage/task/step), 再给错误日志原文, 最后给控制台链接
- cr 删除必须基于用户确认过的显式 tag 列表(`--tags`), 不做自动批量删除; 先 list-tags 圈候选再人工确认
- PushTime 显示"(未知)"的版本不参与 older-than/keep-last 候选(宁漏删不错删), 处置需用户单独确认
