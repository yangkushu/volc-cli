package cmd

import (
	"strings"
	"testing"
)

func TestCheckCredentialsMissing(t *testing.T) {
	oldAK, oldSK := globalOpts.AccessKey, globalOpts.SecretKey
	t.Cleanup(func() { globalOpts.AccessKey, globalOpts.SecretKey = oldAK, oldSK })
	globalOpts.AccessKey, globalOpts.SecretKey = "", ""
	t.Setenv(envAK, "")
	t.Setenv(envSK, "")

	out, err := executeInTest(t, NewCheckCredentialsCmd())
	if err == nil || !strings.Contains(out, "未设置") {
		t.Errorf("AK/SK 未设置时应报错且提示未设置, out=%s err=%v", out, err)
	}
}
