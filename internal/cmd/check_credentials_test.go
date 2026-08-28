package cmd

import (
	"strings"
	"testing"

	"kuopin/volc-cli/internal/opts"
)

func TestCheckCredentialsMissing(t *testing.T) {
	oldAK, oldSK := opts.Global.AccessKey, opts.Global.SecretKey
	t.Cleanup(func() { opts.Global.AccessKey, opts.Global.SecretKey = oldAK, oldSK })
	opts.Global.AccessKey, opts.Global.SecretKey = "", ""
	t.Setenv(envAK, "")
	t.Setenv(envSK, "")

	out, err := executeInTest(t, NewCheckCredentialsCmd())
	if err == nil || !strings.Contains(out, "未设置") {
		t.Errorf("AK/SK 未设置时应报错且提示未设置, out=%s err=%v", out, err)
	}
}
