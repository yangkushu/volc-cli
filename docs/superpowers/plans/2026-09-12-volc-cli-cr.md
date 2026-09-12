# volc-cli 镜像仓库(cr)模块实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 volc-cli 新增 `cr` 模块: 镜像仓库实例/命名空间/制品仓库/版本(Tag)查询, 清理候选圈定(时间/数量/前缀过滤), 与显式 tag 列表删除.

**Architecture:** cobra 子命令组 `volc-cli cr <命令>`, 命令名与 SDK Action 对齐(kebab-case). 分层沿现有结构: `crclient`(SDK 薄封装+翻页) / `crtag`(纯数据过滤逻辑, TDD) / `output`(渲染) / `cmd/cr`(薄接线). 所有删除基于显式 `--tags` 列表, CLI 不做自动批量删除.

**Tech Stack:** Go 1.26, cobra v1.9.x, volcengine-go-sdk v1.2.9(已内置 `service/cr`, API 版本 2022-05-12, 不升版本)

**Spec:** `docs/superpowers/specs/2026-09-12-volc-cli-cr-design.md`

## Global Constraints

- 项目目录: `/home/yangzeqi/workspaces/kuopin/volc-cli`, 工作分支 master(项目惯例直接提交 master)
- 模块名 = SDK 包名小写: `cr`(对应 `service/cr`); 命令名 = Action kebab-case; flag 名 = API 请求字段 kebab-case; `--json` 输出字段保持 SDK 原样大写
- `list-tags` 的 `--older-than/--keep-last/--tag-prefix/--tag-names` 是客户端语义扩展 flag(命令 Long 中说明), 同现有 `--page-size` 别名先例
- 凭证: `--ak/--sk` > 环境变量 `VOLC_ACCESS_KEY`/`VOLC_SECRET_KEY`(SDK 默认链)
- region: `--region`(cr 组 persistent flag) > `VOLC_CR_REGION` > 默认 `cn-north-1`
- `--registry`(cr 组 persistent flag) 未指定时自动 `ListRegistries`: 唯一实例直接用, 多实例报错列出候选
- 删除安全: `delete-tags` 必须显式 `--tags`(逗号分隔)+`--yes`; 超 20 个自动按 20/批切分; 请求级失败即止并报告进度; 批内 Failures 继续; 存在失败以非零码退出; 不做 latest 默认跳过、不做自动圈定
- PushTime 解析失败的 tag 永不进入清理候选(宁漏删不错删)
- AK/SK 值禁止打印到终端/日志/错误
- Go 中文注释用半角标点; commit message 中文简短; 不加 Co-Authored-By 署名
- TDD: `crtag`/`config`/`output` 先写测试; `crclient`/`cmd` 薄接线不写单测

## SDK 关键事实(实现时直接引用, 无需再翻源码)

路径: `$(go env GOMODCACHE)/github.com/volcengine/volcengine-go-sdk@v1.2.9/service/cr/`

- 构造(同 cp 模式): `cfg := volcengine.NewConfig().WithRegion(region)`; ak/sk 非空时 `cfg.WithAkSk(ak, sk)`; `sess, err := session.NewSession(cfg)`; `cr.New(sess, cfg)` 返回 `*cr.CR`
- 方法(全部有 `XxxWithContext(ctx, input)` 形态): `ListRegistries`, `ListNamespaces`, `ListRepositories`, `ListTags`, `DeleteTags`
- 输入/输出字段(v1.2.9 实测确认, 全部指针类型):
  - `ListRegistriesInput{PageNumber *int64, PageSize *int64 /*1..100*/, Filter, ResourceTagFilters}`; `ListRegistriesOutput{Items []*ItemForListRegistriesOutput, TotalCount *int64}`
  - `ItemForListRegistriesOutput{Name, Type, ChargeType, CreateTime, ExpireTime, Project *string, Status *StatusForListRegistriesOutput{Phase *string, Conditions []*string}, ...}`
  - `ListNamespacesInput{Registry *string /*必填,3..30字符*/, PageNumber, PageSize *int64, Filter}`; `ListNamespacesOutput{Items []*ItemForListNamespacesOutput, TotalCount *int64}`
  - `ItemForListNamespacesOutput{Name, CreateTime, Project *string}`
  - `ListRepositoriesInput{Registry *string /*必填*/, PageNumber, PageSize *int64, Filter *FilterForListRepositoriesInput{Namespaces, Names, AccessLevels []*string}}`; `ListRepositoriesOutput{Items []*ItemForListRepositoriesOutput, TotalCount *int64}`
  - `ItemForListRepositoriesOutput{Name, Namespace, AccessLevel, CreateTime, UpdateTime, Description *string}`
  - `ListTagsInput{Registry, Namespace /*必填,2..90*/, Repository *string /*必填*/, PageNumber, PageSize *int64 /*1..100*/, Filter *FilterForListTagsInput{Names, Types []*string}}`; `ListTagsOutput{Items []*ItemForListTagsOutput, TotalCount *int64}`
  - `ItemForListTagsOutput{Name, Digest, PushTime, Type *string, Size *int64, ChartAttribute, ImageAttributes}`
  - `DeleteTagsInput{Registry, Namespace, Repository *string /*必填*/, Names []*string /*官方限制单次最多 20 个*/}`; `DeleteTagsOutput{Successes []*SuccessForDeleteTagsOutput{Name *string}, Failures []*FailureForDeleteTagsOutput{Name, Reason *string}}`
- 取值 helper: `volcengine.StringValue(*string)` / `volcengine.Int64Value(*int64)`

## File Structure

```
internal/
├── opts/opts.go            # 修改: 加 Region, Registry 字段
├── config/config.go        # 修改: 加 EnvCRRegion + ResolveRegion
├── crclient/client.go      # 新建: SDK 薄封装(构造+翻页拉全+单批删除)
├── crtag/
│   ├── time.go             # 新建: ParseOlderThan + ParsePushTime
│   ├── time_test.go
│   ├── filter.go           # 新建: Criteria + TagView + Filter
│   └── filter_test.go
├── output/
│   ├── cr.go               # 新建: cr 各表格渲染 + formatSize
│   └── cr_test.go
└── cmd/cr/
    ├── cr.go               # 新建: 父命令 + --region/--registry + newClient + resolveRegistry
    ├── list_registries.go  # 新建
    ├── list_namespaces.go  # 新建
    ├── list_repositories.go# 新建
    ├── list_tags.go        # 新建: 多 repo 遍历 + 过滤 + JSON 聚合
    └── delete_tags.go      # 新建: --yes 校验 + 20/批 + 汇总
修改: internal/cmd/root.go(挂载+Long), README.md, skills/volc-cli/SKILL.md, CONTRIBUTING.md
```

---

### Task 1: crtag 时间解析(ParseOlderThan / ParsePushTime)

**Files:**
- Create: `internal/crtag/time.go`
- Test: `internal/crtag/time_test.go`

**Interfaces:**
- Produces: `crtag.ParseOlderThan(s string) (time.Duration, error)`; `crtag.ParsePushTime(s string) (time.Time, bool)` — Task 2 的 Filter 依赖 ParsePushTime

- [ ] **Step 1: 写失败测试**

`internal/crtag/time_test.go`:

