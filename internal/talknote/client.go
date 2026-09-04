// Package talknote は Talknote Web API のコントラクト層。
package talknote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseBytes = 32 << 20

type Client struct {
	origin string
	sid    string
	hc     *http.Client
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(client *Client) { client.hc = hc }
}

func New(origin, sid string, opts ...Option) *Client {
	client := &Client{
		origin: strings.TrimRight(origin, "/"),
		sid:    sid,
		hc: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	for _, option := range opts {
		option(client)
	}
	return client
}

type APIError struct {
	Method  string
	Path    string
	Status  int
	Message string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("talknote web api %s %s: HTTP %d", e.Method, e.Path, e.Status)
	}
	return fmt.Sprintf("talknote web api %s %s: HTTP %d: %s", e.Method, e.Path, e.Status, e.Message)
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	if len(query) != 0 {
		path += "?" + query.Encode()
	}
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.origin+path, requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", "TALKNOTE_SID2="+c.sid)
	if method != http.MethodGet {
		req.Header.Set("Origin", c.origin)
		req.Header.Set("Referer", c.origin+"/")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var response struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(responseBody, &response)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			response.Message = "browser session expired or unauthorized; run `tn auth login` again"
		} else if response.Message == "" && resp.StatusCode >= 300 && resp.StatusCode < 400 {
			response.Message = "unexpected redirect"
		} else if response.Message == "" {
			response.Message = "non-JSON error response"
		}
		return &APIError{Method: method, Path: path, Status: resp.StatusCode, Message: response.Message}
	}
	if out == nil {
		return nil
	}
	if len(bytes.TrimSpace(responseBody)) == 0 {
		return fmt.Errorf("talknote web api %s %s: empty response", method, path)
	}
	if err := json.Unmarshal(responseBody, out); err != nil {
		return fmt.Errorf("talknote web api %s %s: decode response: %w", method, path, err)
	}
	return nil
}
