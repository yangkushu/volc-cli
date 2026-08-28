package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/volcengine/volcengine-go-sdk/service/cp"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"kuopin/volc-cli/internal/failure"
)

var stdout io.Writer = os.Stdout

// PrintWorkspacesTo 以表格输出工作区列表.
func PrintWorkspacesTo(w io.Writer, items []*cp.ItemForListWorkspacesOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无工作区")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tVISIBILITY\tCREATE_TIME\tDESCRIPTION")
	for _, ws := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			volcengine.StringValue(ws.Id), volcengine.StringValue(ws.Name),
			volcengine.StringValue(ws.Visibility), volcengine.StringValue(ws.CreateTime),
			volcengine.StringValue(ws.Description))
	}
	return tw.Flush()
}

// PrintPipelinesTo 以表格输出流水线列表.
func PrintPipelinesTo(w io.Writer, items []*cp.ItemForListPipelinesOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无流水线")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tCREATE_TIME\tDESCRIPTION")
	for _, p := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			volcengine.StringValue(p.Id), volcengine.StringValue(p.Name),
			volcengine.StringValue(p.CreateTime), volcengine.StringValue(p.Description))
	}
	return tw.Flush()
}

// PrintRunsTo 以表格输出执行记录(含发布参数).
func PrintRunsTo(w io.Writer, items []*cp.ItemForListPipelineRunsOutput) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无执行记录")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tINDEX\tSTATUS\tTRIGGER\tSTART_TIME\tEND_TIME\tPARAMETERS\tDESCRIPTION")
	for _, r := range items {
		params := formatParams(r.Parameters)
		fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			volcengine.StringValue(r.Id), volcengine.Int64Value(r.Index),
			volcengine.StringValue(r.Status), triggerOf(r.Trigger),
			firstNonEmpty(volcengine.StringValue(r.StartTime), volcengine.StringValue(r.CreateTime)),
			endTimeOr(firstNonEmpty(volcengine.StringValue(r.FinishTime), volcengine.StringValue(r.UpdateTime))),
			strings.Join(params, ","), volcengine.StringValue(r.Description))
	}
	return tw.Flush()
}

// PrintRunDetailTo 树形输出单次运行: 失败分支展开错误信息与日志尾部, 成功分支折叠.
func PrintRunDetailTo(w io.Writer, run *cp.ItemForListPipelineRunsOutput, fs []failure.Failure, url string) error {
	fmt.Fprintf(w, "运行: %s (#%d)  状态: %s  %s ~ %s\n",
		volcengine.StringValue(run.Id), volcengine.Int64Value(run.Index),
		volcengine.StringValue(run.Status),
		firstNonEmpty(volcengine.StringValue(run.StartTime), volcengine.StringValue(run.CreateTime)),
		endTimeOr(firstNonEmpty(volcengine.StringValue(run.FinishTime), volcengine.StringValue(run.UpdateTime))))
	if params := formatParams(run.Parameters); len(params) > 0 {
		fmt.Fprintf(w, "发布参数: %s\n", strings.Join(params, ", "))
	}
	for _, st := range run.Stages {
		fmt.Fprintf(w, "Stage: %s [%s]\n", volcengine.StringValue(st.Name), volcengine.StringValue(st.Status))
		for _, tk := range st.Tasks {
			fmt.Fprintf(w, "  Task: %s [%s]\n", volcengine.StringValue(tk.Name), volcengine.StringValue(tk.Status))
			for _, f := range fs {
				if f.Stage != volcengine.StringValue(st.Name) || f.Task != volcengine.StringValue(tk.Name) {
					continue
				}
				stepLabel := f.Step
				if stepLabel == "" {
					stepLabel = "(task)"
				}
				fmt.Fprintf(w, "    ✘ %s\n", stepLabel)
				fmt.Fprintf(w, "      错误: %s\n", f.Message)
				for _, line := range f.LogTail {
					fmt.Fprintf(w, "      | %s\n", truncate(line, 200))
				}
			}
		}
	}
	fmt.Fprintf(w, "控制台: %s\n", url)
	return nil
}

// PrintWorkspaces 输出到 stdout.
func PrintWorkspaces(items []*cp.ItemForListWorkspacesOutput) error {
	return PrintWorkspacesTo(stdout, items)
}

// PrintPipelines 输出到 stdout.
func PrintPipelines(items []*cp.ItemForListPipelinesOutput) error {
	return PrintPipelinesTo(stdout, items)
}

// PrintRuns 输出到 stdout.
func PrintRuns(items []*cp.ItemForListPipelineRunsOutput) error {
	return PrintRunsTo(stdout, items)
}

// PrintRunDetail 输出到 stdout.
func PrintRunDetail(run *cp.ItemForListPipelineRunsOutput, fs []failure.Failure, url string) error {
	return PrintRunDetailTo(stdout, run, fs, url)
}

// formatParams 将参数列表格式化为 key=value 串, 秘密参数脱敏.
func formatParams(params []*cp.ParameterForListPipelineRunsOutput) []string {
	out := make([]string, 0, len(params))
	for _, p := range params {
		key := volcengine.StringValue(p.Key)
		if volcengine.BoolValue(p.Secret) {
			out = append(out, key+"=***")
			continue
		}
		out = append(out, key+"="+volcengine.StringValue(p.Value))
	}
	return out
}

// triggerOf 提取触发方式.
func triggerOf(t *cp.TriggerForListPipelineRunsOutput) string {
	if t == nil {
		return ""
	}
	return volcengine.StringValue(t.Type)
}

// endTimeOr 空结束时间显示运行中.
func endTimeOr(s string) string {
	if s == "" {
		return "(运行中)"
	}
	return s
}

// firstNonEmpty 返回第一个非空串, 都空返回空串.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// truncate 截断长字符串, 避免日志单行刷屏.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "...(截断)"
}
