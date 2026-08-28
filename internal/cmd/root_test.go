package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// executeInTest 执行命令并捕获输出, 参考 cobra 官方测试模式.
func executeInTest(t *testing.T, root *cobra.Command, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestRootHelp(t *testing.T) {
	out, err := executeInTest(t, NewRootCommand(), "--help")
	if err != nil {
		t.Fatalf("help 不应报错: %v", err)
	}
	for _, want := range []string{"volc-cli", "--json", "--workspace-id"} {
		if !strings.Contains(out, want) {
			t.Errorf("help 输出应包含 %q, 实际: %s", want, out)
		}
	}
}
