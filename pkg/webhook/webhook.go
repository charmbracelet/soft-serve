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
	"net"
	"net/http"
	"time"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/version"
	"github.com/google/go-querystring/query"
	"github.com/google/uuid"
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

// secureHTTPClient creates an HTTP client with SSRF protection.
var secureHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Parse the address to get the IP
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err //nolint:wrapcheck
			}

			// Validate the resolved IP before connecting
			ip := net.ParseIP(host)
			if ip != nil {
				if err := ValidateIPBeforeDial(ip); err != nil {
					return nil, fmt.Errorf("blocked connection to private IP: %w", err)
				}
			}

			// Use standard dialer with timeout
			dialer := &net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	// Don't follow redirects to prevent bypassing IP validation
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

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
		buf.WriteString(v.Encode()) // nolint: errcheck
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
		sig.Write([]byte(reqBody)) // nolint: errcheck
		headers.Add("X-SoftServe-Signature", "sha256="+hex.EncodeToString(sig.Sum(nil)))
	}

	res, reqErr := do(ctx, w.URL, http.MethodPost, headers, &buf)
	var reqHeaders string
	for k, v := range headers {
		reqHeaders += k + ": " + v[0] + "\n"
	}

	resStatus := 0
	resHeaders := ""
	resBody := ""

	if res != nil {
		resStatus = res.StatusCode
		for k, v := range res.Header {
			resHeaders += k + ": " + v[0] + "\n"
		}

		if res.Body != nil {
			defer res.Body.Close() // nolint: errcheck
			b, err := io.ReadAll(res.Body)
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

	for _, w := range webhooks {
		if err := SendWebhook(ctx, w, payload.Event(), payload); err != nil {
			return err
		}
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