```go
package crtag

import (
	"testing"
	"time"
)

func TestParseOlderThan(t *testing.T) {
	cases := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"12h", 12 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"0h", 0, false},
		{"100d", 100 * 24 * time.Hour, false},
		{"30", 0, true},
		{"d", 0, true},
		{"30m", 0, true},
		{"30w", 0, true},
		{"-1d", 0, true},
		{"", 0, true},
		{"3.5d", 0, true},
	}
	for _, c := range cases {
		got, err := ParseOlderThan(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseOlderThan(%q) 应报错, got %v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseOlderThan(%q) 意外报错: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseOlderThan(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParsePushTime(t *testing.T) {
	cases := []struct {
		in  string
		want time.Time
		ok  bool
	}{
		// RFC3339
		{"2026-09-01T10:00:00Z", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), true},
		{"2026-09-01T10:00:00+08:00", time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC), true},
		// 无时区空格分隔, 按 UTC
		{"2026-09-01 10:00:00", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC), true},
		// Unix 秒 / 毫秒
		{"1760000000", time.Unix(1760000000, 0).UTC(), true},
		{"1760000000123", time.Unix(1760000000, 123000000).UTC(), true},
		// 失败
		{"", time.Time{}, false},
		{"not-a-time", time.Time{}, false},
		{"2026-09-01", time.Time{}, false},
	}
	for _, c := range cases {
		got, ok := ParsePushTime(c.in)
		if ok != c.ok {
			t.Errorf("ParsePushTime(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && !got.Equal(c.want) {
			t.Errorf("ParsePushTime(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/crtag/`
Expected: 编译失败, `undefined: ParseOlderThan` / `undefined: ParsePushTime`

- [ ] **Step 3: 实现**

`internal/crtag/time.go`:

```go
// Package crtag 实现镜像 Tag 清理候选的圈定逻辑(纯数据处理, 不做 IO).
package crtag

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var olderThanRe = regexp.MustCompile(`^(\d+)([dh])$`)

// ParseOlderThan 解析 --older-than 参数, 仅支持 Nd(天)/Nh(小时), 如 30d/12h.
func ParseOlderThan(s string) (time.Duration, error) {
	m := olderThanRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("--older-than 格式非法: %q, 仅支持如 30d(天)/12h(小时)", s)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("--older-than 数字非法: %q", s)
	}
	d := time.Duration(n) * time.Hour
	if m[2] == "d" {
		d *= 24
	}
	return d, nil
}

// ParsePushTime 容错解析 SDK 返回的 PushTime 字符串, 依次尝试:
// RFC3339 / "2006-01-02 15:04:05"(按 UTC) / Unix 秒(13 位按毫秒).
// 全部失败返回 ok=false; 该类 tag 永不进入清理候选(宁可漏删不可错删).
func ParsePushTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), true
	}
	if sec, err := strconv.ParseInt(s, 10, 64); err == nil && sec > 0 {
		if sec > 1e12 { // 13 位毫秒时间戳
			return time.Unix(sec/1e3, (sec%1e3)*1e6).UTC(), true
		}
		return time.Unix(sec, 0).UTC(), true
	}
	return time.Time{}, false
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/crtag/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/crtag/
git commit -m "feat: crtag OlderThan/PushTime 解析"
```

---

### Task 2: crtag 清理候选过滤(Criteria / TagView / Filter)

**Files:**
- Create: `internal/crtag/filter.go`
- Test: `internal/crtag/filter_test.go`

**Interfaces:**
- Consumes: Task 1 的 `ParsePushTime`
- Produces: `crtag.Criteria{OlderThan time.Duration; KeepLast int; TagPrefix string; TagNames []string; Now time.Time}`; `crtag.TagView{Tag *cr.ItemForListTagsOutput; Reason string}`; `crtag.Filter(tags []*cr.ItemForListTagsOutput, c Criteria) []TagView` — Task 7 的 list-tags 依赖

- [ ] **Step 1: 写失败测试**

`internal/crtag/filter_test.go`:

```go
package crtag

import (
	"strings"
	"testing"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
)

func sp(s string) *string { return &s }

func mkTag(name, pushTime string) *cr.ItemForListTagsOutput {
	return &cr.ItemForListTagsOutput{Name: sp(name), PushTime: sp(pushTime), Type: sp("Image")}
}

var now = time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)

func names(vs []TagView) []string {
	var out []string
	for _, v := range vs {
		out = append(out, *v.Tag.Name)
	}
	return out
}

func TestFilterOlderThan(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("old", "2026-07-01T00:00:00Z"),  // 73 天前
		mkTag("recent", "2026-09-10T00:00:00Z"), // 2 天前
		mkTag("edge", "2026-08-13T00:00:00Z"),  // 恰好 30 天前(不早于阈值, 不候选)
	}
	got := names(Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, Now: now}))
	if len(got) != 1 || got[0] != "old" {
		t.Errorf("older-than 30d 应只圈定 old, got %v", got)
	}
	// Reason 应含规则说明
	vs := Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, Now: now})
	if !strings.Contains(vs[0].Reason, "older-than") {
		t.Errorf("Reason 应说明 older-than 命中, got %q", vs[0].Reason)
	}
}

func TestFilterKeepLast(t *testing.T) {
	var tags []*cr.ItemForListTagsOutput
	for i := 11; i >= 0; i-- { // 12 个 tag, push 时间递增
		tags = append(tags, mkTag(string(rune('a'+i)), "2026-09-01T00:00:0"+string(rune('0'+i%10))+"Z"))
	}
	// 直接构造明确时间, 避免上面拼接出错
	tags = nil
	for i := 0; i < 12; i++ {
		tags = append(tags, mkTag(string(rune('a'+i)), time.Date(2026, 9, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)))
	}
	got := names(Filter(tags, Criteria{KeepLast: 10, Now: now}))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("keep-last 10 应圈定最早的 a,b, got %v", got)
	}
}

func TestFilterKeepLastUnknownPushTimeNeverCandidate(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("a", "2026-09-01T00:00:00Z"),
		mkTag("b", "2026-09-02T00:00:00Z"),
		mkTag("bad", "garbage"), // PushTime 无法解析
	}
	got := names(Filter(tags, Criteria{KeepLast: 1, Now: now}))
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("PushTime 未知的 bad 永不候选, keep-last 1 只圈定 a, got %v", got)
	}
}

func TestFilterIntersection(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("ci-old1", "2026-07-01T00:00:00Z"),
		mkTag("ci-old2", "2026-07-02T00:00:00Z"),
		mkTag("ci-new", "2026-09-10T00:00:00Z"),
		mkTag("rel-old", "2026-07-03T00:00:00Z"), // 老但无 ci- 前缀
	}
	got := names(Filter(tags, Criteria{
		OlderThan: 30 * 24 * time.Hour,
		TagPrefix: "ci-",
		Now:       now,
	}))
	if len(got) != 2 || got[0] != "ci-old1" || got[1] != "ci-old2" {
		t.Errorf("older-than+prefix 交集应圈定 ci-old1/ci-old2, got %v", got)
	}
}

func TestFilterKeepLastAndOlderThanCombined(t *testing.T) {
	// 12 个老 tag(全部早于 30d) + keep-last 10 → 只圈定最早 2 个
	var tags []*cr.ItemForListTagsOutput
	for i := 0; i < 12; i++ {
		tags = append(tags, mkTag(string(rune('a'+i)), time.Date(2026, 8, 1, 0, 0, i, 0, time.UTC).Format(time.RFC3339)))
	}
	got := names(Filter(tags, Criteria{OlderThan: 30 * 24 * time.Hour, KeepLast: 10, Now: now}))
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("组合条件应只圈定最早 2 个, got %v", got)
	}
}

func TestFilterTagNames(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("v1", "2026-09-10T00:00:00Z"),
		mkTag("v2", "2026-09-11T00:00:00Z"),
	}
	got := names(Filter(tags, Criteria{TagNames: []string{"v2"}, Now: now}))
	if len(got) != 1 || got[0] != "v2" {
		t.Errorf("tag-names 精确匹配应只圈定 v2, got %v", got)
	}
}

func TestFilterZeroCriteriaReturnsAll(t *testing.T) {
	tags := []*cr.ItemForListTagsOutput{
		mkTag("a", "2026-09-10T00:00:00Z"),
		mkTag("b", "garbage"),
	}
	vs := Filter(tags, Criteria{Now: now})
	if len(vs) != 2 {
		t.Fatalf("零值条件应返回全部, got %d", len(vs))
	}
	for _, v := range vs {
		if v.Reason != "" {
			t.Errorf("零值条件下 Reason 应为空, got %q", v.Reason)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/crtag/`
