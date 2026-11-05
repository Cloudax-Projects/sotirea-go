// Package sotirea is the Go client for the Sotirea API: see, and move, the money your
// organisation holds in Sotirea from your own software.
//
//	client := sotirea.New("https://api.example", os.Getenv("SOTIREA_API_KEY"))
//	me, err := client.GetMe(ctx)
//
// Every method answers with the API's data or an error; a refusal from the API is an *Error,
// which says what refused and why. api_gen.go is written from the API's own types and is never
// edited by hand; this file and webhook.go are.
package sotirea

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is this client's version, sent in its User-Agent.
const Version = "1.1.0"

// The codes a refusal can carry, in Error.Code.
const (
	CodeInvalid         = "invalid"
	CodeUnauthenticated = "unauthenticated"
	CodeForbidden       = "forbidden"
	CodeNotFound        = "not_found"
	CodeConflict        = "conflict"
	CodeRateLimited     = "rate_limited"
	CodeUnavailable     = "unavailable"
	CodeInternal        = "internal"
)

// Client calls the Sotirea API with one API key. It is safe for concurrent use.
type Client struct {
	base      string
	key       string
	http      *http.Client
	userAgent string
}

// Option changes how a Client calls the API.
type Option func(*Client)

// WithHTTPClient uses hc for every request, for its timeouts, proxy or transport.
func WithHTTPClient(hc *http.Client) Option { return func(c *Client) { c.http = hc } }

// WithUserAgent puts your application's name before the client's own in the User-Agent.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua + " " + c.userAgent }
}

// New is a client for the API at baseURL, such as Sotirea's API address, with an API key
// (sot_test_... or sot_live_...). Keep the key on your server.
func New(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		base:      strings.TrimRight(baseURL, "/"),
		key:       apiKey,
		http:      &http.Client{Timeout: 30 * time.Second},
		userAgent: "sotirea-go/" + Version,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Error is the API refusing a request, or failing to answer one.
type Error struct {
	// Status is the HTTP status: 400 for invalid, 401, 403, 404, 409, 429, 503 or 500.
	Status int
	// Code is one of the Code constants.
	Code string
	// Message is written for a person and may be shown to one.
	Message string
	// Fields names what refused the request in "rule" (a permission, or a payment rule), or for
	// an invalid request, each field that was wrong.
	Fields map[string]string
	// RequestID is what to quote when asking Sotirea about the request.
	RequestID string
	// RetryAfter is how long to wait before trying again, when the API said.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("sotirea: %s (%d %s, request %s)", e.Message, e.Status, e.Code, e.RequestID)
	}
	return fmt.Sprintf("sotirea: %s (%d %s)", e.Message, e.Status, e.Code)
}

// Rule is what refused the request, such as the permission the key lacks or the payment rule a
// payment broke; empty when none was named.
func (e *Error) Rule() string { return e.Fields["rule"] }

// IsCode says whether err is a refusal with this code, however it was wrapped.
func IsCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

// Page is one page of a list, with where it stands among the rest.
type Page[T any] struct {
	Items      []T
	Page       int
	PageSize   int
	Total      int64
	TotalPages int
	HasNext    bool
	RequestID  string
}

// Meta is what an answer says about itself, beside a single object's data: the request id to quote
// when asking Sotirea about the request. A Page carries its own.
type Meta struct {
	RequestID string
}

func metaOf(m *meta) Meta {
	if m == nil {
		return Meta{}
	}
	return Meta{RequestID: m.RequestID}
}

type meta struct {
	RequestID string `json:"request_id"`
	Page      *struct {
		Page       int   `json:"page"`
		PageSize   int   `json:"page_size"`
		Total      int64 `json:"total"`
		TotalPages int   `json:"total_pages"`
		HasNext    bool  `json:"has_next"`
	} `json:"page"`
}

func (p *Page[T]) fill(m *meta) {
	if p.Items == nil {
		p.Items = []T{}
	}
	if m == nil {
		return
	}
	p.RequestID = m.RequestID
	if m.Page != nil {
		p.Page, p.PageSize, p.Total, p.TotalPages, p.HasNext =
			m.Page.Page, m.Page.PageSize, m.Page.Total, m.Page.TotalPages, m.Page.HasNext
	}
}

// maxAnswer bounds what one answer may make the client hold in memory.
const maxAnswer = 8 << 20

// maxFile bounds a file the client reads into memory, such as a year's statement.
const maxFile = 64 << 20

// send sends one request with the key, answering the response for the caller to read and close.
func (c *Client) send(ctx context.Context, method, path string, query url.Values, header http.Header, body any,
	accept string) (*http.Response, error) {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("sotirea: %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, fmt.Errorf("sotirea: %s %s: %w", method, path, err)
	}
	for k, v := range header {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sotirea: %s %s: %w", method, path, err)
	}
	return resp, nil
}

// envelope is every JSON answer's shape.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Meta    *meta           `json:"meta"`
	Error   *struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

// refused is a refusal as an *Error, from its envelope and its headers.
func refused(resp *http.Response, env envelope) *Error {
	e := &Error{Status: resp.StatusCode, Code: CodeInternal, Message: resp.Status, RequestID: resp.Header.Get("X-Request-Id")}
	if env.Error != nil {
		e.Code, e.Message, e.Fields = env.Error.Code, env.Error.Message, env.Error.Fields
	}
	if env.Meta != nil && env.Meta.RequestID != "" {
		e.RequestID = env.Meta.RequestID
	}
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		e.RetryAfter = time.Duration(s) * time.Second
	}
	return e
}

// do sends one request and decodes the answer's data into out.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, header http.Header, body, out any) (*meta, error) {
	resp, err := c.send(ctx, method, path, query, header, body, "application/json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var env envelope
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxAnswer)).Decode(&env); err != nil {
		return nil, &Error{Status: resp.StatusCode, Code: CodeInternal, RequestID: resp.Header.Get("X-Request-Id"),
			Message: fmt.Sprintf("the API answered %s with a body that is not its JSON", resp.Status)}
	}
	if !env.Success || resp.StatusCode >= 300 {
		return env.Meta, refused(resp, env)
	}
	if env.Meta == nil {
		env.Meta = &meta{}
	}
	if env.Meta.RequestID == "" {
		env.Meta.RequestID = resp.Header.Get("X-Request-Id")
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return env.Meta, fmt.Errorf("sotirea: %s %s: reading the answer: %w", method, path, err)
		}
	}
	return env.Meta, nil
}

// raw sends one request for a file, such as a statement as CSV, and answers its bytes. A refusal
// comes back as JSON and is answered as an *Error.
func (c *Client) raw(ctx context.Context, method, path string, query url.Values, header http.Header) ([]byte, *meta, error) {
	resp, err := c.send(ctx, method, path, query, header, nil, "text/csv, application/json")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	m := &meta{RequestID: resp.Header.Get("X-Request-Id")}
	if resp.StatusCode != http.StatusOK {
		var env envelope
		_ = json.NewDecoder(io.LimitReader(resp.Body, maxAnswer)).Decode(&env)
		return nil, m, refused(resp, env)
	}
	file, err := io.ReadAll(io.LimitReader(resp.Body, maxFile))
	if err != nil {
		return nil, m, fmt.Errorf("sotirea: %s %s: reading the file: %w", method, path, err)
	}
	return file, m, nil
}
