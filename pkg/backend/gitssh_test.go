package backend

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

func TestGitSSHCommandEnv(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{
			name: "accept-new",
			cfg: config.Config{
				DataPath: "/data",
				SSH: config.SSHConfig{
					ClientKeyPath:               "/data/ssh/client_ed25519",
					ClientStrictHostKeyChecking: config.StrictHostKeyCheckingAcceptNew,
				},
			},
			want: `GIT_SSH_COMMAND=ssh -o 'UserKnownHostsFile="/data/ssh/known_hosts"'` +
				` -o StrictHostKeyChecking=accept-new -o BatchMode=yes -i '/data/ssh/client_ed25519'`,
		},
		{
			name: "yes",
			cfg: config.Config{
				DataPath: "/data",
				SSH: config.SSHConfig{
					ClientKeyPath:               "/data/ssh/client_ed25519",
					ClientStrictHostKeyChecking: config.StrictHostKeyCheckingYes,
				},
			},
			want: `GIT_SSH_COMMAND=ssh -o 'UserKnownHostsFile="/data/ssh/known_hosts"'` +
				` -o StrictHostKeyChecking=yes -o BatchMode=yes -i '/data/ssh/client_ed25519'`,
		},
		{
			// ssh splits option values on whitespace regardless of shell
			// quoting, so the path inside the option must be double-quoted
			// or known_hosts ends up at a relative "Support/..." path.
			name: "space in data path",
			cfg: config.Config{
				DataPath: "/Users/me/Library/Application Support/soft-serve",
				SSH: config.SSHConfig{
					ClientKeyPath:               "/Users/me/Library/Application Support/soft-serve/ssh/client_ed25519",
					ClientStrictHostKeyChecking: config.StrictHostKeyCheckingAcceptNew,
				},
			},
			want: `GIT_SSH_COMMAND=ssh -o 'UserKnownHostsFile="/Users/me/Library/Application Support/soft-serve/ssh/known_hosts"'` +
				` -o StrictHostKeyChecking=accept-new -o BatchMode=yes -i '/Users/me/Library/Application Support/soft-serve/ssh/client_ed25519'`,
		},
		{
			// $ and backticks must not expand in the shell; ' must survive
			// the single quoting; % is ssh's token escape and is doubled.
			name: "shell and ssh metacharacters",
			cfg: config.Config{
				DataPath: "/it's/$HOME/`id`/100%",
				SSH: config.SSHConfig{
					ClientKeyPath:               "/it's/$HOME/`id`/100%/key",
					ClientStrictHostKeyChecking: config.StrictHostKeyCheckingAcceptNew,
				},
			},
			want: `GIT_SSH_COMMAND=ssh -o 'UserKnownHostsFile="/it'\''s/$HOME/` + "`id`" + `/100%%/ssh/known_hosts"'` +
				` -o StrictHostKeyChecking=accept-new -o BatchMode=yes -i '/it'\''s/$HOME/` + "`id`" + `/100%%/key'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// known_hosts is joined with the platform separator; the key path
			// is used as configured.
			want := tt.want
			if filepath.Separator != '/' {
				const prefix, suffix = `UserKnownHostsFile="`, `"'`
				i := strings.Index(want, prefix) + len(prefix)
				j := i + strings.Index(want[i:], suffix)
				want = want[:i] + filepath.FromSlash(want[i:j]) + want[j:]
			}
			if got := gitSSHCommandEnv(&tt.cfg); got != want {
				t.Errorf("gitSSHCommandEnv() =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// TestRemoteGitEnv verifies the one composition point carries both the
// remote guards and the ssh command, so a caller cannot pick up one without
// the other.
func TestRemoteGitEnv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DataPath = "/srv/soft"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	env := RemoteGitEnv(cfg, ssrf.ValidatedGitRemote{Transport: ssrf.GitTransportHTTP})
	if !slices.Contains(env, "GIT_CONFIG_KEY_0=http.followRedirects") || !slices.Contains(env, "GIT_CONFIG_VALUE_0=false") {
		t.Errorf("RemoteGitEnv() lacks the redirect guard: %v", env)
	}
	knownHosts := filepath.Join(cfg.DataPath, "ssh", "known_hosts")
	if !slices.ContainsFunc(env, func(e string) bool {
		return strings.HasPrefix(e, "GIT_SSH_COMMAND=ssh ") &&
			strings.Contains(e, `UserKnownHostsFile="`+knownHosts+`"`) &&
			strings.Contains(e, "-i '"+cfg.SSH.ClientKeyPath+"'") &&
			strings.Contains(e, "StrictHostKeyChecking=accept-new")
	}) {
		t.Errorf("RemoteGitEnv() lacks the ssh command: %v", env)
	}
}
