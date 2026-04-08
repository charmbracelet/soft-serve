package database

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type webhookStore struct{}

var _ store.WebhookStore = (*webhookStore)(nil)

// CreateWebhook implements store.WebhookStore.
func (*webhookStore) CreateWebhook(ctx context.Context, h db.Handler, repoID int64, url string, secret string, contentType int, active bool) (int64, error) {
	var id int64
	query := h.Rebind(`INSERT INTO webhooks (repo_id, url, secret, content_type, active, updated_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) RETURNING id;`)
	err := h.GetContext(ctx, &id, query, repoID, url, secret, contentType, active)
	if err != nil {
		return 0, err
	}

	return id, nil
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
	if len(events) == 0 {
		return nil
	}
	// Bulk INSERT to avoid N round-trips for webhooks with many events.
	// Use strings.Builder to build the placeholder list without intermediate
	// string allocations from strings.Join over a []string.
	var pb strings.Builder
	args := make([]interface{}, 0, len(events)*2)
	for i, event := range events {
		if i > 0 {
			pb.WriteString(", ")
		}
		pb.WriteString("(?, ?)")
		args = append(args, webhookID, event)
	}
	query := h.Rebind(`INSERT INTO webhook_events (webhook_id, event) VALUES ` + pb.String())
	_, err := h.ExecContext(ctx, query, args...)
	return err
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

// DeleteWebhookEventsByID implements store.WebhookStore.
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
// Returns the most recent 100 deliveries to prevent unbounded memory use.
// Callers that need older deliveries should use a paginated query.
func (*webhookStore) GetWebhookDeliveriesByWebhookID(ctx context.Context, h db.Handler, webhookID int64) ([]models.WebhookDelivery, error) {
	query := h.Rebind(`SELECT * FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT 100;`)
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

// sqliteMaxPlaceholders is the maximum number of placeholders SQLite allows in
// a single query (SQLITE_MAX_VARIABLE_NUMBER, default 999). We batch queries
// larger than this to stay within the limit on both SQLite and PostgreSQL.
const sqliteMaxPlaceholders = 999

// GetWebhookEventsByWebhookIDs implements store.WebhookStore.
func (*webhookStore) GetWebhookEventsByWebhookIDs(ctx context.Context, h db.Handler, webhookIDs []int64) ([]models.WebhookEvent, error) {
	if len(webhookIDs) == 0 {
		return nil, nil
	}
	var all []models.WebhookEvent
	for i := 0; i < len(webhookIDs); i += sqliteMaxPlaceholders {
		end := i + sqliteMaxPlaceholders
		if end > len(webhookIDs) {
			end = len(webhookIDs)
		}
		batch := webhookIDs[i:end]
		query, args, err := sqlx.In(`SELECT * FROM webhook_events WHERE webhook_id IN (?);`, batch)
		if err != nil {
			return nil, err
		}
		query = h.Rebind(query)
		var whes []models.WebhookEvent
		if err := h.SelectContext(ctx, &whes, query, args...); err != nil {
			return nil, err
		}
		all = append(all, whes...)
	}
	return all, nil
}

// maxWebhooksPerRepo caps the number of webhooks returned per repository to
// prevent unbounded memory use for repos with many configured webhooks.
const maxWebhooksPerRepo = 100

// GetWebhooksByRepoID implements store.WebhookStore.
func (*webhookStore) GetWebhooksByRepoID(ctx context.Context, h db.Handler, repoID int64) ([]models.Webhook, error) {
	query := h.Rebind(`SELECT * FROM webhooks WHERE repo_id = ? LIMIT ?;`)
	var whs []models.Webhook
	err := h.SelectContext(ctx, &whs, query, repoID, maxWebhooksPerRepo)
	return whs, err
}

// GetWebhooksByRepoIDWhereEvent implements store.WebhookStore.
func (*webhookStore) GetWebhooksByRepoIDWhereEvent(ctx context.Context, h db.Handler, repoID int64, events []int) ([]models.Webhook, error) {
	query, args, err := sqlx.In(`SELECT DISTINCT webhooks.*
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

// maxListWebhookDeliveries caps the number of rows returned by
// ListWebhookDeliveriesByWebhookID to prevent unbounded memory use for
// high-volume webhooks. Callers that need more deliveries should paginate.
const maxListWebhookDeliveries = 100

// ListWebhookDeliveriesByWebhookID implements store.WebhookStore.
func (*webhookStore) ListWebhookDeliveriesByWebhookID(ctx context.Context, h db.Handler, webhookID int64) ([]models.WebhookDelivery, error) {
	query := h.Rebind(`SELECT id, response_status, event FROM webhook_deliveries WHERE webhook_id = ? ORDER BY created_at DESC LIMIT ?;`)
	var whds []models.WebhookDelivery
	err := h.SelectContext(ctx, &whds, query, webhookID, maxListWebhookDeliveries)
	return whds, err
}

// UpdateWebhookByID implements store.WebhookStore.
func (*webhookStore) UpdateWebhookByID(ctx context.Context, h db.Handler, repoID int64, id int64, url string, secret string, contentType int, active bool) error {
	query := h.Rebind(`UPDATE webhooks SET url = ?, secret = ?, content_type = ?, active = ?, updated_at = CURRENT_TIMESTAMP WHERE repo_id = ? AND id = ?;`)
	_, err := h.ExecContext(ctx, query, url, secret, contentType, active, repoID, id)
	return err
}