Expected: 编译失败, `undefined: Criteria` / `undefined: TagView` / `undefined: Filter`

- [ ] **Step 3: 实现**

`internal/crtag/filter.go`:

```go
package crtag

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
)

// Criteria 清理候选过滤条件, 多条件为交集; 零值字段不生效, 全零值返回全部(Reason 为空).
type Criteria struct {
	OlderThan time.Duration // >0: PushTime 早于 Now-OlderThan 才候选
	KeepLast  int           // >0: 按 PushTime 降序保留最近 KeepLast 个, 排名 >= KeepLast 才候选
	TagPrefix string        // 非空: tag 名需有该前缀
	TagNames  []string      // 非空: tag 名需精确命中
	Now       time.Time     // 时间基准, 零值取 time.Now()(测试注入)
}

// TagView 单个 tag 的过滤结果视图.
type TagView struct {
	Tag    *cr.ItemForListTagsOutput
	Reason string // 命中规则串(如 "older-than:720h+beyond-keep-last:10"), 无过滤条件时为空
}

// Filter 按 Criteria 圈定候选 tag(交集语义), 保持输入顺序.
// PushTime 解析失败的 tag 永不进入候选(宁漏删不错删).
func Filter(tags []*cr.ItemForListTagsOutput, c Criteria) []TagView {
	now := c.Now
	if now.IsZero() {
		now = time.Now()
	}
	if c.OlderThan == 0 && c.KeepLast == 0 && c.TagPrefix == "" && len(c.TagNames) == 0 {
		views := make([]TagView, 0, len(tags))
		for _, t := range tags {
			views = append(views, TagView{Tag: t})
		}
		return views
	}
	rank := keepLastRank(tags)
	var out []TagView
	for _, t := range tags {
		name := t.Name == nil ? "" : *t.Name
		var reasons []string
		if c.TagPrefix != "" {
			if !strings.HasPrefix(name, c.TagPrefix) {
				continue
			}
			reasons = append(reasons, "prefix:"+c.TagPrefix)
		}
		if len(c.TagNames) > 0 {
			if !containsName(c.TagNames, name) {
				continue
			}
			reasons = append(reasons, "exact")
		}
		if c.OlderThan > 0 {
			pt, ok := parsePushTimeOf(t)
			if !ok || !pt.Before(now.Add(-c.OlderThan)) {
				continue
			}
			reasons = append(reasons, "older-than:"+c.OlderThan.String())
		}
		if c.KeepLast > 0 {
			r, ok := rank[name]
			if !ok || r < c.KeepLast {
				continue
			}
			reasons = append(reasons, "beyond-keep-last:"+strconv.Itoa(c.KeepLast))
		}
		out = append(out, TagView{Tag: t, Reason: strings.Join(reasons, "+")})
	}
	return out
}

// keepLastRank 返回按 PushTime 降序的名次(0=最新); PushTime 未知的 tag 不在结果中(永不淘汰).
func keepLastRank(tags []*cr.ItemForListTagsOutput) map[string]int {
	type named struct {
		name string
		t    time.Time
	}
	var known []named
	for _, t := range tags {
		if pt, ok := parsePushTimeOf(t); ok {
			known = append(known, named{t.Name == nil ? "" : *t.Name, pt})
		}
	}
	sort.SliceStable(known, func(i, j int) bool { return known[i].t.After(known[j].t) })
	rank := make(map[string]int, len(known))
	for i, k := range known {
		rank[k.name] = i
	}
	return rank
}

func parsePushTimeOf(t *cr.ItemForListTagsOutput) (time.Time, bool) {
	if t.PushTime == nil {
		return time.Time{}, false
	}
	return ParsePushTime(*t.PushTime)
}

func containsName(names []string, s string) bool {
	for _, n := range names {
		if n == s {
			return true
		}
	}
	return false
}
```

注意: Go 不支持三元表达式, 上面 `t.Name == nil ? "" : *t.Name` 处需展开为:

```go
name := ""
if t.Name != nil {
	name = *t.Name
}
```

`keepLastRank` 内同理.

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/crtag/`
Expected: PASS (全部用例)

- [ ] **Step 5: Commit**

```bash
git add internal/crtag/
git commit -m "feat: crtag 清理候选过滤(交集语义)"
```

---

### Task 3: config.ResolveRegion + opts 扩展

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/opts/opts.go`
- Test: `internal/config/config_test.go`(追加)

**Interfaces:**
- Produces: `config.ResolveRegion(flagVal string) string`; `opts.Global.Region`, `opts.Global.Registry` — Task 4/6 依赖

- [ ] **Step 1: 写失败测试**

在 `internal/config/config_test.go` 末尾追加(沿用该文件已有的 import 风格, 需补 `testing` 已有则不重复):

```go
func TestResolveRegion(t *testing.T) {
	t.Setenv(EnvCRRegion, "")
	if got := ResolveRegion(""); got != "cn-north-1" {
		t.Errorf("默认 region 应为 cn-north-1, got %s", got)
	}
	t.Setenv(EnvCRRegion, "cn-shanghai")
	if got := ResolveRegion(""); got != "cn-shanghai" {
		t.Errorf("环境变量应生效, got %s", got)
	}
	if got := ResolveRegion("cn-guangzhou"); got != "cn-guangzhou" {
		t.Errorf("flag 应优先, got %s", got)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/config/`
Expected: 编译失败, `undefined: EnvCRRegion` / `undefined: ResolveRegion`

- [ ] **Step 3: 实现**

`internal/config/config.go`: 常量块加一行, 文件末尾追加函数:

```go
EnvCRRegion = "VOLC_CR_REGION"
```

```go
// DefaultCRRegion cr 模块默认 region(与 codepipeline 默认 region 一致).
const DefaultCRRegion = "cn-north-1"

// ResolveRegion 返回 cr 模块生效的 region: flag 非空优先, 其次环境变量, 默认 cn-north-1.
func ResolveRegion(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(EnvCRRegion); v != "" {
		return v
	}
	return DefaultCRRegion
}
```

`internal/opts/opts.go` 的 `Opts` 结构体追加两个字段(带注释):

```go
Region   string // cr 模块 region(cr 组 persistent flag 绑定)
Registry string // cr 模块实例名(cr 组 persistent flag 绑定, 空则自动解析)
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/config/ && go build ./...`
Expected: PASS / 无输出

- [ ] **Step 5: Commit**

```bash
git add internal/config/ internal/opts/
git commit -m "feat: cr region 解析与 opts 扩展"
```

---

### Task 4: crclient SDK 薄封装

**Files:**
- Create: `internal/crclient/client.go`

**Interfaces:**
- Consumes: Task 3 的 `config.ResolveRegion`(由 cmd 层传入 region 字符串, crclient 只接收)
- Produces — Task 6/7/8 依赖以下全部签名:

```go
crclient.New(ak, sk, region string) (*Client, error)
(*Client).ListRegistriesAll(ctx context.Context) ([]*cr.ItemForListRegistriesOutput, error)
(*Client).ListNamespacesAll(ctx context.Context, registry string) ([]*cr.ItemForListNamespacesOutput, error)
(*Client).ListRepositoriesAll(ctx context.Context, registry string, namespaces []string) ([]*cr.ItemForListRepositoriesOutput, error)
(*Client).ListTagsAll(ctx context.Context, registry, namespace, repository string) ([]*cr.ItemForListTagsOutput, error)
(*Client).DeleteTags(ctx context.Context, registry, namespace, repository string, names []string) (*cr.DeleteTagsOutput, error)
```

按项目惯例 SDK 薄封装不写单测(真实连通性靠手动验收), 本任务无测试步骤, 以 `go build` + `go vet` 通过为完成标准.

- [ ] **Step 1: 实现**

