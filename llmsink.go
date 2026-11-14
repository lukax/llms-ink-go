// Package llmsink is a client for the llms.ink API (https://llms.ink/docs/):
// turn any public link — a site, docs page, GitHub repo, PDF or video — into
// llms.txt or llms-full.txt, and check a site's own llms.txt.
//
// An API key is optional. Without one, requests are anonymous and share the
// per-IP limits; set LLMS_INK_KEY or use WithKey for your own daily quota.
package llmsink

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Version is this module's version, sent in the User-Agent.
const Version = "0.1.0"

// DefaultBaseURL is the production API origin.
const DefaultBaseURL = "https://llms.ink"

// Client calls the llms.ink API. The zero value is not usable; call New.
type Client struct {
	key     string
	baseURL string
	http    *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithKey sets the API key (sent as "Authorization: Bearer <key>"). An empty
// key means anonymous. Overrides LLMS_INK_KEY.
func WithKey(key string) Option { return func(c *Client) { c.key = key } }

// WithBaseURL sets the API origin, e.g. for tests. Defaults to DefaultBaseURL.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient sets the HTTP client. Defaults to one with a 120s timeout.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// New returns a Client. The key defaults to the LLMS_INK_KEY environment variable.
func New(opts ...Option) *Client {
	c := &Client{
		key:     strings.TrimSpace(os.Getenv("LLMS_INK_KEY")),
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Mode selects the output: ModeLLMS (the llms.txt index) or ModeFull (llms-full.txt).
type Mode string

const (
	ModeLLMS Mode = "llms"
	ModeFull Mode = "full"
)

// LlmsOptions are optional parameters for Client.Llms.
type LlmsOptions struct {
	// Mode defaults to ModeLLMS.
	Mode Mode
	// Fresh skips the cache and rebuilds (counts against the daily quota).
	Fresh bool
}

// Segment is a transcript segment, in source-video seconds.
type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// Transcript is a video transcript with timestamps.
type Transcript struct {
	Source   string    `json:"source"`
	Segments []Segment `json:"segments"`
}

// Media describes a video result.
type Media struct {
	Duration   *float64    `json:"duration"` // seconds; nil if unknown
	HasVideo   bool        `json:"hasVideo"`
	HasAudio   bool        `json:"hasAudio"`
	Thumbnail  *string     `json:"thumbnail"`
	Downloads  []string    `json:"downloads"` // "mp4", "m4a"
	IsLive     bool        `json:"isLive,omitempty"`
	Transcript *Transcript `json:"transcript,omitempty"`
}

// Result is the response of GET /api/v0/llms.
type Result struct {
	URL         string  `json:"url"`         // the link after normalization
	Kind        string  `json:"kind"`        // site, section, page, repo, document, video
	Source      string  `json:"source"`      // native or generated
	Mode        Mode    `json:"mode"`        // llms or full
	Title       string  `json:"title"`       //
	Text        string  `json:"text"`        // the llms.txt / llms-full.txt Markdown
	Tokens      int     `json:"tokens"`      // approximate (characters ÷ 4)
	Pages       int     `json:"pages"`       // links (llms) or pages inlined (full)
	Model       *string `json:"model"`       // nil for native files
	Cached      bool    `json:"cached"`      //
	GeneratedAt string  `json:"generatedAt"` // RFC 3339
	Media       *Media  `json:"media,omitempty"`
}

// CheckIssue is one finding of Client.Check.
type CheckIssue struct {
	Level   string `json:"level"` // error, warning, info
	Code    string `json:"code"`  // e.g. missing_title, broken_link
	Line    *int   `json:"line"`
	Message string `json:"message"`
}

// BrokenLink is a link in llms.txt that failed; Status is 0 on timeout.
type BrokenLink struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
}

// CheckResult is the response of GET /api/v0/check.
type CheckResult struct {
	Site     string  `json:"site"`
	URL      string  `json:"url"`
	Found    bool    `json:"found"`
	Status   int     `json:"status"`
	Title    *string `json:"title"`
	Tokens   int     `json:"tokens"`
	Sections int     `json:"sections"`
	Links    struct {
		Total   int          `json:"total"`
		Checked int          `json:"checked"`
		Broken  []BrokenLink `json:"broken"`
	} `json:"links"`
	LlmsFull bool         `json:"llmsFull"`
	Issues   []CheckIssue `json:"issues"`
	Summary  struct {
		Errors   int `json:"errors"`
		Warnings int `json:"warnings"`
	} `json:"summary"`
	CheckedAt string `json:"checkedAt"`
}

// Error is an API error (RFC 9457 problem details; codes at
// https://llms.ink/docs/#errors). Use errors.As, or IsInvalidKey,
// IsRateLimited and IsQuotaExceeded.
type Error struct {
	Status     int           // HTTP status
	Code       string        // e.g. invalid_key, rate_limited, quota_exceeded, invalid_url
	Title      string        // HTTP status title
	Detail     string        // for people
	RetryAfter time.Duration // from Retry-After on 429; 0 if absent
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("llms.ink: %d %s: %s", e.Status, e.Code, e.Detail)
	}
	return fmt.Sprintf("llms.ink: %d: %s", e.Status, e.Detail)
}

func hasCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

// IsInvalidKey reports whether err is a 401 invalid_key (unknown or revoked key).
func IsInvalidKey(err error) bool { return hasCode(err, "invalid_key") }

// IsRateLimited reports whether err is a 429 rate_limited (burst limit); see Error.RetryAfter.
func IsRateLimited(err error) bool { return hasCode(err, "rate_limited") }

// IsQuotaExceeded reports whether err is a 429 quota_exceeded (daily quota); see Error.RetryAfter.
func IsQuotaExceeded(err error) bool { return hasCode(err, "quota_exceeded") }

// Llms returns llms.txt (or llms-full.txt with ModeFull) for any public link.
func (c *Client) Llms(ctx context.Context, link string, opts LlmsOptions) (*Result, error) {
	q := url.Values{"url": {link}}
	if opts.Mode != "" {
		q.Set("mode", string(opts.Mode))
	}
	if opts.Fresh {
		q.Set("fresh", "1")
	}
	var r Result
	if err := c.get(ctx, "/api/v0/llms", q, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Check validates a site's own llms.txt: format, broken links, llms-full.txt.
func (c *Client) Check(ctx context.Context, link string) (*CheckResult, error) {
	var r CheckResult
	if err := c.get(ctx, "/api/v0/check", url.Values{"url": {link}}, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "llms-ink-go/"+Version)
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("llms.ink: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("llms.ink: reading response: %w", err)
	}
	if res.StatusCode >= 400 {
		return problem(res, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("llms.ink: decoding response: %w", err)
	}
	return nil
}

func problem(res *http.Response, body []byte) *Error {
	var p struct {
		Status int    `json:"status"`
		Code   string `json:"code"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	_ = json.Unmarshal(body, &p)
	e := &Error{Status: res.StatusCode, Code: p.Code, Title: p.Title, Detail: p.Detail}
	if p.Status != 0 {
		e.Status = p.Status
	}
	if e.Title == "" {
		e.Title = http.StatusText(res.StatusCode)
	}
	if e.Detail == "" {
		e.Detail = e.Title
	}
	if s, err := strconv.Atoi(strings.TrimSpace(res.Header.Get("Retry-After"))); err == nil && s > 0 {
		e.RetryAfter = time.Duration(s) * time.Second
	}
	return e
}
