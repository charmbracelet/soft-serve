package backend

import (
	"context"
	"encoding/json"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
	"github.com/google/uuid"
)

// CreateWebhook creates a webhook for a repository.
func (b *Backend) CreateWebhook(ctx context.Context, repo proto.Repository, url string, contentType webhook.ContentType, secret string, events []webhook.Event, active bool) error {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	url = utils.Sanitize(url)

	if secret == "" {
		logger := log.FromContext(ctx)
		logger.Warn("webhook created without a secret — deliveries will not be signed and cannot be verified by the receiver", "url", url)
	}

	// Validate webhook URL to prevent SSRF attacks
	if err := webhook.ValidateWebhookURL(ctx, url); err != nil {
		return err //nolint:wrapcheck
	}

	return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		lastID, err := datastore.CreateWebhook(ctx, tx, repo.ID(), url, secret, int(contentType), active)
		if err != nil {
			return db.WrapError(err)
		}

		evs := make([]int, len(events))
		for i, e := range events {
			evs[i] = int(e)
		}
		if err := datastore.CreateWebhookEvents(ctx, tx, lastID, evs); err != nil {
			return db.WrapError(err)
		}

		return nil
	})
}

// Webhook returns a webhook for a repository.
func (b *Backend) Webhook(ctx context.Context, repo proto.Repository, id int64) (webhook.Hook, error) {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	var wh webhook.Hook
	if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		h, err := datastore.GetWebhookByID(ctx, tx, repo.ID(), id)
		if err != nil {
			return db.WrapError(err)
		}
		events, err := datastore.GetWebhookEventsByWebhookID(ctx, tx, id)
		if err != nil {
			return db.WrapError(err)
		}

		wh = webhook.Hook{
			Webhook:     h,
			ContentType: webhook.ContentType(h.ContentType), //nolint:gosec
			Events:      make([]webhook.Event, len(events)),
		}
		for i, e := range events {
			wh.Events[i] = webhook.Event(e.Event)
		}

		return nil
	}); err != nil {
		return webhook.Hook{}, db.WrapError(err)
	}

	return wh, nil
}

// ListWebhooks lists webhooks for a repository.
func (b *Backend) ListWebhooks(ctx context.Context, repo proto.Repository) ([]webhook.Hook, error) {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	var webhooks []models.Webhook
	webhookEvents := map[int64][]models.WebhookEvent{}
	if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		webhooks, err = datastore.GetWebhooksByRepoID(ctx, tx, repo.ID())
		if err != nil {
			return err
		}

		if len(webhooks) == 0 {
			return nil
		}

		ids := make([]int64, len(webhooks))
		for i, h := range webhooks {
			ids[i] = h.ID
		}
		allEvents, err := datastore.GetWebhookEventsByWebhookIDs(ctx, tx, ids)
		if err != nil {
			return err
		}
		for _, e := range allEvents {
			webhookEvents[e.WebhookID] = append(webhookEvents[e.WebhookID], e)
		}

		return nil
	}); err != nil {
		return nil, db.WrapError(err)
	}

	hooks := make([]webhook.Hook, len(webhooks))
	for i, h := range webhooks {
		events := make([]webhook.Event, len(webhookEvents[h.ID]))
		for i, e := range webhookEvents[h.ID] {
			events[i] = webhook.Event(e.Event)
		}

		hooks[i] = webhook.Hook{
			Webhook:     h,
			ContentType: webhook.ContentType(h.ContentType), //nolint:gosec
			Events:      events,
		}
	}

	return hooks, nil
}

// UpdateWebhook updates a webhook.
func (b *Backend) UpdateWebhook(ctx context.Context, repo proto.Repository, id int64, url string, contentType webhook.ContentType, secret string, updatedEvents []webhook.Event, active bool) error {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	// Sanitize and validate webhook URL — mirrors CreateWebhook which also
	// calls utils.Sanitize before ValidateWebhookURL.
	url = utils.Sanitize(url)
	if err := webhook.ValidateWebhookURL(ctx, url); err != nil {
		return err
	}

	return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		if err := datastore.UpdateWebhookByID(ctx, tx, repo.ID(), id, url, secret, int(contentType), active); err != nil {
			return db.WrapError(err)
		}

		currentEvents, err := datastore.GetWebhookEventsByWebhookID(ctx, tx, id)
		if err != nil {
			return db.WrapError(err)
		}

		// Build a set of current events for O(1) lookups.
		currentSet := make(map[int]int64, len(currentEvents))
		for _, e := range currentEvents {
			currentSet[e.Event] = e.ID
		}

		// Delete events that are no longer in the list.
		updatedSet := make(map[webhook.Event]bool, len(updatedEvents))
		for _, e := range updatedEvents {
			updatedSet[e] = true
		}
		var toBeDeleted []int64
		for _, e := range currentEvents {
			if !updatedSet[webhook.Event(e.Event)] {
				toBeDeleted = append(toBeDeleted, e.ID)
			}
		}

		// Guard the delete call: sqlx.In returns an error for empty slices.
		if len(toBeDeleted) > 0 {
			if err := datastore.DeleteWebhookEventsByID(ctx, tx, toBeDeleted); err != nil {
				return db.WrapError(err)
			}
		}

		// Prune events that are already in the list.
		var newEvents []int
		for _, e := range updatedEvents {
			if _, exists := currentSet[int(e)]; !exists {
				newEvents = append(newEvents, int(e))
			}
		}

		if len(newEvents) > 0 {
			if err := datastore.CreateWebhookEvents(ctx, tx, id, newEvents); err != nil {
				return db.WrapError(err)
			}
		}

		return nil
	})
}

