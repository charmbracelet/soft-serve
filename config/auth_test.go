package config

import (
	"testing"

	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/gliderlabs/ssh"
	"github.com/matryer/is"
)

func TestAuth(t *testing.T) {
	adminKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH a@b"
	adminPk, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(adminKey))
	dummyKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b"
	dummyPk, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(dummyKey))
	cases := []struct {
		name   string
		cfg    Config
		repo   string
		key    ssh.PublicKey
		access git.AccessLevel
	}{
		// Repo access
		{
			name:   "anon access: no-access, anonymous user",
			access: git.NoAccess,
			repo:   "foo",
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
			},
		},
		{
			name:   "anon access: no-access, anonymous user with admin user",
			access: git.NoAccess,
			repo:   "foo",
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, authd user",
			key:    dummyPk,
			repo:   "foo",
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, anonymous user with admin user",
			key:    dummyPk,
			repo:   "foo",
			access: git.NoAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, admin user",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-only, anonymous user",
			repo:   "foo",
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
			},
		},
		{
			name:   "anon access: read-only, authd user",
			repo:   "foo",
			key:    dummyPk,
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-only, admin user",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-write, anonymous user",
			repo:   "foo",
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-write",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
			},
		},
		{
			name:   "anon access: read-write, authd user",
			repo:   "foo",
			key:    dummyPk,
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-write",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		}, {
			name:   "anon access: read-write, admin user",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "read-write",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, anonymous user",
			repo:   "foo",
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, authd user",
			repo:   "foo",
			key:    dummyPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		}, {
			name:   "anon access: admin-access, admin user",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
					},
				},
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},

		// Collabs
		{
			name:   "anon access: no-access, authd user, collab",
			key:    dummyPk,
			repo:   "foo",
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name: "user",
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, authd user, collab, private repo",
			key:    dummyPk,
			repo:   "foo",
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo:    "foo",
						Private: true,
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name: "user",
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, admin user, collab, private repo",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Repos: []RepoConfig{
					{
						Repo:    "foo",
						Private: true,
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name:  "admin",
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-only, authd user, collab, private repo",
			repo:   "foo",
			key:    dummyPk,
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Repos: []RepoConfig{
					{
						Repo:    "foo",
						Private: true,
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name: "user",
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, anonymous user, collab",
			repo:   "foo",
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
						Collabs: []string{
							"user",
						},
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, authd user, collab",
			repo:   "foo",
			key:    dummyPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name: "user",
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		}, {
			name:   "anon access: admin-access, admin user, collab",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Repos: []RepoConfig{
					{
						Repo: "foo",
						Collabs: []string{
							"user",
						},
					},
				},
				Users: []User{
					{
						Name:  "admin",
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},

		// New repo
		{
			name:   "anon access: no-access, anonymous user, new repo",
			access: git.NoAccess,
			repo:   "foo",
			cfg: Config{
				AnonAccess: "no-access",
			},
		},
		{
			name:   "anon access: no-access, authd user, new repo",
			key:    dummyPk,
			repo:   "foo",
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, authd user, new repo, with user",
			key:    dummyPk,
			repo:   "foo",
			access: git.NoAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Users: []User{
					{
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: no-access, admin user, new repo",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "no-access",
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-only, anonymous user, new repo",
			repo:   "foo",
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "read-only",
			},
		},
		{
			name:   "anon access: read-only, authd user, new repo",
			repo:   "foo",
			key:    dummyPk,
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-only, admin user, new repo",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "read-only",
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-write, anonymous user, new repo",
			repo:   "foo",
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-write",
			},
		},
		{
			name:   "anon access: read-write, authd user, new repo",
			repo:   "foo",
			key:    dummyPk,
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-write",
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: read-write, admin user, new repo",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "read-write",
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, anonymous user, new repo",
			repo:   "foo",
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
			},
		},
		{
			name:   "anon access: admin-access, authd user, new repo",
			repo:   "foo",
			key:    dummyPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Users: []User{
					{
						PublicKeys: []string{
							dummyKey,
						},
					},
				},
			},
		},
		{
			name:   "anon access: admin-access, admin user, new repo",
			repo:   "foo",
			key:    adminPk,
			access: git.AdminAccess,
			cfg: Config{
				AnonAccess: "admin-access",
				Users: []User{
					{
						Admin: true,
						PublicKeys: []string{
							adminKey,
						},
					},
				},
			},
		},

		// No users
		{
			name:   "anon access: read-only, no users",
			repo:   "foo",
			access: git.ReadOnlyAccess,
			cfg: Config{
				AnonAccess: "read-only",
			},
		},
		{
			name:   "anon access: read-write, no users",
			repo:   "foo",
			access: git.ReadWriteAccess,
			cfg: Config{
				AnonAccess: "read-write",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			is := is.New(t)
			al := c.cfg.accessForKey(c.repo, c.key)
			is.Equal(al, c.access)
		})
	}
}
