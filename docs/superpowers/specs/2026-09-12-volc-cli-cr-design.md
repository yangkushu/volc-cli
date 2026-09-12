# volc-cli 镜像仓库(cr)模块设计

> 2026-09-12 与用户确认的设计. 实现计划见后续 plans 文档.

## 背景与目标

volc-cli 现有 `codepipeline` 模块(纯查询). 本次接入火山云镜像仓库(Container Registry, SDK 包 `service/cr`, API 版本 2022-05-12), 提供:

- 实例/命名空间/制品仓库/镜像版本(Tag)的查询
- 过期镜像清理能力(按时间/数量/前缀圈定候选 + 显式删除)

## 范围

**做**:

- `list-registries` / `list-namespaces` / `list-repositories` / `list-tags` / `delete-tags` 五个命令
- `list-tags` 的客户端过滤参数(`--older-than`/`--keep-last`/`--tag-prefix`/`--tag-names`), 作为清理候选圈定手段

**不做**(明确排除, 用户确认):

- 不做自动批量删除: 所有删除必须基于调用者显式指定的 tag 列表(`--tags`), CLI 永不自动圈定并删除
- 不做 `latest` 等特殊 tag 的默认跳过(防呆交给调用者判断)
- 不做 `delete-repository` / `delete-namespace` / `delete-registry` / 创建/修改类命令(YAGNI, 后续有需要再加)
- 不做 tag 语义判断(线上/测试 tag 的区分由调用者通过 `--tag-prefix` 等参数自行表达)

## 命令面

模块名 `cr`(= SDK 包名). 命令名与 SDK Action 对齐(kebab-case), flag 名与 API 请求字段对齐; `list-tags` 的过滤 flag 为客户端语义扩展(Long 中记录, 同 `--page-size` 先例).

```
volc-cli cr list-registries                                          # ListRegistries
volc-cli cr list-namespaces  --registry <R>                          # ListNamespaces
volc-cli cr list-repositories --registry <R> [--namespace <N>]       # ListRepositories
volc-cli cr list-tags        --registry <R> --namespace <N> [--repository <X>]
                             [--tag-names t1,t2] [--tag-prefix ci-]
                             [--older-than 30d] [--keep-last 10]     # ListTags(+客户端过滤)
volc-cli cr delete-tags      --registry <R> --namespace <N> --repository <X>
                             --tags t1,t2 --yes                      # DeleteTags
```

### 通用规则

- `--region`(cr 组 persistent flag): CR 实例是区域性资源(北京/上海/广州/柔佛). 优先级 `--region` > 环境变量 `VOLC_CR_REGION` > 默认 `cn-north-1`(与 codepipeline 默认 region 一致)
- `--registry` 解析对齐 workspace 自动解析先例: 未指定时自动 `ListRegistries`, 唯一实例直接用, 多实例报错列出候选; `list-registries` 本身不需要
- 分页: CLI 内部翻页拉全(PageSize=100 循环), 不暴露分页 flag(与 codepipeline 行为一致)
- `--json`: 单 repo 查询输出 SDK 结构原样字段; 多 repo 聚合输出 `[{Repository, Tags}]`; delete-tags 输出汇总 Successes/Failures

### list-tags 过滤参数

- `--older-than 30d`: PushTime 早于 N 天前; 支持 `Nd`/`Nh` 两种格式, 非法报错
- `--keep-last 10`: 每个 repo 按 PushTime 降序保留最近 N 个, 其余为候选
- `--tag-prefix ci-`: tag 名前缀匹配
- `--tag-names t1,t2`: 精确匹配指定 tag
- 多条件为**交集**; 至少一个过滤条件时输出候选语义(表格含 REASON 列说明命中规则); 无过滤条件时退化为普通列表
- `--repository` 省略时遍历该 namespace 全部制品仓库(只读聚合, 服务清理场景全量扫描)

### delete-tags 防护

- `--tags` 逗号分隔显式列表; 缺 `--yes` 时报错退出并回显目标清单
- 超过 20 个自动按 20/批切分调用 DeleteTags(对调用者透明)
- 分批策略: 顺序调用; **请求级失败即止**(报告已完成批次与未执行清单, 避免权限/网络错误下继续打 API); 批内逐条 Failures 继续
- 输出逐条 Successes/Failures; 存在失败以非零码退出

## 代码结构

沿 CONTRIBUTING「扩展新模块」流程, 分层依赖单向不变:

```
internal/
├── opts/opts.go          # 扩展 Region 字段(cr 组 flag 绑定)
├── config/config.go      # 扩展 ResolveRegion: flag > VOLC_CR_REGION > cn-north-1
├── crclient/client.go    # service/cr 薄封装: New(region) + 各 List* 翻页拉全版 + DeleteTags 单批
├── crtag/                # 核心逻辑包(对齐 failure 包定位, 纯数据处理可测)
│   ├── filter.go         # older-than/keep-last/tag-prefix/tag-names 交集判定 + REASON 生成
│   ├── time.go           # --older-than 参数解析 + PushTime 容错解析(RFC3339→unix 秒兜底)
│   └── *_test.go
├── output/               # 扩展: registries/namespaces/repositories/tags 表格 + delete 结果渲染
└── cmd/cr/
    ├── cr.go             # 父命令 + --region persistent flag + resolveRegistry
    ├── list_registries.go / list_namespaces.go / list_repositories.go
    ├── list_tags.go      # 翻页聚合 + 多 repo 遍历 + crtag 过滤
    └── delete_tags.go    # --yes 检查 + 分批 + 结果汇总
```

`root.go` 挂载 `root.AddCommand(cr.NewCRCmd())`.

## 关键实现决策

1. **PushTime 安全语义**: PushTime 解析失败的 tag **永不进入候选**(宁可漏删不可错删), 列表标注 `PushTime未知`; `--keep-last` 排序时同样排除其出候选资格. 时间比较统一 UTC; PushTime 无时区信息按 UTC 处理
2. **翻页在 crclient**(如 `ListTagsAll`), cmd 层保持薄接线
3. **PushTime 实际格式未在文档中确认**(官方文档 JS 渲染抓取不到): 解析实现容错(RFC3339 优先, unix 秒兜底), 真实格式在手动验收 `list-registries`/`list-tags` 时校准
4. **密钥安全**: 错误回显沿用 scrubSecret 约束, 不含 AK/SK 值
5. SDK 版本不变(v1.2.9 已内置 `service/cr`)

## 测试与验收

- TDD 单测: `crtag`(过滤/时间解析, 表驱动覆盖 PushTime 多格式/组合条件/未知时间/边界值)、`config.ResolveRegion`、`output` 新增 Print 函数
- `crclient`/cmd 层按惯例不写单测; 真实连通性手动验收
- 手动验收: 真实凭证跑 `list-registries`(region 行为/翻页)、`list-tags`(PushTime 真实格式校准)、`delete-tags`(先删测试 tag)

## 文档同步

- `root.go` Long: 支持模块清单加 cr; **「当前仅查询命令」表述更新**——cr 引入首个写操作 `delete-tags`, 线上修改确认规则实际生效
- `skills/volc-cli/SKILL.md`: 触发条件扩展(镜像仓库查询/清理); cr 清理工作流 = `list-tags` 圈候选 → 展示用户并获明确确认 → `delete-tags --tags <显式列表> --yes`
- README 用法、CONTRIBUTING 目录结构
