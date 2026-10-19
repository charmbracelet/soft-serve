package backend

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
)

// RemoteGitEnv returns the environment for a git subprocess that touches
// user-supplied remotes: the guards that keep the remotes' validation honest
// (see ssrf.GitEnv) and the server's own ssh client command. It must reach
// every git command touching a remote, or the validation means nothing.
func RemoteGitEnv(cfg *config.Config, remotes ...ssrf.ValidatedGitRemote) []string {
	return append(ssrf.GitEnv(remotes...), gitSSHCommandEnv(cfg))
}

// gitSSHCommandEnv is the GIT_SSH_COMMAND entry for the server's ssh client:
// its own known_hosts under the data path, the configured host key policy,
// no prompting, and the client key.
func gitSSHCommandEnv(cfg *config.Config) string {
	knownHosts := filepath.Join(cfg.DataPath, "ssh", "known_hosts")

	// ssh splits an option value on whitespace even when the shell handed it
	// over as one argument, but honours double quotes inside the value. The
	// data path is kept free of double quotes and newlines by config
	// validation, since nothing escapes them inside ssh's quotes.
	knownHostsOpt := `UserKnownHostsFile="` + sshEscape(knownHosts) + `"`

	// git runs this through the shell, so each argument is single-quoted.
	return "GIT_SSH_COMMAND=ssh" +
		" -o " + shellQuote(knownHostsOpt) +
		" -o StrictHostKeyChecking=" + cfg.SSH.ClientStrictHostKeyChecking +
		" -o BatchMode=yes" +
		" -i " + shellQuote(sshEscape(cfg.SSH.ClientKeyPath))
}

// sshEscape doubles % so ssh's token expansion (%d, %u, ...) leaves the
// path alone. ssh expands tokens in identity files given with -i as well.
func sshEscape(path string) string {
	return strings.ReplaceAll(path, "%", "%%")
}

// shellQuote wraps s in single quotes so the shell takes it literally.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
