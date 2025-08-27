package lfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/charmbracelet/log/v2"
)

// BasicTransferAdapter implements the "basic" adapter.
type BasicTransferAdapter struct {
	client *http.Client
}

// Name returns the name of the adapter.
func (a *BasicTransferAdapter) Name() string {
	return "basic"
}

// Download reads the download location and downloads the data.
func (a *BasicTransferAdapter) Download(ctx context.Context, _ Pointer, l *Link) (io.ReadCloser, error) {
	resp, err := a.performRequest(ctx, "GET", l, nil, nil)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// Upload sends the content to the LFS server.
func (a *BasicTransferAdapter) Upload(ctx context.Context, p Pointer, r io.Reader, l *Link) error {
	res, err := a.performRequest(ctx, "PUT", l, r, func(req *http.Request) {
		if len(req.Header.Get("Content-Type")) == 0 {
			req.Header.Set("Content-Type", "application/octet-stream")
		}

		if req.Header.Get("Transfer-Encoding") == "chunked" {
			req.TransferEncoding = []string{"chunked"}
		}

		req.ContentLength = p.Size
	})
	if err != nil {
		return err
	}
	return res.Body.Close()
}

// Verify calls the verify handler on the LFS server.
func (a *BasicTransferAdapter) Verify(ctx context.Context, p Pointer, l *Link) error {
	logger := log.FromContext(ctx).WithPrefix("lfs")
	b, err := json.Marshal(p)
	if err != nil {
		logger.Errorf("Error encoding json: %v", err)
		return err
	}

	res, err := a.performRequest(ctx, "POST", l, bytes.NewReader(b), func(req *http.Request) {
		req.Header.Set("Content-Type", MediaType)
	})
	if err != nil {
		return err
	}
	return res.Body.Close()
}

func (a *BasicTransferAdapter) performRequest(ctx context.Context, method string, l *Link, body io.Reader, callback func(*http.Request)) (*http.Response, error) {
	logger := log.FromContext(ctx).WithPrefix("lfs")
	logger.Debugf("Calling: %s %s", method, l.Href)

	req, err := http.NewRequestWithContext(ctx, method, l.Href, body)
	if err != nil {
		logger.Errorf("Error creating request: %v", err)
		return nil, err
	}
	for key, value := range l.Header {
		req.Header.Set(key, value)
	}
	req.Header.Set("Accept", MediaType)

	if callback != nil {
		callback(req)
	}

	res, err := a.client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}
		logger.Errorf("Error while processing request: %v", err)
		return res, err
	}

	if res.StatusCode != http.StatusOK {
		return res, handleErrorResponse(res)
	}

	return res, nil
}

func handleErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()

	er, err := decodeResponseError(resp.Body)
	if err != nil {
		return fmt.Errorf("Request failed with status %s", resp.Status)
	}
	return errors.New(er.Message)
}

func decodeResponseError(r io.Reader) (ErrorResponse, error) {
	var er ErrorResponse
	err := json.NewDecoder(r).Decode(&er)
	if err != nil {
		log.Error("Error decoding json: %v", err)
	}
	return er, err
}
