package utils

import "testing"

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
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := SanitizeRepo(c.in); got != c.out {
				t.Errorf("expected %q, got %q", c.out, got)
			}
		})
	}
}

func TestSanitizeRepoPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
	}{
		{
			name:  "path traversal with ../",
			input: "../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "path traversal with ../ in middle",
			input: "repo/../../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "path traversal with /../",
			input: "/../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "path traversal with absolute escape",
			input: "/repo/../../../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "path traversal with ..\\",
			input: "..\\..\\windows\\path",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "multiple ../ sequences",
			input: "../../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
		{
			name:  "path traversal after normal chars",
			input: "valid/../../etc/passwd",
			want:  "", // Should return empty for path traversal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeRepo(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeRepo(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
