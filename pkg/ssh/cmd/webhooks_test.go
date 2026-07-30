package cmd

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/google/uuid"
	"github.com/matryer/is"
	_ "modernc.org/sqlite"
)

// TestWebhookDeliveriesAreScopedToRepository verifies that webhook deliveries
// cannot be read across repositories.
//
// The `repo webhook deliveries list|get` commands authorize the caller against
// the REPOSITORY argument, but the delivery lookup used to be keyed only on
// the numeric webhook ID. Since any user who owns a repository has admin
// access to it, a user could pass their own repository name together with
// another repository's webhook ID and read that webhook's stored deliveries.
//
// Deliveries store the full request URL, headers (including the HMAC
// signature), request body, and response body, so this is a cross-repository
// disclosure of private repository event payloads.
func TestWebhookDeliveriesAreScopedToRepository(t *testing.T) {
	is := is.New(t)
	ctx, be := newAuthTestContext(t)

	// The victim owns a private repository with a webhook, and that webhook
	// has a recorded delivery containing sensitive request/response data.
	victimCtx := withUser(t, ctx, be, "victim", false)
	victim := proto.UserFromContext(victimCtx)
	victimRepo, err := be.CreateRepository(victimCtx, "victim-repo", victim, proto.RepositoryOptions{Private: true})
	is.NoErr(err)
	is.NoErr(be.CreateWebhook(victimCtx, victimRepo, "http://example.com/hook", webhook.ContentTypeJSON, "s3cret", []webhook.Event{webhook.EventPush}, true))

	victimHooks, err := be.ListWebhooks(victimCtx, victimRepo)
	is.NoErr(err)
	is.Equal(len(victimHooks), 1)
	victimHookID := victimHooks[0].ID

	const (
		secretBody      = "private-repo-payload"
		secretSignature = "X-SoftServe-Signature: sha256=leaked-signature\n"
	)
	deliveryID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	is.NoErr(store.FromContext(ctx).CreateWebhookDelivery(
		ctx, db.FromContext(ctx), deliveryID, victimHookID, int(webhook.EventPush),
		"http://example.com/hook", "POST", nil,
		secretSignature, secretBody, 200, "Content-Type: text/plain\n", "victim response body",
	))

	// The attacker owns their own repository, so they hold admin access to
	// it, but they have no access at all to the victim's repository.
	attackerCtx := withUser(t, ctx, be, "attacker", false)
	attacker := proto.UserFromContext(attackerCtx)
	_, err = be.CreateRepository(attackerCtx, "attacker-repo", attacker, proto.RepositoryOptions{})
	is.NoErr(err)

	hookID := strconv.FormatInt(victimHookID, 10)

	// Listing deliveries by naming their own repository must not enumerate
	// the victim webhook's delivery IDs.
	stdout, _, err := runRepoOutput(t, attackerCtx, "webhook", "deliveries", "list", "attacker-repo", hookID)
	if err == nil {
		t.Error("expected error listing another repository's webhook deliveries")
	}
	if strings.Contains(stdout, deliveryID.String()) {
		t.Errorf("leaked delivery ID in output: %q", stdout)
	}

	// Reading a specific delivery must not disclose its stored request or
	// response data either.
	stdout, _, err = runRepoOutput(t, attackerCtx, "webhook", "deliveries", "get", "attacker-repo", hookID, deliveryID.String())
	if err == nil {
		t.Error("expected error getting another repository's webhook delivery")
	}
	for _, secret := range []string{secretBody, "leaked-signature", "victim response body"} {
		if strings.Contains(stdout, secret) {
			t.Errorf("leaked %q in output: %q", secret, stdout)
		}
	}

	// The owner can still read their own webhook's deliveries.
	stdout, _, err = runRepoOutput(t, victimCtx, "webhook", "deliveries", "list", "victim-repo", hookID)
	is.NoErr(err)
	if !strings.Contains(stdout, deliveryID.String()) {
		t.Errorf("owner could not list own deliveries, got: %q", stdout)
	}

	stdout, _, err = runRepoOutput(t, victimCtx, "webhook", "deliveries", "get", "victim-repo", hookID, deliveryID.String())
	is.NoErr(err)
	if !strings.Contains(stdout, secretBody) {
		t.Errorf("owner could not read own delivery body, got: %q", stdout)
	}
}
