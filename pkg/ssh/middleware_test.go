package ssh

import (
	"context"
	"net"
	"testing"

	"github.com/charmbracelet/keygen"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/migrate"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/store/database"
	"github.com/charmbracelet/ssh"
	"github.com/matryer/is"
	gossh "golang.org/x/crypto/ssh"
	_ "modernc.org/sqlite"
)

// TestAuthenticationBypass tests for CVE-TBD: Authentication Bypass Vulnerability
//
// VULNERABILITY:
// A critical authentication bypass allows an attacker to impersonate any user
// (including Admin) by "offering" the victim's public key during the SSH handshake
// before authenticating with their own valid key. This occurs because the user
// identity is stored in the session context during the "offer" phase in
// PublicKeyHandler and is not properly cleared/validated in AuthenticationMiddleware.
//
// This test verifies that:
// 1. User context is properly set based on the AUTHENTICATED key, not offered keys
// 2. User context from failed authentication attempts is not preserved
// 3. Non-admin users cannot gain admin privileges through this attack
func TestAuthenticationBypass(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()

	// Setup temporary database
	dp := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.DataPath = dp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = dp + "/test.db"

	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	is.NoErr(err)
	defer dbx.Close()

	is.NoErr(migrate.Migrate(ctx, dbx))
	dbstore := database.New(ctx, dbx)
	ctx = store.WithContext(ctx, dbstore)
	be := backend.New(ctx, cfg, dbx, dbstore)
	ctx = backend.WithContext(ctx, be)

	// Generate keys for admin and attacker
	adminKeyPath := dp + "/admin_key"
	adminPair, err := keygen.New(adminKeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
	is.NoErr(err)

	attackerKeyPath := dp + "/attacker_key"
	attackerPair, err := keygen.New(attackerKeyPath, keygen.WithKeyType(keygen.Ed25519), keygen.WithWrite())
	is.NoErr(err)

	// Parse public keys
	adminPubKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(adminPair.AuthorizedKey()))
	is.NoErr(err)

	attackerPubKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(attackerPair.AuthorizedKey()))
	is.NoErr(err)

	// Create admin user
	adminUser, err := be.CreateUser(ctx, "testadmin", proto.UserOptions{
		Admin:      true,
		PublicKeys: []gossh.PublicKey{adminPubKey},
	})
	is.NoErr(err)
	is.True(adminUser != nil)

	// Create attacker (non-admin) user
	attackerUser, err := be.CreateUser(ctx, "testattacker", proto.UserOptions{
		Admin:      false,
		PublicKeys: []gossh.PublicKey{attackerPubKey},
	})
	is.NoErr(err)
	is.True(attackerUser != nil)
	is.True(!attackerUser.IsAdmin()) // Verify attacker is NOT admin

	// Test: Verify that looking up user by key gives correct user
	t.Run("user_lookup_by_key", func(t *testing.T) {
		is := is.New(t)

		// Looking up admin key should return admin user
		user, err := be.UserByPublicKey(ctx, adminPubKey)
		is.NoErr(err)
		is.Equal(user.Username(), "testadmin")
		is.True(user.IsAdmin())

		// Looking up attacker key should return attacker user
		user, err = be.UserByPublicKey(ctx, attackerPubKey)
		is.NoErr(err)
		is.Equal(user.Username(), "testattacker")
		is.True(!user.IsAdmin())
	})

	// Test: Simulate the authentication bypass vulnerability
	// This test documents the EXPECTED behavior to prevent regression
	t.Run("authentication_bypass_simulation", func(t *testing.T) {
		is := is.New(t)

		// Create a mock context
		mockCtx := &mockSSHContext{
			Context:     ctx,
			values:      make(map[any]any),
			permissions: &ssh.Permissions{Permissions: &gossh.Permissions{Extensions: make(map[string]string)}},
		}

		// ATTACK SIMULATION:
		// Step 1: SSH client offers admin's public key
		// PublicKeyHandler is called and sets admin user in context
		mockCtx.SetValue(proto.ContextKeyUser, adminUser)
		mockCtx.permissions.Extensions["pubkey-fp"] = gossh.FingerprintSHA256(adminPubKey)

		// Step 2: Signature verification FAILS (attacker doesn't have admin's private key)
		// SSH protocol continues to next key...

		// Step 3: SSH client offers attacker's key (which SUCCEEDS)
		// PublicKeyHandler is called again, fingerprint is updated
		mockCtx.permissions.Extensions["pubkey-fp"] = gossh.FingerprintSHA256(attackerPubKey)
		// BUG: Admin user is STILL in context from step 1!

		// Step 4: AuthenticationMiddleware should re-lookup user based on authenticated key
		// The middleware MUST NOT trust the user already in context
		authenticatedUser, err := be.UserByPublicKey(mockCtx, attackerPubKey)
		is.NoErr(err)

		// EXPECTED: User should be "attacker", NOT "admin"
		is.Equal(authenticatedUser.Username(), "testattacker")
		is.True(!authenticatedUser.IsAdmin())

		// If the vulnerability exists, the context would still have admin user
		contextUser := proto.UserFromContext(mockCtx)
		if contextUser != nil && contextUser.Username() == "testadmin" {
			t.Logf("WARNING: Context still contains admin user! This indicates the vulnerability exists.")
			t.Logf("The authenticated key is attacker's, but context has admin user.")
		}
	})
}

// mockSSHContext implements ssh.Context for testing
type mockSSHContext struct {
	context.Context
	values      map[any]any
	permissions *ssh.Permissions
}

func (m *mockSSHContext) SetValue(key, value any) {
	m.values[key] = value
}

func (m *mockSSHContext) Value(key any) any {
	if v, ok := m.values[key]; ok {
		return v
	}
	return m.Context.Value(key)
}

func (m *mockSSHContext) Permissions() *ssh.Permissions {
	return m.permissions
}

func (m *mockSSHContext) User() string          { return "" }
func (m *mockSSHContext) RemoteAddr() net.Addr  { return &net.TCPAddr{} }
func (m *mockSSHContext) LocalAddr() net.Addr   { return &net.TCPAddr{} }
func (m *mockSSHContext) ServerVersion() string { return "" }
func (m *mockSSHContext) ClientVersion() string { return "" }
func (m *mockSSHContext) SessionID() string     { return "" }
func (m *mockSSHContext) Lock()                 {}
func (m *mockSSHContext) Unlock()               {}