`internal/crclient/client.go`:

```go
// Package crclient 封装火山云镜像仓库(CR) OpenAPI 访问.
// 基于官方 SDK github.com/volcengine/volcengine-go-sdk/service/cr(API 版本 2022-05-12).
// CR 实例是区域性资源, region 由调用方传入.
package crclient

import (
	"context"
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"github.com/volcengine/volcengine-go-sdk/volcengine/session"
)

const (
	listPageSize = 100  // List* 接口 PageSize 上限
	maxPages     = 1000 // 翻页防御上限(10 万条), 避免异常响应导致死循环
)

// Client 持有已配置凭证的 cr 服务实例.
type Client struct {
	svc *cr.CR
}

// New 创建 client, ak/sk 为空时依赖 SDK 默认凭证链(环境变量等).
// 凭证优先级与 CLI 全局约定一致: flag > 环境变量.
func New(ak, sk, region string) (*Client, error) {
	cfg := volcengine.NewConfig().WithRegion(region)
	if ak != "" || sk != "" {
		cfg = cfg.WithAkSk(ak, sk)
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建 session 失败: %w", err)
	}
	return &Client{svc: cr.New(sess, cfg)}, nil
}

// ListRegistriesAll 列出当前 region 全部镜像仓库实例(自动翻页拉全).
func (c *Client) ListRegistriesAll(ctx context.Context) ([]*cr.ItemForListRegistriesOutput, error) {
	var items []*cr.ItemForListRegistriesOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListRegistriesWithContext(ctx, &cr.ListRegistriesInput{
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListRegistries 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListNamespacesAll 列出实例下全部命名空间(自动翻页拉全).
func (c *Client) ListNamespacesAll(ctx context.Context, registry string) ([]*cr.ItemForListNamespacesOutput, error) {
	var items []*cr.ItemForListNamespacesOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListNamespacesWithContext(ctx, &cr.ListNamespacesInput{
			Registry:   &registry,
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListNamespaces 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListRepositoriesAll 列出实例全部制品仓库; namespaces 非空时按命名空间过滤(自动翻页拉全).
func (c *Client) ListRepositoriesAll(ctx context.Context, registry string, namespaces []string) ([]*cr.ItemForListRepositoriesOutput, error) {
	var items []*cr.ItemForListRepositoriesOutput
	in := &cr.ListRepositoriesInput{Registry: &registry}
	if len(namespaces) > 0 {
		in.Filter = &cr.FilterForListRepositoriesInput{Namespaces: namespaces}
	}
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		in.PageNumber = &page
		in.PageSize = &size
		resp, err := c.svc.ListRepositoriesWithContext(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("ListRepositories 第 %d 页失败: %w", page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// ListTagsAll 列出某制品仓库全部版本 tag(自动翻页拉全, 不带服务端 Filter, 过滤由客户端做).
func (c *Client) ListTagsAll(ctx context.Context, registry, namespace, repository string) ([]*cr.ItemForListTagsOutput, error) {
	var items []*cr.ItemForListTagsOutput
	for page := int64(1); page <= maxPages; page++ {
		size := int64(listPageSize)
		resp, err := c.svc.ListTagsWithContext(ctx, &cr.ListTagsInput{
			Registry:   &registry,
			Namespace:  &namespace,
			Repository: &repository,
			PageNumber: &page,
			PageSize:   &size,
		})
		if err != nil {
			return nil, fmt.Errorf("ListTags(%s/%s) 第 %d 页失败: %w", namespace, repository, page, err)
		}
		items = append(items, resp.Items...)
		if int64(len(items)) >= volcengine.Int64Value(resp.TotalCount) || len(resp.Items) < listPageSize {
			break
		}
	}
	return items, nil
}

// DeleteTags 删除指定版本(单批调用, 调用方负责按 20/批切分, 官方单次上限 20 个).
func (c *Client) DeleteTags(ctx context.Context, registry, namespace, repository string, names []string) (*cr.DeleteTagsOutput, error) {
	ptrs := make([]*string, 0, len(names))
	for _, n := range names {
		n := n
		ptrs = append(ptrs, &n)
	}
	return c.svc.DeleteTagsWithContext(ctx, &cr.DeleteTagsInput{
		Registry:   &registry,
		Namespace:  &namespace,
		Repository: &repository,
		Names:      ptrs,
	})
}
```

- [ ] **Step 2: 编译与 vet 验证**

Run: `go build ./... && go vet ./internal/crclient/`
Expected: 无输出(通过)

- [ ] **Step 3: Commit**

```bash
git add internal/crclient/
git commit -m "feat: crclient SDK 薄封装(翻页拉全)"
```

---

### Task 5: output 渲染(registries / namespaces / repositories / tags / 删除结果)

**Files:**
- Create: `internal/output/cr.go`
- Test: `internal/output/cr_test.go`

**Interfaces:**
- Consumes: Task 2 的 `crtag.TagView`
- Produces — Task 6/7/8 依赖:

```go
output.PrintRegistriesTo(w io.Writer, items []*cr.ItemForListRegistriesOutput) error
output.PrintNamespacesTo(w io.Writer, items []*cr.ItemForListNamespacesOutput) error
output.PrintRepositoriesTo(w io.Writer, items []*cr.ItemForListRepositoriesOutput) error
output.PrintTagsTo(w io.Writer, views []crtag.TagView, withRepo, withReason bool) error
output.PrintDeleteTagsResultTo(w io.Writer, deleted, remaining []string, failures []*cr.FailureForDeleteTagsOutput) error
```

- [ ] **Step 1: 写失败测试**

`internal/output/cr_test.go`(复用 output_test.go 已有的 `sptr`/`iptr` helper, 同包可见):

```go
package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"kuopin/volc-cli/internal/crtag"
)

func TestPrintRegistriesTable(t *testing.T) {
	var buf bytes.Buffer
	items := []*cr.ItemForListRegistriesOutput{
		{Name: sptr("reg-1"), Type: sptr("Standard"), CreateTime: sptr("2026-01-01T00:00:00Z"),
			Status: &cr.StatusForListRegistriesOutput{Phase: sptr("Running")}},
	}
	if err := PrintRegistriesTo(&buf, items); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"reg-1", "Standard", "Running"} {
		if !strings.Contains(out, want) {
			t.Errorf("实例表格应包含 %q, got:\n%s", want, out)
		}
	}
}

func TestPrintRegistriesEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintRegistriesTo(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "无镜像仓库实例") {
		t.Errorf("空列表应提示无实例(并提醒检查 --region), got: %s", buf.String())
	}
}

func TestPrintNamespacesAndRepositories(t *testing.T) {
	var buf bytes.Buffer
	ns := []*cr.ItemForListNamespacesOutput{{Name: sptr("team-a"), CreateTime: sptr("2026-01-01T00:00:00Z")}}
	if err := PrintNamespacesTo(&buf, ns); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "team-a") {
		t.Errorf("命名空间表格应包含 team-a, got:\n%s", buf.String())
	}

	buf.Reset()
	repos := []*cr.ItemForListRepositoriesOutput{
		{Name: sptr("svc-api"), Namespace: sptr("team-a"), AccessLevel: sptr("Private")},
	}
	if err := PrintRepositoriesTo(&buf, repos); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "svc-api") || !strings.Contains(buf.String(), "Private") {
		t.Errorf("制品仓库表格应包含 svc-api/Private, got:\n%s", buf.String())
	}
}

func TestPrintTagsTable(t *testing.T) {
	var buf bytes.Buffer
	views := []crtag.TagView{
		{Tag: &cr.ItemForListTagsOutput{Name: sptr("v1"), PushTime: sptr("2026-09-01T00:00:00Z"),
			Type: sptr("Image"), Size: iptr(150 * 1024 * 1024), Digest: sptr("sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")}},
		{Tag: &cr.ItemForListTagsOutput{Name: sptr("bad"), PushTime: sptr("garbage"), Type: sptr("Image")}},
	}
	// 带 REASON 列(过滤模式)
	if err := PrintTagsTo(&buf, views, false, true); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"v1", "150.0MB", "sha256:0123456789abc…", "未知"} {
		if !strings.Contains(out, want) {
			t.Errorf("tag 表格应包含 %q, got:\n%s", want, out)
		}
	}
	// 带 REPOSITORY 列(多 repo 模式)
	buf.Reset()
	views[0].Reason = "older-than:720h"
	if err := PrintTagsTo(&buf, views, true, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "older-than:720h") {
		t.Errorf("多 repo+过滤模式应含 REASON, got:\n%s", buf.String())
	}
}

func TestPrintDeleteTagsResult(t *testing.T) {
	var buf bytes.Buffer
	failures := []*cr.FailureForDeleteTagsOutput{{Name: sptr("t3"), Reason: sptr("immutable tag")}}
	if err := PrintDeleteTagsResultTo(&buf, []string{"t1", "t2"}, []string{"t4"}, failures); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"t1", "t2", "t3", "immutable tag", "t4", "失败 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("删除结果应包含 %q, got:\n%s", want, out)
		}
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/output/`
Expected: 编译失败, `undefined: PrintRegistriesTo` 等

