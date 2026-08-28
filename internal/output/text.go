package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
	"kuopin/volc-cli/internal/failure"
)

var stdout io.Writer = os.Stdout

// PrintPipelinesTo 以表格输出流水线列表.
func PrintPipelinesTo(w io.Writer, items []models.Pipeline) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无流水线")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tLAST_STATUS\tUPDATE_TIME\tTRIGGERER")
	for _, p := range items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", p.Id, p.Name, p.LastStatus, p.UpdateTime, p.Triggerer)
	}
	return tw.Flush()
}

// PrintRecordsTo 以表格输出执行记录(含发布参数 DynamicEnvs).
func PrintRecordsTo(w io.Writer, items []models.PipelineRecord) error {
	if len(items) == 0 {
		fmt.Fprintln(w, "无执行记录")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tTRIGGER_MODE\tSTART_TIME\tEND_TIME\tDYNAMIC_ENVS\tDESCRIPTION")
	for _, r := range items {
		envs := make([]string, 0, len(r.DynamicEnvs))
		for _, kv := range r.DynamicEnvs {
			envs = append(envs, kv.Key+"="+kv.Value)
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Id, r.Status, r.TriggerMode, r.StartTime, endTimeOr(r.EndTime), strings.Join(envs, ","), r.Description)
	}
	return tw.Flush()
}

// PrintRecordDetailTo 树形输出单条记录: 失败分支展开错误信息, 成功分支折叠.
func PrintRecordDetailTo(w io.Writer, record *models.PipelineRecord, fs []failure.Failure, url string) error {
	fmt.Fprintf(w, "记录: %s  状态: %s  %s ~ %s\n", record.Id, record.Status, record.StartTime, endTimeOr(record.EndTime))
	if len(record.DynamicEnvs) > 0 {
		envs := make([]string, 0, len(record.DynamicEnvs))
		for _, kv := range record.DynamicEnvs {
			envs = append(envs, kv.Key+"="+kv.Value)
		}
		fmt.Fprintf(w, "发布参数: %s\n", strings.Join(envs, ", "))
	}
	for _, st := range record.Stages {
		fmt.Fprintf(w, "Stage: %s [%s]\n", st.Name, st.Status)
		for _, tk := range st.Tasks {
			fmt.Fprintf(w, "  Task: %s [%s]\n", tk.Name, tk.Status)
			for _, sp := range tk.Steps {
				if sp.Status == "Failed" {
					fmt.Fprintf(w, "    ✘ Step: %s [%s]\n", sp.Name, sp.Status)
					for _, f := range fs {
						if f.Stage == st.Name && f.Task == tk.Name && f.Step == sp.Name {
							fmt.Fprintf(w, "      错误: %s\n", f.Message)
							for _, kv := range f.Details {
								fmt.Fprintf(w, "        %s: %s\n", kv.Key, truncate(kv.Value, 200))
							}
						}
					}
				} else {
					fmt.Fprintf(w, "    ✔ Step: %s [%s]\n", sp.Name, sp.Status)
				}
			}
		}
	}
	fmt.Fprintf(w, "控制台: %s\n", url)
	return nil
}

// PrintPipelines 输出到 stdout.
func PrintPipelines(items []models.Pipeline) error {
	return PrintPipelinesTo(stdout, items)
}

// PrintRecords 输出到 stdout.
func PrintRecords(items []models.PipelineRecord) error {
	return PrintRecordsTo(stdout, items)
}

// PrintRecordDetail 输出到 stdout.
func PrintRecordDetail(record *models.PipelineRecord, fs []failure.Failure, url string) error {
	return PrintRecordDetailTo(stdout, record, fs, url)
}

// endTimeOr 空结束时间显示运行中.
func endTimeOr(s string) string {
	if s == "" {
		return "(运行中)"
	}
	return s
}

// truncate 截断长字符串, 避免 Result KV 单值刷屏.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "...(截断)"
}
