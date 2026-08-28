package config

import (
	"testing"
)

func TestResolveCredentialsFlagWins(t *testing.T) {
	t.Setenv("VOLC_ACCESSKEY", "env-ak")
	t.Setenv("VOLC_SECRETKEY", "env-sk")
	ak, sk := ResolveCredentials("flag-ak", "flag-sk")
	if ak != "flag-ak" || sk != "flag-sk" {
		t.Errorf("flag 应优先, got ak=%q sk=%q", ak, sk)
	}
}

func TestResolveCredentialsEnvFallback(t *testing.T) {
	t.Setenv("VOLC_ACCESSKEY", "env-ak")
	t.Setenv("VOLC_SECRETKEY", "env-sk")
	ak, sk := ResolveCredentials("", "")
	if ak != "env-ak" || sk != "env-sk" {
		t.Errorf("env 应兜底, got ak=%q sk=%q", ak, sk)
	}
}

func TestMaskSecret(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", "未设置"},
		{"abc", "***"},
		{"AKLT1234567890abcdef", "AKLT...(len=20)"},
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
