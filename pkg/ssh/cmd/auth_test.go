package cmd

import (
	"errors"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/matryer/is"
)

// TestGlobalCommandsIgnoreRepositoryAccess is the regression test for the
// privilege escalation where a non-admin could run global admin commands by
// creating a repository named after the command's first argument. `user
// set-admin victim true` used to be authorized by the caller's admin access
// to a repository they happened to own called "victim".
func TestGlobalCommandsIgnoreRepositoryAccess(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	attackerCtx := withUser(t, ctx, be, "attacker", false)

	// The attacker owns a repository named "victim", so they have
	// AdminAccess *to that repository*.
	attacker := proto.UserFromContext(attackerCtx)
	_, err := be.CreateRepository(attackerCtx, "victim", attacker, proto.RepositoryOptions{})
	is.NoErr(err)
	is.Equal(be.AccessLevelForUser(attackerCtx, "victim", attacker), access.AdminAccess)

	// Repository admin access must not authorize global user commands.
	for _, args := range [][]string{
		{"set-admin", "victim", "true"},
		{"create", "victim2"},
		{"delete", "victim"},
		{"list"},
		{"info", "victim"},
		{"set-username", "victim", "victim3"},
	} {
		if err := runUser(t, attackerCtx, args...); !errors.Is(err, proto.ErrUnauthorized) {
			t.Errorf("user %v: expected ErrUnauthorized, got %v", args, err)
		}
	}

	// The attacker must not have become an admin.
	reloaded, err := be.User(ctx, "attacker")
	is.NoErr(err)
	is.Equal(reloaded.IsAdmin(), false)
}

// TestGlobalCommandsIgnoreAnonAdminAccess covers the variant that needs no
// repository at all: with anon-access set to admin-access,
// AccessLevelForUser returns AdminAccess to any authenticated user for a
// nonexistent repository name.
func TestGlobalCommandsIgnoreAnonAdminAccess(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)
	is.NoErr(be.SetAnonAccess(ctx, access.AdminAccess))

	attackerCtx := withUser(t, ctx, be, "attacker", false)
	attacker := proto.UserFromContext(attackerCtx)
	is.Equal(be.AccessLevelForUser(attackerCtx, "nonexistent", attacker), access.AdminAccess)

	err := runUser(t, attackerCtx, "set-admin", "attacker", "true")
	if !errors.Is(err, proto.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}

	reloaded, err := be.User(ctx, "attacker")
	is.NoErr(err)
	is.Equal(reloaded.IsAdmin(), false)
}

// TestUserSubcommandsAreGatedByParent verifies the gate lives on the `user`
// parent command, so subcommands cannot be left unprotected by omission.
func TestUserSubcommandsAreGatedByParent(t *testing.T) {
	is := is.New(t)
	c := UserCommand()
	is.True(c.PersistentPreRunE != nil)

	for _, sub := range c.Commands() {
		if sub.PersistentPreRunE != nil {
			t.Errorf("subcommand %q sets its own PersistentPreRunE; the parent gate is authoritative", sub.Name())
		}
	}
}

// TestSettingsSubcommandsAreGatedByParent is the settings counterpart.
func TestSettingsSubcommandsAreGatedByParent(t *testing.T) {
	is := is.New(t)
	c := SettingsCommand()
	is.True(c.PersistentPreRunE != nil)

	for _, sub := range c.Commands() {
		if sub.PersistentPreRunE != nil {
			t.Errorf("subcommand %q sets its own PersistentPreRunE; the parent gate is authoritative", sub.Name())
		}
	}
}

// TestCollabAddCannotExceedCallerAccess verifies a read-write collaborator
// cannot grant admin-access, which would escalate beyond their own level.
func TestCollabAddCannotExceedCallerAccess(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)

	ownerCtx := withUser(t, ctx, be, "owner", false)
	owner := proto.UserFromContext(ownerCtx)
	_, err := be.CreateRepository(ownerCtx, "repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	collabCtx := withUser(t, ctx, be, "collab", false)
	_ = withUser(t, ctx, be, "puppet", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "collab", access.ReadWriteAccess))
	is.Equal(be.AccessLevelForUser(collabCtx, "repo", proto.UserFromContext(collabCtx)), access.ReadWriteAccess)

	// Granting above the caller's own level must be refused.
	err = runRepo(t, collabCtx, "collab", "add", "repo", "puppet", "admin-access")
	if !errors.Is(err, proto.ErrExceedsAccessLevel) {
		t.Fatalf("expected ErrExceedsAccessLevel, got %v", err)
	}

	_, isCollab, _ := be.IsCollaborator(ctx, "repo", "puppet")
	is.Equal(isCollab, false)

	// Granting at or below the caller's own level is still allowed.
	is.NoErr(runRepo(t, collabCtx, "collab", "add", "repo", "puppet", "read-only"))
	level, isCollab, err := be.IsCollaborator(ctx, "repo", "puppet")
	is.NoErr(err)
	is.True(isCollab)
	is.Equal(level, access.ReadOnlyAccess)
}

// TestCollabRemoveCannotDemoteHigherAccess verifies a read-write collaborator
// cannot remove an admin-access collaborator. Without this, removal plus
// re-adding at a lower level is a demotion primitive that sidesteps the cap
// on granting.
func TestCollabRemoveCannotDemoteHigherAccess(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)

	ownerCtx := withUser(t, ctx, be, "owner", false)
	owner := proto.UserFromContext(ownerCtx)
	_, err := be.CreateRepository(ownerCtx, "repo", owner, proto.RepositoryOptions{})
	is.NoErr(err)

	collabCtx := withUser(t, ctx, be, "collab", false)
	_ = withUser(t, ctx, be, "boss", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "collab", access.ReadWriteAccess))
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "boss", access.AdminAccess))

	err = runRepo(t, collabCtx, "collab", "remove", "repo", "boss")
	if !errors.Is(err, proto.ErrExceedsAccessLevel) {
		t.Fatalf("expected ErrExceedsAccessLevel, got %v", err)
	}

	level, isCollab, err := be.IsCollaborator(ctx, "repo", "boss")
	is.NoErr(err)
	is.True(isCollab)
	is.Equal(level, access.AdminAccess)

	// Overwriting a higher-level collaborator via `add` is refused too.
	err = runRepo(t, collabCtx, "collab", "add", "repo", "boss", "read-only")
	if !errors.Is(err, proto.ErrExceedsAccessLevel) {
		t.Fatalf("expected ErrExceedsAccessLevel on add-overwrite, got %v", err)
	}

	// Removing a peer at or below the caller's level still works.
	_ = withUser(t, ctx, be, "peer", false)
	is.NoErr(be.AddCollaborator(ownerCtx, "repo", "peer", access.ReadOnlyAccess))
	is.NoErr(runRepo(t, collabCtx, "collab", "remove", "repo", "peer"))
}