- [ ] **Step 3: 实现**

`internal/output/cr.go`:

```go
package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/crtag"
)

// PrintRegistriesTo 以表格输出镜像仓库实例列表.
func PrintRegistriesTo(w io.Writer, items []*cr.ItemForListRegistriesOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无镜像仓库实例(请确认 --region 与凭证)")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tTYPE\tPHASE\tCREATE_TIME\tEXPIRE_TIME")
	for _, r := range items {
		phase := ""
		if r.Status != nil {
			phase = volcengine.StringValue(r.Status.Phase)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			volcengine.StringValue(r.Name), volcengine.StringValue(r.Type), phase,
			volcengine.StringValue(r.CreateTime), volcengine.StringValue(r.ExpireTime))
	}
	return tw.Flush()
}

// PrintNamespacesTo 以表格输出命名空间列表.
func PrintNamespacesTo(w io.Writer, items []*cr.ItemForListNamespacesOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无命名空间")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tPROJECT\tCREATE_TIME")
	for _, n := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			volcengine.StringValue(n.Name), volcengine.StringValue(n.Project),
			volcengine.StringValue(n.CreateTime))
	}
	return tw.Flush()
}

// PrintRepositoriesTo 以表格输出 OCI 制品仓库列表.
func PrintRepositoriesTo(w io.Writer, items []*cr.ItemForListRepositoriesOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无制品仓库")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE\tNAME\tACCESS_LEVEL\tCREATE_TIME")
	for _, r := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			volcengine.StringValue(r.Namespace), volcengine.StringValue(r.Name),
			volcengine.StringValue(r.AccessLevel), volcengine.StringValue(r.CreateTime))
	}
	return tw.Flush()
}

// PrintTagsTo 以表格输出 tag 列表/清理候选.
// withRepo: 多 repo 聚合时输出 REPOSITORY 列; withReason: 过滤模式输出 REASON 列.
func PrintTagsTo(w io.Writer, views []crtag.TagView, withRepo, withReason bool) error {
	if len(views) == 0 {
		fmt.Fprintln(w, "无版本 tag")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	head := []string{"NAME", "TYPE", "PUSH_TIME", "SIZE", "DIGEST"}
	if withRepo {
		head = append([]string{"REPOSITORY"}, head...)
	}
	if withReason {
		head = append(head, "REASON")
	}
	fmt.Fprintln(tw, strings.Join(head, "\t"))
	for _, v := range views {
		t := v.Tag
		pushTime := volcengine.StringValue(t.PushTime)
		if _, ok := crtag.ParsePushTime(pushTime); pushTime != "" && !ok {
			pushTime += "(未知)"
		}
		if pushTime == "" {
			pushTime = "-"
		}
		row := []string{
			volcengine.StringValue(t.Name), volcengine.StringValue(t.Type),
			pushTime, formatSize(t.Size), shortDigest(t.Digest),
		}
		if withRepo {
			row = append([]string{v.Repository}, row...)
		}
		if withReason {
			row = append(row, v.Reason)
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

// PrintDeleteTagsResultTo 输出删除结果: 已删除/失败/未执行清单.
func PrintDeleteTagsResultTo(w io.Writer, deleted, remaining []string, failures []*cr.FailureForDeleteTagsOutput) error {
	fmt.Fprintf(w, "已删除 %d 个: %s\n", len(deleted), strings.Join(deleted, ", "))
	if len(failures) > 0 {
		fmt.Fprintf(w, "失败 %d 个:\n", len(failures))
		for _, f := range failures {
			fmt.Fprintf(w, "  %s\t%s\n", volcengine.StringValue(f.Name), volcengine.StringValue(f.Reason))
		}
	}
	if len(remaining) > 0 {
		fmt.Fprintf(w, "未执行 %d 个(请求中断): %s\n", len(remaining), strings.Join(remaining, ", "))
	}
	return nil
}

// shortDigest 截断 digest 到可读长度.
func shortDigest(d *string) string {
	s := volcengine.StringValue(d)
	if len(s) > 19 {
		return s[:19] + "…"
	}
	return s
}

// formatSize 将字节数渲染为人类可读.
func formatSize(b *int64) string {
	n := volcengine.Int64Value(b)
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1fGB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}
```

注意: 上面 `PrintTagsTo` 用到 `v.Repository`, 因此 **Task 2 的 `TagView` 需加 `Repository string` 字段**:
在 `internal/crtag/filter.go` 的 `TagView` 中加 `Repository string`(Filter 的调用方负责填充; Filter 自身不依赖它). 本任务一并修改并补一条断言到 Task 2 的测试(`TestFilterZeroCriteriaReturnsAll` 中加 `if v.Repository != "" { t.Errorf(...) }` 可不加, 该字段由 cmd 层填充, crtag 不设置).

同时补便捷包装(风格对齐现有 output):

```go
// PrintRegistries 等便捷包装输出到 stdout.
func PrintRegistries(items []*cr.ItemForListRegistriesOutput) error { return PrintRegistriesTo(stdout, items) }
func PrintNamespaces(items []*cr.ItemForListNamespacesOutput) error { return PrintNamespacesTo(stdout, items) }
func PrintRepositories(items []*cr.ItemForListRepositoriesOutput) error {
	return PrintRepositoriesTo(stdout, items)
}
func PrintTags(views []crtag.TagView, withRepo, withReason bool) error {
	return PrintTagsTo(stdout, views, withRepo, withReason)
}
func PrintDeleteTagsResult(deleted, remaining []string, failures []*cr.FailureForDeleteTagsOutput) error {
	return PrintDeleteTagsResultTo(stdout, deleted, remaining, failures)
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `go test ./internal/output/ ./internal/crtag/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/output/ internal/crtag/
git commit -m "feat: cr 各列表与删除结果渲染"
```

---

### Task 6: cmd/cr 父命令 + list-registries / list-namespaces / list-repositories + root 挂载

**Files:**
- Create: `internal/cmd/cr/cr.go`
- Create: `internal/cmd/cr/list_registries.go`
- Create: `internal/cmd/cr/list_namespaces.go`
- Create: `internal/cmd/cr/list_repositories.go`
- Modify: `internal/cmd/root.go`(挂载 + Long 更新)

**Interfaces:**
- Consumes: Task 3 `config.ResolveCredentials/ResolveRegion` + `opts.Global`; Task 4 crclient 全部; Task 5 `PrintRegistries/PrintNamespaces/PrintRepositories`
- Produces: `cr.NewCRCmd() *cobra.Command`; 包内 `newClient() (*crclient.Client, error)`; `resolveRegistry(client *crclient.Client, ctx context.Context) (string, error)` — Task 7/8 复用

- [ ] **Step 1: 实现父命令与公共解析**

`internal/cmd/cr/cr.go`:

```go
// Package cr 提供 volc-cli cr 子命令组, 对应火山云镜像仓库 OpenAPI(API 版本 2022-05-12).
package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/config"
	"kuopin/volc-cli/internal/crclient"
	"kuopin/volc-cli/internal/opts"
)

