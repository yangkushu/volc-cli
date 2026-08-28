// Package output 负责终端渲染, text 为人类可读, json 为结构化输出.
package output

import (
	"encoding/json"
	"io"
)

// PrintJSONTo 将 v 以缩进 JSON 写入 w.
func PrintJSONTo(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// PrintJSON 输出到 stdout.
func PrintJSON(v any) error {
	return PrintJSONTo(stdout, v)
}
