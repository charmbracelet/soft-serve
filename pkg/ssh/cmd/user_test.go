package cmd

import (
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/sshutils"
)

// reconstructKey mirrors the logic in userCreateCommand.RunE:
// join the -k flag value with any trailing positional args and parse.
func reconstructKey(flagValue string, extraArgs []string) (string, error) {
	keyStr := strings.TrimSpace(strings.Join(append([]string{flagValue}, extraArgs...), " "))
	_, _, err := sshutils.ParseAuthorizedKey(keyStr)
	return keyStr, err
}

// TestUserCreateKeyReconstruction verifies the key-joining logic used when
// OpenSSH strips shell quoting and delivers the key as separate tokens.
//
// Note: the empty-key guard (cmd.Flags().Changed("key") && key == "")
// is exercised only via integration tests (testscript/testdata) because
// invoking RunE requires passing the checkIfAdmin PersistentPreRunE,
// which needs a full backend context.
func TestUserCreateKeyReconstruction(t *testing.T) {
	// Generate a real ed25519 key in authorized_keys format for tests.
	const testKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBzKEBMH+cKg8+8v7CJrbPBpbmMHbzSENKgmHhYRhM89 test@host"
	keyType := "ssh-ed25519"
	keyBody := "AAAAC3NzaC1lZDI1NTE5AAAAIBzKEBMH+cKg8+8v7CJrbPBpbmMHbzSENKgmHhYRhM89"
	comment := "test@host"

	tests := []struct {
		name      string
		flagValue string // value cobra captures for -k
		extraArgs []string
		wantErr   bool
		wantKey   string // expected reconstructed key string
	}{
		{
			name:      "full key in flag value (proper quoting preserved)",
			flagValue: testKey,
			extraArgs: nil,
			wantKey:   testKey,
		},
		{
			name:      "key type in flag, body as extra arg (SSH quoting stripped)",
			flagValue: keyType,
			extraArgs: []string{keyBody},
			wantKey:   keyType + " " + keyBody,
		},
		{
			name:      "key type in flag, body and comment as extra args",
			flagValue: keyType,
			extraArgs: []string{keyBody, comment},
			wantKey:   keyType + " " + keyBody + " " + comment,
		},
		{
			name:      "key type only (missing body) returns parse error",
			flagValue: keyType,
			extraArgs: nil,
			wantErr:   true,
		},
		{
			name:      "empty flag value returns parse error",
			flagValue: "",
			extraArgs: nil,
			wantErr:   true,
		},
		{
			name:      "invalid base64 body returns parse error",
			flagValue: keyType,
			extraArgs: []string{"!!!not-base64!!!"},
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reconstructKey(tc.flagValue, tc.extraArgs)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got key %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantKey {
				t.Errorf("reconstructed key = %q; want %q", got, tc.wantKey)
			}
		})
	}
}
