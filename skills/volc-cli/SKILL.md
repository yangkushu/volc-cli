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
