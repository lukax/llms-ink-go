//go:build live

package llmsink

import (
	"context"
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

func TestLiveInvalidKey(t *testing.T) {
	_, err := New(WithKey("lk_00000000000000000000000000000000")).Llms(context.Background(), "llmstxt.org", LlmsOptions{})
	if !IsInvalidKey(err) {
		t.Fatalf("want invalid_key, got %v", err)
	}
	t.Logf("%v", err)
}
