package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type webhookStore struct{}

var _ store.WebhookStore = (*webhookStore)(nil)

// CreateWebhook implements store.WebhookStore.
func (*webhookStore) CreateWebhook(ctx context.Context, h db.Handler, repoID int64, url string, secret string, contentType int, active bool) (int64, error) {
	query := h.Rebind(`INSERT INTO webhooks (repo_id, url, secret, content_type, active, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`)
	result, err := h.ExecContext(ctx, query, repoID, url, secret, contentType, active)
	if err != nil {
		return 0, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastID, nil
}

// CreateWebhookDelivery implements store.WebhookStore.
func (*webhookStore) CreateWebhookDelivery(ctx context.Context, h db.Handler, id uuid.UUID, webhookID int64, event int, url string, method string, requestError error, requestHeaders string, requestBody string, responseStatus int, responseHeaders string, responseBody string) error {
	query := h.Rebind(`INSERT INTO webhook_deliveries (id, webhook_id, event, request_url, request_method, request_error, request_headers, request_body, response_status, response_headers, response_body)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`)
	var reqErr string
	if requestError != nil {
		reqErr = requestError.Error()
	}
	_, err := h.ExecContext(ctx, query, id, webhookID, event, url, method, reqErr, requestHeaders, requestBody, responseStatus, responseHeaders, responseBody)
	return err
}

// CreateWebhookEvents implements store.WebhookStore.
func (*webhookStore) CreateWebhookEvents(ctx context.Context, h db.Handler, webhookID int64, events []int) error {
	query := h.Rebind(`INSERT INTO webhook_events (webhook_id, event)
			VALUES (?, ?);`)
	for _, event := range events {
		_, err := h.ExecContext(ctx, query, webhookID, event)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteWebhookByID implements store.WebhookStore.
func (*webhookStore) DeleteWebhookByID(ctx context.Context, h db.Handler, id int64) error {
	query := h.Rebind(`DELETE FROM webhooks WHERE id = ?;`)
	_, err := h.ExecContext(ctx, query, id)
	return err
}

// DeleteWebhookForRepoByID implements store.WebhookStore.
func (*webhookStore) DeleteWebhookForRepoByID(ctx context.Context, h db.Handler, repoID int64, id int64) error {
	query := h.Rebind(`DELETE FROM webhooks WHERE repo_id = ? AND id = ?;`)
	_, err := h.ExecContext(ctx, query, repoID, id)
	return err
}

// DeleteWebhookDeliveryByID implements store.WebhookStore.
func (*webhookStore) DeleteWebhookDeliveryByID(ctx context.Context, h db.Handler, webhookID int64, id uuid.UUID) error {
	query := h.Rebind(`DELETE FROM webhook_deliveries WHERE webhook_id = ? AND id = ?;`)
	_, err := h.ExecContext(ctx, query, webhookID, id)
	return err
}

// DeleteWebhookEventsByWebhookID implements store.WebhookStore.
func (*webhookStore) DeleteWebhookEventsByID(ctx context.Context, h db.Handler, ids []int64) error {
	query, args, err := sqlx.In(`DELETE FROM webhook_events WHERE id IN (?);`, ids)
	if err != nil {
		return err
	}

	query = h.Rebind(query)
	_, err = h.ExecContext(ctx, query, args...)
	return err
}

// GetWebhookByID implements store.WebhookStore.
func (*webhookStore) GetWebhookByID(ctx context.Context, h db.Handler, repoID int64, id int64) (models.Webhook, error) {
	query := h.Rebind(`SELECT * FROM webhooks WHERE repo_id = ? AND id = ?;`)
	var wh models.Webhook
	err := h.GetContext(ctx, &wh, query, repoID, id)
	return wh, err
}

// GetWebhookDeliveriesByWebhookID implements store.WebhookStore.
func (*webhookStore) GetWebhookDeliveriesByWebhookID(ctx context.Context, h db.Handler, webhookID int64) ([]models.WebhookDelivery, error) {
	query := h.Rebind(`SELECT * FROM webhook_deliveries WHERE webhook_id = ?;`)
	var whds []models.WebhookDelivery
	err := h.SelectContext(ctx, &whds, query, webhookID)
	return whds, err
}

// GetWebhookDeliveryByID implements store.WebhookStore.
func (*webhookStore) GetWebhookDeliveryByID(ctx context.Context, h db.Handler, webhookID int64, id uuid.UUID) (models.WebhookDelivery, error) {
	query := h.Rebind(`SELECT * FROM webhook_deliveries WHERE webhook_id = ? AND id = ?;`)
	var whd models.WebhookDelivery
	err := h.GetContext(ctx, &whd, query, webhookID, id)
	return whd, err
}

// GetWebhookEventByID implements store.WebhookStore.
func (*webhookStore) GetWebhookEventByID(ctx context.Context, h db.Handler, id int64) (models.WebhookEvent, error) {
	query := h.Rebind(`SELECT * FROM webhook_events WHERE id = ?;`)
	var whe models.WebhookEvent
	err := h.GetContext(ctx, &whe, query, id)
	return whe, err
}

// GetWebhookEventsByWebhookID implements store.WebhookStore.
func (*webhookStore) GetWebhookEventsByWebhookID(ctx context.Context, h db.Handler, webhookID int64) ([]models.WebhookEvent, error) {
	query := h.Rebind(`SELECT * FROM webhook_events WHERE webhook_id = ?;`)
	var whes []models.WebhookEvent
	err := h.SelectContext(ctx, &whes, query, webhookID)
	return whes, err
}

// GetWebhooksByRepoID implements store.WebhookStore.
func (*webhookStore) GetWebhooksByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Webhook, error) {
	query := h.Rebind(`SELECT * FROM webhooks WHERE repo_id = ?;`)
	var whs []models.Webhook
	err := h.SelectContext(ctx, &whs, query, repoID)
	return whs, err
}

// GetWebhooksByRepoIDWhereEvent implements store.WebhookStore.
func (*webhookStore) GetWebhooksByRepoIDWhereEvent(ctx context.Context, h db.Handler, repoID int64, events []int) ([]models.Webhook, error) {
	query, args, err := sqlx.In(`SELECT webhooks.*
			FROM webhooks
			INNER JOIN webhook_events ON webhooks.id = webhook_events.webhook_id
			WHERE webhooks.repo_id = ? AND webhook_events.event IN (?);`, repoID, events)
	if err != nil {
		return nil, err
	}

	query = h.Rebind(query)
	var whs []models.Webhook
	err = h.SelectContext(ctx, &whs, query, args...)
	return whs, err
}

// ListWebhookDeliveriesByWebhookID implements store.WebhookStore.
func (*webhookStore) ListWebhookDeliveriesByWebhookID(ctx context.Context, h db.Handler, webhookID int64) ([]models.WebhookDelivery, error) {
	query := h.Rebind(`SELECT id, response_status, event FROM webhook_deliveries WHERE webhook_id = ?;`)
	var whds []models.WebhookDelivery
	err := h.SelectContext(ctx, &whds, query, webhookID)
	return whds, err
}

// UpdateWebhookByID implements store.WebhookStore.
func (*webhookStore) UpdateWebhookByID(ctx context.Context, h db.Handler, repoID int64, id int64, url string, secret string, contentType int, active bool) error {
	query := h.Rebind(`UPDATE webhooks SET url = ?, secret = ?, content_type = ?, active = ?, updated_at = CURRENT_TIMESTAMP WHERE repo_id = ? AND id = ?;`)
	_, err := h.ExecContext(ctx, query, url, secret, contentType, active, repoID, id)
	return err
}
