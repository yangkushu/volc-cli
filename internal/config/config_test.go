package config

import (
	"testing"
)

func TestResolveCredentialsFlagWins(t *testing.T) {
	t.Setenv("VOLC_ACCESS_KEY", "env-ak")
	t.Setenv("VOLC_SECRET_KEY", "env-sk")
	ak, sk := ResolveCredentials("flag-ak", "flag-sk")
	if ak != "flag-ak" || sk != "flag-sk" {
		t.Errorf("flag 应优先, got ak=%q sk=%q", ak, sk)
	}
}

func TestResolveCredentialsEnvFallback(t *testing.T) {
	t.Setenv("VOLC_ACCESS_KEY", "env-ak")
	t.Setenv("VOLC_SECRET_KEY", "env-sk")
	ak, sk := ResolveCredentials("", "")
	if ak != "env-ak" || sk != "env-sk" {
		t.Errorf("env 应兜底, got ak=%q sk=%q", ak, sk)
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "未设置"},
		{"abc", "*** (len=3)"},
		{"AKLT1234567890abcdef", "*** (len=20)"},
	}
	for _, c := range cases {
		if got := MaskSecret(c.in); got != c.want {
			t.Errorf("MaskSecret(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestResolveWorkspaceIdMissing(t *testing.T) {
	if _, err := ResolveWorkspaceId(""); err == nil {
		t.Error("未设置 workspace 应报错")
	}
}

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
