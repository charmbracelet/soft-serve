package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/ssrf"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/version"
	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
)

const (
	maxRetries     = 3
	retryBaseDelay = time.Second
)

// Hook is a repository webhook.
type Hook struct {
	models.Webhook
	ContentType ContentType
	Events      []Event
}

// Delivery is a webhook delivery.
type Delivery struct {
	models.WebhookDelivery
	Event Event
}

// secureHTTPClient is an HTTP client with SSRF protection.
var secureHTTPClient = ssrf.NewSecureClient()

// do sends a webhook.
// Caller must close the returned body.
func do(ctx context.Context, url string, method string, headers http.Header, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = headers
	res, err := secureHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func isRetryableStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func doWithRetry(ctx context.Context, url string, method string, headers http.Header, body string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		res, err := do(ctx, url, method, headers, strings.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		if !isRetryableStatus(res.StatusCode) {
			return res, nil
		}

		lastErr = fmt.Errorf("server returned %d", res.StatusCode)
		if res.Body != nil {
			res.Body.Close() //nolint: errcheck
		}
	}
	return nil, lastErr
}

// SendWebhook sends a webhook event.
func SendWebhook(ctx context.Context, w models.Webhook, event Event, payload interface{}) error {
	var buf bytes.Buffer
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)

	contentType := ContentType(w.ContentType) //nolint:gosec
	switch contentType {
	case ContentTypeJSON:
		if err := json.NewEncoder(&buf).Encode(payload); err != nil {
			return err
		}
	case ContentTypeForm:
		v, err := query.Values(payload)
		if err != nil {
			return err
		}
		buf.WriteString(v.Encode()) //nolint: errcheck
	default:
		return ErrInvalidContentType
	}

	headers := http.Header{}
	headers.Add("Content-Type", contentType.String())
	headers.Add("User-Agent", "SoftServe/"+version.Version)
	headers.Add("X-SoftServe-Event", event.String())

	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	headers.Add("X-SoftServe-Delivery", id.String())

	reqBody := buf.String()
	if w.Secret != "" {
		sig := hmac.New(sha256.New, []byte(w.Secret))
		sig.Write([]byte(reqBody)) //nolint: errcheck
		headers.Add("X-SoftServe-Signature", "sha256="+hex.EncodeToString(sig.Sum(nil)))
	}

	res, reqErr := doWithRetry(ctx, w.URL, http.MethodPost, headers, reqBody)
	headerKeys := make([]string, 0, len(headers))
	for k := range headers {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	var reqHeadersB strings.Builder
	for _, k := range headerKeys {
		reqHeadersB.WriteString(k + ": " + strings.Join(headers[k], ", ") + "\n")
	}
	reqHeaders := reqHeadersB.String()

	resStatus := 0
	resHeaders := ""
	resBody := ""

	if res != nil {
		resStatus = res.StatusCode
		var resHeadersB strings.Builder
		for k, v := range res.Header {
			resHeadersB.WriteString(k + ": " + strings.Join(v, ", ") + "\n")
		}
		resHeaders = resHeadersB.String()

		if res.Body != nil {
			defer res.Body.Close()                                //nolint: errcheck
			b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20)) // 1 MiB
			if err != nil {
				return err
			}

			resBody = string(b)
		}
	}

	return db.WrapError(datastore.CreateWebhookDelivery(ctx, dbx, id, w.ID, int(event), w.URL, http.MethodPost, reqErr, reqHeaders, reqBody, resStatus, resHeaders, resBody))
}

// SendEvent sends a webhook event.
func SendEvent(ctx context.Context, payload EventPayload) error {
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	webhooks, err := datastore.GetWebhooksByRepoIDWhereEvent(ctx, dbx, payload.RepositoryID(), []int{int(payload.Event())})
	if err != nil {
		return db.WrapError(err)
	}

	var errs []error
	for _, w := range webhooks {
		if err := SendWebhook(ctx, w, payload.Event(), payload); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func repoURL(publicURL string, repo string) string {
	return fmt.Sprintf("%s/%s.git", publicURL, utils.SanitizeRepo(repo))
}

func getDefaultBranch(repo proto.Repository) (string, error) {
	branch, err := proto.RepositoryDefaultBranch(repo)
	// XXX: we check for ErrReferenceNotExist here because we don't want to
	// return an error if the repo is an empty repo.
	// This means that the repo doesn't have a default branch yet and this is
	// the first push to it.
	if err != nil && !errors.Is(err, git.ErrReferenceNotExist) {
		return "", err
	}

	return branch, nil
}
