package utils

import (
	"strings"
	"testing"
)

func TestValidateRepo(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, repo := range []string{
			"lower",
			"Upper",
			"with-dash",
			"with/slash",
			"withnumb3r5",
			"with.dot",
			"with_underline",
			"a/b/c",
			"foo.github/bar",
			"gitfoo/bar.gitx",
		} {
			t.Run(repo, func(t *testing.T) {
				if err := ValidateRepo(repo); err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			})
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, repo := range []string{
			"with$",
			"with@",
			"with!",
			"",
			".",
			"..",
			"nested.git/refs/heads/x",
			"A.GIT/b",
			"victim.git/hooks",
			"foo/.git",
			"foo/../bar",
			"foo//bar",
			"/leading",
			"trailing/",
			"suffix.git",
			"double.git.git",
		} {
			t.Run(repo, func(t *testing.T) {
				if err := ValidateRepo(repo); err == nil {
					t.Error("expected an error, got nil")
				}
			})
		}
	})
}

func TestSanitizeRepo(t *testing.T) {
	cases := []struct {
		in, out string
	}{
		{"lower", "lower"},
		{"Upper", "Upper"},
		{"with/slash", "with/slash"},
		{"with.dot", "with.dot"},
		{"/with_forward_slash", "with_forward_slash"},
		{"withgitsuffix.git", "withgitsuffix"},
		{"double.git.git", "double"},
		{"triple.git.git.git", "triple"},
		{"..git", ""},
		{".git", ""},
		{"foo/.git", "foo"},
		{"foo/../bar", "bar"},
		{"../../etc", "etc"},
		{"trailing/", "trailing"},
		{"nested.git/refs/heads/x", "nested.git/refs/heads/x"},
		{"a/..git", "a"},
		{"a/b/...git", "a"},
		{"a/b/..git", "a/b"},
		{"foo.git/.git", "foo"},
		{strings.Repeat(".git", 20000), ""},
		{strings.Repeat("/.git", 20000), ""},
		{"x" + strings.Repeat(".git", 20000), "x"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := SanitizeRepo(c.in)
			if got != c.out {
				t.Errorf("expected %q, got %q", c.out, got)
			}
			if again := SanitizeRepo(got); again != got {
				t.Errorf("not a fixed point: %q -> %q", got, again)
			}
		})
	}
}