// NewCRCmd 创建 cr 模块父命令.
func NewCRCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cr",
		Short: "镜像仓库(Container Registry)操作",
		Long: "对应镜像仓库 OpenAPI(API 版本 2022-05-12), 命令名与 Action 对齐:\n" +
			"  list-registries     ListRegistries   列出当前 region 全部实例\n" +
			"  list-namespaces     ListNamespaces   列出实例下全部命名空间\n" +
			"  list-repositories   ListRepositories 列出命名空间下全部 OCI 制品仓库\n" +
			"  list-tags           ListTags         列出制品仓库全部版本, 可按清理候选过滤(客户端语义)\n" +
			"  delete-tags         DeleteTags       删除指定版本, 需显式 --tags 列表与 --yes 二次确认\n\n" +
			"删除是不可恢复的线上修改: Agent 调用前必须向用户说明目标与影响并获得明确确认.",
	}
	pf := c.PersistentFlags()
	pf.StringVar(&opts.Global.Region, "region", "", "实例所在 region, 默认读环境变量 VOLC_CR_REGION 或 cn-north-1")
	pf.StringVar(&opts.Global.Registry, "registry", "", "镜像仓库实例名, 未指定时自动解析(唯一实例直接用, 多实例报错列出候选)")
	c.AddCommand(newListRegistriesCmd(), newListNamespacesCmd(), newListRepositoriesCmd(),
		newListTagsCmd(), newDeleteTagsCmd())
	return c
}

// newClient 用全局凭证与生效 region 构造 crclient, 失败时报错.
func newClient() (*crclient.Client, error) {
	ak, sk := config.ResolveCredentials(opts.Global.AccessKey, opts.Global.SecretKey)
	return crclient.New(ak, sk, config.ResolveRegion(opts.Global.Region))
}

// resolveRegistry 解析实例名: --registry 优先, 未配置时自动列出(唯一实例直接用).
func resolveRegistry(client *crclient.Client, ctx context.Context) (string, error) {
	if opts.Global.Registry != "" {
		return opts.Global.Registry, nil
	}
	items, err := client.ListRegistriesAll(ctx)
	if err != nil {
		return "", fmt.Errorf("registry 未配置且自动获取失败: %w(请用 --registry 指定或先跑 list-registries 查看)", err)
	}
	switch len(items) {
	case 0:
		return "", fmt.Errorf("当前 region(%s)下无镜像仓库实例, 请检查 --region", config.ResolveRegion(opts.Global.Region))
	case 1:
		if items[0].Name == nil {
			return "", fmt.Errorf("实例名为空, 请用 --registry 指定")
		}
		return *items[0].Name, nil
	default:
		names := make([]string, 0, len(items))
		for _, r := range items {
			names = append(names, volcengine.StringValue(r.Name))
		}
		return "", fmt.Errorf("当前 region 下有多个实例: %s, 请用 --registry 指定", strings.Join(names, ", "))
	}
}
```

注: `resolveRegistry` 中 `volcengine.StringValue` 需要 import `"github.com/volcengine/volcengine-go-sdk/volcengine"`.

- [ ] **Step 2: 实现三个 list 命令**

`internal/cmd/cr/list_registries.go`:

```go
package cr

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListRegistriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-registries",
		Short: "ListRegistries: 列出当前 region 全部镜像仓库实例",
		Long:  "ListRegistries: 列出当前 region 全部镜像仓库实例(自动翻页拉全).\n实例是区域性资源, 无实例时请检查 --region 是否正确.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			items, err := client.ListRegistriesAll(context.Background())
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintRegistries(items)
		},
	}
}
```

`internal/cmd/cr/list_namespaces.go`:

```go
package cr

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListNamespacesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-namespaces",
		Short: "ListNamespaces: 列出实例下全部命名空间",
		Long:  "ListNamespaces: 列出实例下全部命名空间(自动翻页拉全).\nRegistry 未配置时自动解析(唯一实例直接用, 多实例用 --registry 指定).",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			registry, err := resolveRegistry(client, context.Background())
			if err != nil {
				return err
			}
			items, err := client.ListNamespacesAll(context.Background(), registry)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintNamespaces(items)
		},
	}
}
```

`internal/cmd/cr/list_repositories.go`:

```go
package cr

import (
	"context"

	"github.com/spf13/cobra"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

func newListRepositoriesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list-repositories",
		Short: "ListRepositories: 列出制品仓库",
		Long:  "ListRepositories: 列出实例下全部 OCI 制品仓库(自动翻页拉全).\n" +
			"--namespace 过滤指定命名空间(Filter.Namespaces, 可逗号分隔多个); 省略时列出全部.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newClient()
			if err != nil {
				return err
			}
			registry, err := resolveRegistry(client, context.Background())
			if err != nil {
				return err
			}
			var namespaces []string
			if namespacesFlag != "" {
				for _, n := range strings.Split(namespacesFlag, ",") {
					if n = strings.TrimSpace(n); n != "" {
						namespaces = append(namespaces, n)
					}
				}
			}
			items, err := client.ListRepositoriesAll(context.Background(), registry, namespaces)
			if err != nil {
				return err
			}
			if opts.Global.JSON {
				return output.PrintJSON(items)
			}
			return output.PrintRepositories(items)
		},
	}
	c.Flags().StringVar(&namespacesFlag, "namespace", "", "按命名空间过滤, 逗号分隔多个")
	return c
}

// namespacesFlag 由 list-repositories 的 --namespace flag 绑定(包级变量, 各命令文件独立持有自己的 flag 变量).
var namespacesFlag string
```

- [ ] **Step 3: root 挂载与 Long 更新**

`internal/cmd/root.go` 修改: import 加 `"kuopin/volc-cli/internal/cmd/cr"`, `root.AddCommand(cr.NewCRCmd())`; Long 中 `"当前支持模块: codepipeline(持续交付)."` 改为:

```
"当前支持模块: codepipeline(持续交付), cr(镜像仓库).\n\n"
```

并把末尾 `"当前命令仅检查凭证或查询, 无需修改确认."` 改为:

```
"cr delete-tags 是删除类写操作, 执行前必须获得用户对具体目标的明确确认."
```

- [ ] **Step 4: 编译与现有测试回归**

Run: `go build ./... && go test ./... && go run . cr --help`
Expected: build/test 通过; cr --help 显示 5 个子命令与 --region/--registry flag

- [ ] **Step 5: Commit**

```bash
git add internal/cmd/
git commit -m "feat: cr 父命令与 list-registries/namespaces/repositories"
```

注: 此时 `newListTagsCmd`/`newDeleteTagsCmd` 尚未定义, 编译会失败. 本任务临时在 `cr.go` 中先创建最小占位实现(仅 Use/Short, RunE 返回 nil 不实现), Task 7/8 再替换为完整实现:

```go
func newListTagsCmd() *cobra.Command {
	return &cobra.Command{Use: "list-tags", Short: "ListTags(待实现)", RunE: func(*cobra.Command, []string) error { return nil }}
}

