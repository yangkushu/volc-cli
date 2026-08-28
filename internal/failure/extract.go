// Package failure 从流水线执行记录中提取失败定位信息.
package failure

import (
	"strings"

	"github.com/volcengine/volc-sdk-golang/service/codePipeline/models"
)

// messageKeys Message 取值优先级: 命中这些 key 的 KV 优先作为 Message.
var messageKeys = []string{"error", "message", "errorMsg", "reason", "log"}

// Failure 一处失败的定位路径与错误信息.
type Failure struct {
	Stage   string          `json:"Stage"`
	Task    string          `json:"Task"`
	Step    string          `json:"Step"`
	Message string          `json:"Message"`
	Details []models.KVPair `json:"Details,omitempty"`
}

// Extract 遍历 Stage→Task→Step, 收集所有 Status==Failed 的 step 失败信息.
func Extract(record *models.PipelineRecord) []Failure {
	var out []Failure
	if record == nil {
		return out
	}
	for _, st := range record.Stages {
		for _, tk := range st.Tasks {
			for _, sp := range tk.Steps {
				if sp.Status != "Failed" {
					continue
				}
				out = append(out, Failure{
					Stage:   st.Name,
					Task:    tk.Name,
					Step:    sp.Name,
					Message: pickMessage(sp.Result),
					Details: sp.Result,
				})
			}
		}
	}
	return out
}

// pickMessage 按优先级选可读错误信息, 全部为空时回退第一个非空 KV.
func pickMessage(kvs []models.KVPair) string {
	for _, k := range messageKeys {
		for _, kv := range kvs {
			if strings.EqualFold(kv.Key, k) && strings.TrimSpace(kv.Value) != "" {
				return kv.Value
			}
		}
	}
	for _, kv := range kvs {
		if strings.TrimSpace(kv.Value) != "" {
			return kv.Value
		}
	}
	return "详见控制台日志"
}