// DeleteWebhook deletes a webhook for a repository.
func (b *Backend) DeleteWebhook(ctx context.Context, repo proto.Repository, id int64) error {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	return dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		_, err := datastore.GetWebhookByID(ctx, tx, repo.ID(), id)
		if err != nil {
			return db.WrapError(err)
		}
		if err := datastore.DeleteWebhookForRepoByID(ctx, tx, repo.ID(), id); err != nil {
			return db.WrapError(err)
		}

		return nil
	})
}

// ListWebhookDeliveries lists webhook deliveries for a webhook.
// It verifies that the webhook belongs to the given repository before returning
// results to prevent cross-repo information leakage.
func (b *Backend) ListWebhookDeliveries(ctx context.Context, repo proto.Repository, id int64) ([]webhook.Delivery, error) {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	var deliveries []models.WebhookDelivery
	if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		// Verify webhook belongs to this repository before listing deliveries.
		if _, err := datastore.GetWebhookByID(ctx, tx, repo.ID(), id); err != nil {
			return db.WrapError(err)
		}

		var lerr error
		deliveries, lerr = datastore.ListWebhookDeliveriesByWebhookID(ctx, tx, id)
		if lerr != nil {
			return db.WrapError(lerr)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	ds := make([]webhook.Delivery, len(deliveries))
	for i, d := range deliveries {
		ds[i] = webhook.Delivery{
			WebhookDelivery: d,
			Event:           webhook.Event(d.Event),
		}
	}

	return ds, nil
}

// RedeliverWebhookDelivery redelivers a webhook delivery.
func (b *Backend) RedeliverWebhookDelivery(ctx context.Context, repo proto.Repository, id int64, delID uuid.UUID) error {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	var delivery models.WebhookDelivery
	var wh models.Webhook
	if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		wh, err = datastore.GetWebhookByID(ctx, tx, repo.ID(), id)
		if err != nil {
			b.logger.Errorf("error getting webhook: %v", err)
			return db.WrapError(err)
		}

		delivery, err = datastore.GetWebhookDeliveryByID(ctx, tx, id, delID)
		if err != nil {
			return db.WrapError(err)
		}

		return nil
	}); err != nil {
		return db.WrapError(err)
	}

	b.logger.Infof("redelivering webhook delivery %s for webhook %d", delID, id)

	var payload json.RawMessage
	if err := json.Unmarshal([]byte(delivery.RequestBody), &payload); err != nil {
		b.logger.Errorf("error unmarshaling webhook payload: %v", err)
		return err
	}

	return webhook.SendWebhook(ctx, wh, webhook.Event(delivery.Event), payload)
}

// WebhookDelivery returns a webhook delivery.
// It verifies that the webhook belongs to the given repository before
// returning the delivery to prevent cross-repo information leakage.
func (b *Backend) WebhookDelivery(ctx context.Context, repo proto.Repository, webhookID int64, id uuid.UUID) (webhook.Delivery, error) {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	var delivery webhook.Delivery
	if err := dbx.TransactionContext(ctx, func(tx *db.Tx) error {
		// Verify webhook belongs to this repository before fetching delivery.
		if _, err := datastore.GetWebhookByID(ctx, tx, repo.ID(), webhookID); err != nil {
			return db.WrapError(err)
		}

		d, err := datastore.GetWebhookDeliveryByID(ctx, tx, webhookID, id)
		if err != nil {
			return db.WrapError(err)
		}

		delivery = webhook.Delivery{
			WebhookDelivery: d,
			Event:           webhook.Event(d.Event),
		}

		return nil
	}); err != nil {
		return webhook.Delivery{}, db.WrapError(err)
	}

	return delivery, nil
}
