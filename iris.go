// Package iris sends transactional email through Iris (https://iris.orvlabs.com) and verifies its webhooks.
package iris

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

const DefaultBaseURL = "https://iris.orvlabs.com"

// Client calls the Iris API with one key. Use it from your server only: the key must stay secret.
type Client struct {
	Key        string
	BaseURL    string
	HTTPClient *http.Client
}

// New returns a client for key (iris_… or iris_test_…).
func New(key string) *Client {
	return &Client{Key: key, BaseURL: DefaultBaseURL, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

type Address struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// Attachment content is base64; Iris accepts 10 MB decoded at most in total.
type Attachment struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Payment adds a 221 Pay link (Wave, Orange Money) of your linked merchant account. Amounts are whole CFA francs.
type Payment struct {
	AmountXOF   int64    `json:"amountXof"`
	Reference   string   `json:"reference"`
	Description string   `json:"description"`
	ExpiresIn   int      `json:"expiresIn,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}

type Email struct {
	Sender      Address           `json:"sender"`
	To          []Address         `json:"to,omitempty"`
	Cc          []Address         `json:"cc,omitempty"`
	Bcc         []Address         `json:"bcc,omitempty"`
	ReplyTo     *Address          `json:"replyTo,omitempty"`
	Subject     string            `json:"subject,omitempty"`
	HTMLContent string            `json:"htmlContent,omitempty"`
	TextContent string            `json:"textContent,omitempty"`
	TemplateID  int64             `json:"templateId,omitempty"`
	Params      map[string]any    `json:"params,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Attachment  []Attachment      `json:"attachment,omitempty"`
	ScheduledAt *time.Time        `json:"scheduledAt,omitempty"`
	BatchID     string            `json:"batchId,omitempty"`
	Payment     *Payment          `json:"payment,omitempty"`
	Unsubscribe bool              `json:"unsubscribe,omitempty"`
}

// Sent is the answer to SendEmail. BatchID is set only for a scheduled email.
type Sent struct {
	MessageID  string   `json:"messageId"`
	MessageIDs []string `json:"messageIds"`
	BatchID    string   `json:"batchId"`
}

// Error is an answer other than 2xx. Branch on Status and Code; Message is French text that may change.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("iris: %d %s: %s", e.Status, e.Code, e.Message) }

// Retryable reports a temporary failure (429, 502, 503): retry with an increasing delay.
func (e *Error) Retryable() bool {
	return e.Status == http.StatusTooManyRequests || e.Status == http.StatusBadGateway || e.Status == http.StatusServiceUnavailable
}

// SendEmail queues an email. It has no idempotency key: retrying after a network timeout can send it twice.
func (c *Client) SendEmail(ctx context.Context, email Email) (*Sent, error) {
	var out Sent
	if err := c.do(ctx, http.MethodPost, "/v3/smtp/email", email, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", c.Key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		e := &Error{Status: res.StatusCode}
		if json.Unmarshal(raw, e) != nil || e.Code == "" {
			e.Code, e.Message = "http_error", string(raw)
		}
		return e
	}
	return json.Unmarshal(raw, out)
}

// Event is an email event webhook. Deduplicate on ID: retries carry the same one.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      struct {
		MessageID string `json:"messageId"`
		Email     string `json:"email"`
		URL       string `json:"url"`
	} `json:"data"`
}

// VerifyWebhook checks the Standard Webhooks signature of a delivery with your whsec_ secret, then decodes it.
// Pass the raw body, before any parsing, and the request headers.
func VerifyWebhook(secret string, headers http.Header, body []byte) (*Event, error) {
	wh, err := standardwebhooks.NewWebhook(secret)
	if err != nil {
		return nil, err
	}
	if err := wh.Verify(body, headers); err != nil {
		return nil, err
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
