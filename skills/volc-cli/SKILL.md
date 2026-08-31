---
name: volc-cli
description: 查询火山云持续交付(CodePipeline)流水线的运行记录与失败原因. 用于排查流水线构建/发布失败、查看某次发布的参数、或列出工作区全部流水线.
allowed-tools: Bash(volc-cli:*)
---

# volc-cli 流水线排查

## 触发条件

- 用户提到流水线/构建失败, 要查失败原因或错误日志
- 用户要查某条流水线的运行记录、发布参数
- 用户要列出火山云持续交付工作区的流水线

## 前置要求

- 已安装 volc-cli(见仓库 README 安装章节)
- 环境变量已配置: `VOLC_ACCESS_KEY`、`VOLC_SECRET_KEY`
- WorkspaceId 无需配置, 命令会自动解析(多工作区时按提示加 --workspace-id)
- 未配置凭证时先跑 `volc-cli check-credentials` 看提示, 不要猜测

## 操作流程

1. 先看环境是否就绪: `volc-cli check-credentials`; 报错时按输出提示修复
2. 查失败原因: `volc-cli codepipeline failures <流水线名或ID> [--page-size N]`
   - 默认返回最后一次失败运行, 输出 stage→task→step 定位、失败日志尾部、控制台 URL
3. 查运行记录与参数: `volc-cli codepipeline list-pipeline-runs <流水线名或ID> [--page-size N]`
4. 查指定运行的 step 日志: `volc-cli codepipeline get-task-run-log <流水线名或ID> --run-id <运行ID> [--tail N]`
5. 列流水线(不知道名称/ID 时): `volc-cli codepipeline list-pipelines`
6. 列工作区(不知道 WorkspaceId 时): `volc-cli codepipeline list-workspaces`

## 关键规则

- 流水线定位失败(提示"未找到流水线")时, 先 `list-pipelines` 拿真实名称, 不要猜
- workspace-id 未设置报错时, 先 `list-workspaces` 拿真实 ID, 不要猜
- 失败信息在失败 step 的日志尾部里; 日志为空时, 把控制台 URL 给用户, 并说明完整日志在控制台
- AK/SK 是敏感信息: 永远不要把 key 内容写进命令输出、日志或对话
- 给用户排查结论时, 先给定位(stage/task/step), 再给错误日志原文, 最后给控制台链接
