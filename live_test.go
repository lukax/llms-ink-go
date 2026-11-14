//go:build live

package llmsink

import (
	"context"
	"errors"
	"testing"
)

// go test -tags live -run Live -v ./...
func TestLiveLlms(t *testing.T) {
	r, err := New(WithKey("")).Llms(context.Background(), "llmstxt.org", LlmsOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %s tokens=%d cached=%v", r.URL, r.Title, r.Tokens, r.Cached)
	c, err := New(WithKey("")).Check(context.Background(), "llmstxt.org")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("check found=%v errors=%d", c.Found, c.Summary.Errors)
}

func TestLiveCheckText(t *testing.T) {
	r, err := New(WithKey("")).CheckText(context.Background(), "# Ex\n\n> S\n\n## Docs\n\n- [A](/a.md): n\n", CheckTextOptions{Base: "https://example.com"})
	if notDeployed(err) {
		t.Skipf("POST /api/v0/check not deployed: %v", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	if r.Parsed == nil || len(r.Parsed.Sections) != 1 {
		t.Fatalf("parsed = %+v", r.Parsed)
	}
	t.Logf("url=%s link=%s", *r.URL, r.Parsed.Sections[0].Links[0].URL)
}

// notDeployed reports a 404/405 from an API that predates POST /api/v0/check.
func notDeployed(err error) bool {
	var e *Error
	return errors.As(err, &e) && (e.Status == 404 || e.Status == 405)
}

func TestLiveInvalidKey(t *testing.T) {
	_, err := New(WithKey("lk_00000000000000000000000000000000")).Llms(context.Background(), "llmstxt.org", LlmsOptions{})
	if !IsInvalidKey(err) {
		t.Fatalf("want invalid_key, got %v", err)
	}
	t.Logf("%v", err)
}