func newDeleteTagsCmd() *cobra.Command {
	return &cobra.Command{Use: "delete-tags", Short: "DeleteTags(待实现)", RunE: func(*cobra.Command, []string) error { return nil }}
}
```

占位文件分别放 `internal/cmd/cr/list_tags.go` / `delete_tags.go`, 后续任务直接覆盖其内容.

---

### Task 7: list-tags 命令(多 repo 遍历 + 过滤 + JSON 聚合)

**Files:**
- Modify: `internal/cmd/cr/list_tags.go`(覆盖 Task 6 的占位实现)

**Interfaces:**
- Consumes: Task 2 `crtag.{Criteria, TagView, Filter, ParseOlderThan}`; Task 4 `ListRepositoriesAll/ListTagsAll`; Task 5 `PrintTags`; Task 6 `newClient/resolveRegistry`
- Produces: 可用的 `volc-cli cr list-tags` 命令

- [ ] **Step 1: 实现命令**

`internal/cmd/cr/list_tags.go` 完整替换为:

```go
package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"kuopin/volc-cli/internal/crtag"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// repoTags --json 多 repo 聚合输出结构(字段名保持 SDK 风格).
type repoTags struct {
	Repository string                      `json:"Repository"`
	Items      []*cr.ItemForListTagsOutput `json:"Items"`
}

var (
	ltRepository string
	ltTagNames   string
	ltTagPrefix  string
	ltOlderThan  string
	ltKeepLast   int
)

func newListTagsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list-tags",
		Short: "ListTags: 列出制品仓库全部版本",
		Long: "ListTags: 列出制品仓库全部版本 tag(自动翻页拉全).\n" +
			"--repository 必填; 省略时遍历 --namespace 下全部制品仓库(只读聚合).\n\n" +
			"以下为清理候选过滤参数(客户端语义, 非 API 字段, 多条件交集):\n" +
			"  --older-than 30d   PushTime 早于 N 天前(支持 Nd/Nh)\n" +
			"  --keep-last 10      按 PushTime 降序保留最近 N 个, 其余为候选\n" +
			"  --tag-prefix ci-    tag 名前缀匹配\n" +
			"  --tag-names t1,t2   tag 名精确匹配\n" +
			"带任一过滤参数时输出 REASON 列; PushTime 无法解析的 tag 永不进入候选.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ltNamespace() == "" {
				return fmt.Errorf("--namespace 必填")
			}
			criteria := crtag.Criteria{TagPrefix: ltTagPrefix, KeepLast: ltKeepLast}
			if ltTagNames != "" {
				for _, n := range strings.Split(ltTagNames, ",") {
					if n = strings.TrimSpace(n); n != "" {
						criteria.TagNames = append(criteria.TagNames, n)
					}
				}
			}
			if ltOlderThan != "" {
				d, err := crtag.ParseOlderThan(ltOlderThan)
				if err != nil {
					return err
				}
				criteria.OlderThan = d
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			registry, err := resolveRegistry(client, ctx)
			if err != nil {
				return err
			}
			ns := ltNamespace()
			repos := []string{ltRepository}
			if ltRepository == "" {
				items, err := client.ListRepositoriesAll(ctx, registry, []string{ns})
				if err != nil {
					return err
				}
				repos = repos[:0]
				for _, r := range items {
					repos = append(repos, deref(r.Name))
				}
			}
			multiRepo := len(repos) > 1
			var allViews []crtag.TagView
			var jsonOut []repoTags
			for _, repo := range repos {
				tags, err := client.ListTagsAll(ctx, registry, ns, repo)
				if err != nil {
					return err
				}
				views := crtag.Filter(tags, criteria)
				for i := range views {
					views[i].Repository = repo
				}
				allViews = append(allViews, views...)
				if opts.Global.JSON {
					items := make([]*cr.ItemForListTagsOutput, 0, len(views))
					for _, v := range views {
						items = append(items, v.Tag)
					}
					jsonOut = append(jsonOut, repoTags{Repository: repo, Items: items})
				}
			}
			if opts.Global.JSON {
				if !multiRepo {
					// 单 repo 输出 SDK 结构原样字段数组
					items := []*cr.ItemForListTagsOutput{}
					if len(jsonOut) == 1 {
						items = jsonOut[0].Items
					}
					return output.PrintJSON(items)
				}
				return output.PrintJSON(jsonOut)
			}
			withReason := criteria.OlderThan > 0 || criteria.KeepLast > 0 || criteria.TagPrefix != "" || len(criteria.TagNames) > 0
			return output.PrintTags(allViews, multiRepo, withReason)
		},
	}
	f := c.Flags()
	f.StringVar(&ltRepository, "repository", "", "制品仓库名; 省略时遍历命名空间下全部仓库")
	f.StringVar(&ltTagNames, "tag-names", "", "tag 名精确匹配, 逗号分隔")
	f.StringVar(&ltTagPrefix, "tag-prefix", "", "tag 名前缀匹配")
	f.StringVar(&ltOlderThan, "older-than", "", "PushTime 早于该时长(如 30d/12h), 客户端过滤")
	f.IntVar(&ltKeepLast, "keep-last", 0, "每个仓库按 PushTime 降序保留最近 N 个, 其余为候选")
	_ = ltNamespace // 见下
	return c
}
```

上面引用了两个待补 helper, 在本文件补充(命名空间 flag 本任务起也由本命令持有):

```go
var ltNamespaceFlag string

func ltNamespace() string { return ltNamespaceFlag }
```

并把 `f := c.Flags()` 后追加:

```go
	f.StringVar(&ltNamespaceFlag, "namespace", "", "命名空间(必填)")
```

`deref` helper 加到 `cr.go`:

```go
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

`resolveRegistry` 中的 `volcengine.StringValue` 调用可统一改用 `deref`(可选, 保持简单).

- [ ] **Step 2: 编译与全量测试**

Run: `go build ./... && go test ./... && go run . cr list-tags --help`
Expected: build/test 通过; help 显示全部过滤 flag

- [ ] **Step 3: Commit**

```bash
git add internal/cmd/cr/
git commit -m "feat: cr list-tags(多repo遍历+清理候选过滤)"
```

---

### Task 8: delete-tags 命令(显式列表 + --yes + 20/批)

**Files:**
- Modify: `internal/cmd/cr/delete_tags.go`(覆盖 Task 6 的占位实现)

**Interfaces:**
- Consumes: Task 4 `DeleteTags`; Task 5 `PrintDeleteTagsResult`; Task 6 `newClient/resolveRegistry`
- Produces: 可用的 `volc-cli cr delete-tags` 命令

- [ ] **Step 1: 实现命令**

`internal/cmd/cr/delete_tags.go` 完整替换为:

