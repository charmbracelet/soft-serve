package config

import (
	"os"
	"testing"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/gliderlabs/ssh"
	"github.com/matryer/is"
)

func TestAuth(t *testing.T) {
	is := is.New(t)
	adminKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINMwLvyV3ouVrTysUYGoJdl5Vgn5BACKov+n9PlzfPwH a@b"
	adminPk, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(adminKey))
	dummyKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFxIobhwtfdwN7m1TFt9wx3PsfvcAkISGPxmbmbauST8 a@b"
	dummyPk, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(dummyKey))
	dp := t.TempDir()
	is.NoErr(os.Setenv("SOFT_SERVE_INITIAL_ADMIN_KEY", adminKey))
	is.NoErr(os.Setenv("SOFT_SERVE_DATA_PATH", dp))
	t.Cleanup(func() {
		is.NoErr(os.Unsetenv("SOFT_SERVE_INITIAL_ADMIN_KEY"))
		is.NoErr(os.Unsetenv("SOFT_SERVE_DATA_PATH"))
		is.NoErr(os.RemoveAll(dp))
	})
	cfg := DefaultConfig()
	cases := []struct {
		name           string
		repo           string
		key            ssh.PublicKey
		anonAccess     proto.AccessLevel
		expectedAccess proto.AccessLevel
	}{
		// Repo access
		{
			name:           "anon access: no-access, anonymous user",
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.NoAccess,
			repo:           "foo",
		},
		{
			name:           "anon access: no-access, anonymous user with admin user",
			expectedAccess: proto.NoAccess,
			anonAccess:     proto.NoAccess,
			repo:           "foo",
		},
		{
			name:           "anon access: no-access, authd user",
			key:            dummyPk,
			repo:           "foo",
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.ReadOnlyAccess,
		},
		{
			name:           "anon access: no-access, admin user",
			repo:           "foo",
			key:            adminPk,
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.AdminAccess,
		},
		{
			name:           "anon access: read-only, anonymous user",
			repo:           "foo",
			anonAccess:     proto.ReadOnlyAccess,
			expectedAccess: proto.ReadOnlyAccess,
		},
		{
			name:           "anon access: read-only, authd user",
			repo:           "foo",
			key:            dummyPk,
			anonAccess:     proto.ReadOnlyAccess,
			expectedAccess: proto.ReadOnlyAccess,
		},
		{
			name:           "anon access: read-only, admin user",
			repo:           "foo",
			key:            adminPk,
			anonAccess:     proto.ReadOnlyAccess,
			expectedAccess: proto.AdminAccess,
		},
		{
			name:           "anon access: read-write, anonymous user",
			repo:           "foo",
			anonAccess:     proto.ReadWriteAccess,
			expectedAccess: proto.ReadWriteAccess,
		},
		{
			name:           "anon access: read-write, authd user",
			repo:           "foo",
			key:            dummyPk,
			anonAccess:     proto.ReadWriteAccess,
			expectedAccess: proto.ReadWriteAccess,
		},
		{
			name:           "anon access: read-write, admin user",
			repo:           "foo",
			key:            adminPk,
			anonAccess:     proto.ReadWriteAccess,
			expectedAccess: proto.AdminAccess,
		},
		{
			name:           "anon access: admin-access, anonymous user",
			repo:           "foo",
			anonAccess:     proto.AdminAccess,
			expectedAccess: proto.AdminAccess,
		},
		{
			name:           "anon access: admin-access, authd user",
			repo:           "foo",
			key:            dummyPk,
			anonAccess:     proto.AdminAccess,
			expectedAccess: proto.AdminAccess,
		},
		{
			name:           "anon access: admin-access, admin user",
			repo:           "foo",
			key:            adminPk,
			anonAccess:     proto.AdminAccess,
			expectedAccess: proto.AdminAccess,
		},

		// TODO: fix this
		// // Collabs
		// {
		// 	name:           "anon access: no-access, authd user, collab",
		// 	key:            dummyPk,
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "no-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo: "foo",
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name: "user",
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: no-access, authd user, collab, private repo",
		// 	key:            dummyPk,
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "no-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo:    "foo",
		// 				Private: true,
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name: "user",
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: no-access, admin user, collab, private repo",
		// 	repo:           "foo",
		// 	key:            adminPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "no-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo:    "foo",
		// 				Private: true,
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name:  "admin",
		// 				Admin: true,
		// 				PublicKeys: []string{
		// 					adminKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: read-only, authd user, collab, private repo",
		// 	repo:           "foo",
		// 	key:            dummyPk,
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-only",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo:    "foo",
		// 				Private: true,
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name: "user",
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: admin-access, anonymous user, collab",
		// 	repo:           "foo",
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo: "foo",
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: admin-access, authd user, collab",
		// 	repo:           "foo",
		// 	key:            dummyPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo: "foo",
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name: "user",
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// }, {
		// 	name:           "anon access: admin-access, admin user, collab",
		// 	repo:           "foo",
		// 	key:            adminPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 		Repos: []RepoConfig{
		// 			{
		// 				Repo: "foo",
		// 				Collabs: []string{
		// 					"user",
		// 				},
		// 			},
		// 		},
		// 		Users: []User{
		// 			{
		// 				Name:  "admin",
		// 				Admin: true,
		// 				PublicKeys: []string{
		// 					adminKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },

		// New repo
		{
			name:           "anon access: no-access, anonymous user, new repo",
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.NoAccess,
			repo:           "foo",
		},
		{
			name:           "anon access: no-access, authd user, new repo",
			key:            dummyPk,
			repo:           "foo",
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.ReadOnlyAccess,
		},
		{
			name:           "anon access: no-access, admin user, new repo",
			repo:           "foo",
			key:            adminPk,
			anonAccess:     proto.NoAccess,
			expectedAccess: proto.AdminAccess,
		},
		// {
		// 	name:           "anon access: read-only, anonymous user, new repo",
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadOnlyAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-only",
		// 	},
		// },
		// {
		// 	name:           "anon access: read-only, authd user, new repo",
		// 	repo:           "foo",
		// 	key:            dummyPk,
		// 	expectedAccess: proto.ReadOnlyAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-only",
		// 		Users: []User{
		// 			{
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: read-only, admin user, new repo",
		// 	repo:           "foo",
		// 	key:            adminPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-only",
		// 		Users: []User{
		// 			{
		// 				Admin: true,
		// 				PublicKeys: []string{
		// 					adminKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: read-write, anonymous user, new repo",
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-write",
		// 	},
		// },
		// {
		// 	name:           "anon access: read-write, authd user, new repo",
		// 	repo:           "foo",
		// 	key:            dummyPk,
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-write",
		// 		Users: []User{
		// 			{
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: read-write, admin user, new repo",
		// 	repo:           "foo",
		// 	key:            adminPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-write",
		// 		Users: []User{
		// 			{
		// 				Admin: true,
		// 				PublicKeys: []string{
		// 					adminKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: admin-access, anonymous user, new repo",
		// 	repo:           "foo",
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 	},
		// },
		// {
		// 	name:           "anon access: admin-access, authd user, new repo",
		// 	repo:           "foo",
		// 	key:            dummyPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 		Users: []User{
		// 			{
		// 				PublicKeys: []string{
		// 					dummyKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:           "anon access: admin-access, admin user, new repo",
		// 	repo:           "foo",
		// 	key:            adminPk,
		// 	expectedAccess: proto.AdminAccess,
		// 	cfg: Config{
		// 		AnonAccess: "admin-access",
		// 		Users: []User{
		// 			{
		// 				Admin: true,
		// 				PublicKeys: []string{
		// 					adminKey,
		// 				},
		// 			},
		// 		},
		// 	},
		// },

		// // No users
		// {
		// 	name:           "anon access: read-only, no users",
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadOnlyAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-only",
		// 	},
		// },
		// {
		// 	name:           "anon access: read-write, no users",
		// 	repo:           "foo",
		// 	expectedAccess: proto.ReadWriteAccess,
		// 	cfg: Config{
		// 		AnonAccess: "read-write",
		// 	},
		// },
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			is := is.New(t)
			cfg.AnonAccess = c.anonAccess
			al := cfg.accessForKey(c.repo, c.key)
			is.Equal(al, c.expectedAccess)
		})
	}
}
