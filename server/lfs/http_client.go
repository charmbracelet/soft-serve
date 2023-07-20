package lfs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/charmbracelet/log"
)

// httpClient is a Git LFS client to communicate with a LFS source API.
type httpClient struct {
	client    *http.Client
	endpoint  Endpoint
	transfers map[string]TransferAdapter
}

var _ Client = (*httpClient)(nil)

// newHTTPClient returns a new Git LFS client.
func newHTTPClient(endpoint Endpoint) *httpClient {
	return &httpClient{
		client:   http.DefaultClient,
		endpoint: endpoint,
		transfers: map[string]TransferAdapter{
			"basic": &BasicTransferAdapter{http.DefaultClient},
		},
	}
}

// Download implements Client.
func (c *httpClient) Download(ctx context.Context, objects []Pointer, callback DownloadCallback) error {
	return c.performOperation(ctx, objects, callback, nil)
}

// Upload implements Client.
func (c *httpClient) Upload(ctx context.Context, objects []Pointer, callback UploadCallback) error {
	return c.performOperation(ctx, objects, nil, callback)
}

func (c *httpClient) transferNames() []string {
	names := make([]string, len(c.transfers))
	i := 0
	for name := range c.transfers {
		names[i] = name
		i++
	}
	return names
}

// batch performs a batch request to the LFS server.
func (c *httpClient) batch(ctx context.Context, operation string, objects []Pointer) (*BatchResponse, error) {
	logger := log.FromContext(ctx).WithPrefix("lfs")
	url := fmt.Sprintf("%s/objects/batch", c.endpoint.String())

	// TODO: support ref
	request := &BatchRequest{operation, c.transferNames(), nil, objects, hashAlgo}

	payload := new(bytes.Buffer)
	err := json.NewEncoder(payload).Encode(request)
	if err != nil {
		logger.Errorf("Error encoding json: %v", err)
		return nil, err
	}

	logger.Debugf("Calling: %s", url)

	req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
	if err != nil {
		logger.Errorf("Error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("Content-type", MediaType)
	req.Header.Set("Accept", MediaType)

	res, err := c.client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		logger.Errorf("Error while processing request: %v", err)
		return nil, err
	}
	defer res.Body.Close() // nolint: errcheck

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected server response: %s", res.Status)
	}

	var response BatchResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		logger.Errorf("Error decoding json: %v", err)
		return nil, err
	}

	if len(response.Transfer) == 0 {
		response.Transfer = "basic"
	}

	return &response, nil
}

func (c *httpClient) performOperation(ctx context.Context, objects []Pointer, dc DownloadCallback, uc UploadCallback) error {
	logger := log.FromContext(ctx).WithPrefix("lfs")
	if len(objects) == 0 {
		return nil
	}

	operation := "download"
	if uc != nil {
		operation = "upload"
	}

	result, err := c.batch(ctx, operation, objects)
	if err != nil {
		return err
	}

	transferAdapter, ok := c.transfers[result.Transfer]
	if !ok {
		return fmt.Errorf("TransferAdapter not found: %s", result.Transfer)
	}

	for _, object := range result.Objects {
		if object.Error != nil {
			objectError := errors.New(object.Error.Message)
			logger.Debugf("Error on object %v: %v", object.Pointer, objectError)
			if uc != nil {
				if _, err := uc(object.Pointer, objectError); err != nil {
					return err
				}
			} else {
				if err := dc(object.Pointer, nil, objectError); err != nil {
					return err
				}
			}
			continue
		}

		if uc != nil {
			if len(object.Actions) == 0 {
				logger.Debugf("%v already present on server", object.Pointer)
				continue
			}

			link, ok := object.Actions["upload"]
			if !ok {
				logger.Debugf("%+v", object)
				return errors.New("Missing action 'upload'")
			}

			content, err := uc(object.Pointer, nil)
			if err != nil {
				return err
			}

			err = transferAdapter.Upload(ctx, object.Pointer, content, link)

			content.Close() // nolint: errcheck

			if err != nil {
				return err
			}

			link, ok = object.Actions["verify"]
			if ok {
				if err := transferAdapter.Verify(ctx, object.Pointer, link); err != nil {
					return err
				}
			}
		} else {
			link, ok := object.Actions["download"]
			if !ok {
				logger.Debugf("%+v", object)
				return errors.New("Missing action 'download'")
			}

			content, err := transferAdapter.Download(ctx, object.Pointer, link)
			if err != nil {
				return err
			}

			if err := dc(object.Pointer, content, nil); err != nil {
				return err
			}
		}
	}

	return nil
}
