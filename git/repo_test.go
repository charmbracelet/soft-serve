package git

import (
	gopath "path"
	"strings"
	"testing"
)

func TestGitPathHelpersUseForwardSlashes(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"join", gopath.Join("dot_config", "bat"), "dot_config/bat"},
		{"dir", gopath.Dir("dot_config/bat"), "dot_config"},
		{"clean nested", gopath.Clean("dot_config//bat"), "dot_config/bat"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("got %q, want %q", tc.got, tc.want)
			}
			if strings.Contains(tc.got, `\`) {
				t.Fatalf("git paths must use forward slashes, got %q", tc.got)
			}
		})
	}
}

func TestTreePathCleansGitPaths(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"dot_config/bat", "dot_config/bat"},
		{"dot_config//bat", "dot_config/bat"},
		{".", ""},
		{"", ""},
	} {
		got := gopath.Clean(tc.in)
		if got == "." {
			got = ""
		}
		if got != tc.want {
			t.Fatalf("clean(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
