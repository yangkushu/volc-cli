package output

import (
	"fmt"
	"io"
	"strconv"
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
// tagCounts 非空时追加 TAG_COUNT 列, key 为 "<namespace>/<name>", 缺失项显示 "-".
func PrintRepositoriesTo(w io.Writer, items []*cr.ItemForListRepositoriesOutput, tagCounts map[string]int64) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无制品仓库")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if tagCounts == nil {
		fmt.Fprintln(tw, "NAMESPACE\tNAME\tACCESS_LEVEL\tCREATE_TIME")
		for _, r := range items {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
				volcengine.StringValue(r.Namespace), volcengine.StringValue(r.Name),
				volcengine.StringValue(r.AccessLevel), volcengine.StringValue(r.CreateTime))
		}
		return tw.Flush()
	}
	fmt.Fprintln(tw, "NAMESPACE\tNAME\tACCESS_LEVEL\tCREATE_TIME\tTAG_COUNT")
	for _, r := range items {
		count := "-"
		if n, ok := tagCounts[RepoKey(r)]; ok {
			count = strconv.FormatInt(n, 10)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			volcengine.StringValue(r.Namespace), volcengine.StringValue(r.Name),
			volcengine.StringValue(r.AccessLevel), volcengine.StringValue(r.CreateTime), count)
	}
	return tw.Flush()
}

// RepoKey 返回制品仓库的计数 map key: "<namespace>/<name>".
func RepoKey(r *cr.ItemForListRepositoriesOutput) string {
	return volcengine.StringValue(r.Namespace) + "/" + volcengine.StringValue(r.Name)
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

// PrintRegistries 等便捷包装输出到 stdout.
func PrintRegistries(items []*cr.ItemForListRegistriesOutput) error {
	return PrintRegistriesTo(stdout, items)
}
func PrintNamespaces(items []*cr.ItemForListNamespacesOutput) error {
	return PrintNamespacesTo(stdout, items)
}
func PrintRepositories(items []*cr.ItemForListRepositoriesOutput, tagCounts map[string]int64) error {
	return PrintRepositoriesTo(stdout, items, tagCounts)
}
func PrintTags(views []crtag.TagView, withRepo, withReason bool) error {
	return PrintTagsTo(stdout, views, withRepo, withReason)
}
func PrintDeleteTagsResult(deleted, remaining []string, failures []*cr.FailureForDeleteTagsOutput) error {
	return PrintDeleteTagsResultTo(stdout, deleted, remaining, failures)
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