```go
package cr

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/volcengine/volcengine-go-sdk/service/cr"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/opts"
	"kuopin/volc-cli/internal/output"
)

// deleteTagBatch 官方 DeleteTags 单次请求上限.
const deleteTagBatch = 20

var (
	dtNamespace  string
	dtRepository string
	dtTags       string
	dtYes        bool
)

func newDeleteTagsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "delete-tags",
		Short: "DeleteTags: 删除制品仓库中指定版本",
		Long: "DeleteTags: 删除制品仓库中指定的版本 tag, 不可恢复.\n" +
			"必须显式指定 --tags(逗号分隔的精确列表)并加 --yes 二次确认; 不做任何自动圈定.\n" +
			"超过 20 个自动分批调用; 请求级失败即止并报告进度, 批内逐条失败继续, 存在失败时以非零码退出.\n" +
			"Agent 调用前必须先向用户展示删除清单并获得明确确认.",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := parseTagList(dtTags)
			if len(names) == 0 {
				return fmt.Errorf("--tags 不能为空: 请用逗号分隔显式列出要删除的版本名")
			}
			if dtNamespace == "" || dtRepository == "" {
				return fmt.Errorf("--namespace 与 --repository 必填")
			}
			if !dtYes {
				return fmt.Errorf("删除是不可恢复操作, 请核对后加 --yes 重试:\n"+
					"  tags: %s", strings.Join(names, ","))
			}
			client, err := newClient()
			if err != nil {
				return err
			}
			ctx := context.Background()
			registry, err := resolveRegistry(client, ctx)
			if err != nil {
				return err
			}
			var deleted, remaining []string
			var failures []*cr.FailureForDeleteTagsOutput
			for i := 0; i < len(names); i += deleteTagBatch {
				end := i + deleteTagBatch
				if end > len(names) {
					end = len(names)
				}
				batch := names[i:end]
				resp, err := client.DeleteTags(ctx, registry, dtNamespace, dtRepository, batch)
				if err != nil {
					// 失败批次本身未删除, 未执行清单从本批起算
					remaining = names[i:]
					_ = output.PrintDeleteTagsResult(deleted, remaining, failures)
					return fmt.Errorf("第 %d 批(共 %d 批)请求失败: %w", i/deleteTagBatch+1,
						(len(names)+deleteTagBatch-1)/deleteTagBatch, err)
				}
				for _, s := range resp.Successes {
					deleted = append(deleted, volcengine.StringValue(s.Name))
				}
				failures = append(failures, resp.Failures...)
			}
			if opts.Global.JSON {
				return output.PrintJSON(map[string]any{
					"Registry": registry, "Namespace": dtNamespace, "Repository": dtRepository,
					"Deleted": deleted, "Failures": failures, "Remaining": remaining,
				})
			}
			if err := output.PrintDeleteTagsResult(deleted, remaining, failures); err != nil {
				return err
			}
			if len(failures) > 0 {
				return fmt.Errorf("%d 个版本删除失败(见上方失败清单)", len(failures))
			}
			return nil
		},
	}
	f := c.Flags()
	f.StringVar(&dtNamespace, "namespace", "", "命名空间(必填)")
	f.StringVar(&dtRepository, "repository", "", "制品仓库名(必填)")
	f.StringVar(&dtTags, "tags", "", "要删除的版本名, 逗号分隔显式列表(必填)")
	f.BoolVar(&dtYes, "yes", false, "确认执行删除(不可恢复)")
	return c
}

// parseTagList 解析逗号分隔的 tag 名: 去空白/去空项/保序去重.
func parseTagList(s string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, n := range strings.Split(s, ",") {
		if n = strings.TrimSpace(n); n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}
```

- [ ] **Step 2: 编译与全量测试**

Run: `go build ./... && go test ./... && go run . cr delete-tags --help`
Expected: build/test 通过; help 显示 --tags/--yes 说明

- [ ] **Step 3: 冒烟(无凭证环境下验证参数校验路径)**

Run: `go run . cr delete-tags --namespace demo --repository demo --tags v1` (不带 --yes)
Expected: 非零退出, 错误信息回显 tags 清单并提示加 --yes

Run: `go run . cr list-tags` (不带 --namespace)
Expected: 报错 `--namespace 必填`

- [ ] **Step 4: Commit**

```bash
git add internal/cmd/cr/
git commit -m "feat: cr delete-tags(显式列表+--yes+分批)"
```

---

### Task 9: 文档同步 + 全量验收

**Files:**
- Modify: `README.md`
- Modify: `skills/volc-cli/SKILL.md`
- Modify: `CONTRIBUTING.md`

**Interfaces:**
- Consumes: Task 1-8 的全部成果

- [ ] **Step 1: 更新 SKILL.md**

`skills/volc-cli/SKILL.md` 修改点:
- frontmatter `description` 扩展: 追加 `也支持镜像仓库(cr)实例/命名空间/制品仓库/版本查询与过期版本清理(显式列表删除).`
- 「触发条件」追加: `- 用户要查火山云镜像仓库的实例/命名空间/仓库/版本, 或要清理过期镜像版本`
- 「线上修改确认」节的末句 `当前 CLI 仅提供凭证检查与查询命令, 此规则不表示已支持修改命令.` 改为: `当前 CLI 的写操作仅 cr delete-tags(删除镜像版本), 执行前必须先把删除清单展示给用户并获得明确确认.`
- 「操作流程」追加 cr 小节:

```markdown
7. 镜像仓库查询: `volc-cli cr list-registries`(region 默认 cn-north-1, 用 --region 切换)
8. 列命名空间/仓库/版本: `volc-cli cr list-namespaces` / `list-repositories --namespace N` / `list-tags --namespace N [--repository X]`
9. 清理过期版本(两步, 不自动批量删):
   - 圈候选: `volc-cli cr list-tags --namespace N [--repository X] --older-than 30d [--keep-last 10] [--tag-prefix ci-]`
   - 把候选清单展示给用户, 获得对具体 tag 列表的明确确认后执行:
     `volc-cli cr delete-tags --namespace N --repository X --tags t1,t2 --yes`
```

- 「关键规则」追加:

```markdown
- cr 删除必须基于用户确认过的显式 tag 列表(`--tags`), 不做自动批量删除; 先 list-tags 圈候选再人工确认
- PushTime 显示"未知"的版本不参与 older-than/keep-last 候选(宁漏删不错删), 处置需用户单独确认
```

- [ ] **Step 2: 更新 README.md**

「命令结构与规范」段(文件头部说明处)追加 cr 模块示例一行; 新增「镜像仓库(cr)」用法小节(放在 codepipeline 用法之后), 内容含: 5 条命令示例、region/registry 解析规则、清理两步流示例、删除安全说明(显式列表 + --yes + 不可恢复)。

- [ ] **Step 3: 更新 CONTRIBUTING.md**

「目录结构」代码块补 `crclient`/`crtag`/`cmd/cr` 三行; 「项目说明」的模块列表加 `cr(镜像仓库, API 版本 2022-05-12)`; 「全局约束」的 workspace 一条后追加 region 规则一行:

```
- cr region 优先级: `--region` flag > 环境变量 `VOLC_CR_REGION` > 默认 `cn-north-1`; cr 删除命令须显式 `--tags` 列表 + `--yes`
```

- [ ] **Step 4: 全量验证**

Run: `go build ./... && go test ./... && go vet ./...`
Expected: 全部通过无输出

Run: `go run . --help && go run . cr --help`
Expected: 模块清单含 cr; cr 帮助含 5 命令

- [ ] **Step 5: 手动验收(需真实凭证, 交给用户执行或经用户同意后执行)**

```bash
export VOLC_ACCESS_KEY=... VOLC_SECRET_KEY=...
go run . cr list-registries                 # 验证 region/凭证/表格输出
go run . cr list-namespaces                 # 验证 registry 自动解析
go run . cr list-tags --namespace <N> --repository <X>   # 关键: 确认 PUSH_TIME 列是否可读
```

重点校准: 若 PUSH_TIME 列全部显示"(未知)", 说明真实格式不在 ParsePushTime 支持列表中 → 从 `--json` 输出取一条原始 PushTime 值, 在 `internal/crtag/time.go` 的 ParsePushTime 中补对应格式分支与测试用例, 重跑单测后 commit(`fix: PushTime 解析支持真实格式`).

最后在一个测试用途的 tag 上验证删除流:

```bash
go run . cr delete-tags --namespace <N> --repository <X> --tags <测试tag>   # 无 --yes 应报错
go run . cr delete-tags --namespace <N> --repository <X> --tags <测试tag> --yes
```

- [ ] **Step 6: Commit**

```bash
git add README.md skills/ CONTRIBUTING.md
git commit -m "docs: cr 模块文档同步"
```

---

## Self-Review 结论

- **Spec 覆盖**: 5 命令(Task 6/7/8)、region/registry 解析(Task 3/6)、翻页(Task 4)、过滤交集与安全语义(Task 1/2)、JSON 形态(Task 7/8)、分批与失败策略(Task 8)、文档同步(Task 9)均有对应任务; spec 的"不做"清单未引入对应实现
- **类型一致性**: `TagView.Repository` 在 Task 5 中由 output 使用、Task 7 填充, Task 2 定义(计划内已标注补充该字段); `deref`/`ltNamespace` 等 helper 定义位置已写明
- **占位符**: Task 6 的临时占位命令是显式的两阶段策略(编译可通过, Task 7/8 覆盖), 非未完成占位
