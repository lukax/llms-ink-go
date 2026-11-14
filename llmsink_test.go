package llmsink

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func stub(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(h)
	t.Cleanup(s.Close)
	return New(WithBaseURL(s.URL), WithKey(""))
}

func TestLlms(t *testing.T) {
	var got *http.Request
	c := stub(t, func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"url":"https://example.com/","kind":"site","source":"generated","mode":"full","title":"Ex","text":"# Ex","tokens":2,"pages":1,"model":"m","cached":true,"generatedAt":"2026-01-01T00:00:00Z","media":{"duration":null,"hasVideo":true,"hasAudio":true,"thumbnail":null,"downloads":["mp4"]}}`)
	})
	c.key = "lk_test"
	r, err := c.Llms(context.Background(), "example.com", LlmsOptions{Mode: ModeFull, Fresh: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.URL.Path != "/api/v0/llms" || got.URL.Query().Get("url") != "example.com" || got.URL.Query().Get("mode") != "full" || got.URL.Query().Get("fresh") != "1" {
		t.Errorf("request = %s", got.URL)
	}
	if got.Header.Get("Authorization") != "Bearer lk_test" || got.Header.Get("User-Agent") != "llms-ink-go/"+Version {
		t.Errorf("headers = %v", got.Header)
	}
	if r.Title != "Ex" || r.Mode != ModeFull || !r.Cached || *r.Model != "m" || r.Media == nil || r.Media.Duration != nil || r.Media.Downloads[0] != "mp4" {
		t.Errorf("result = %+v", r)
	}
}

func TestAnonymousAndEnv(t *testing.T) {
	t.Setenv("LLMS_INK_KEY", "lk_env")
	if c := New(); c.key != "lk_env" {
		t.Errorf("env key = %q", c.key)
	}
	c := stub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("anonymous request sent Authorization")
		}
		fmt.Fprint(w, `{"site":"https://a.b","url":"https://a.b/llms.txt","found":true,"status":200,"title":null,"tokens":1,"sections":1,"links":{"total":1,"checked":1,"broken":[{"url":"x","status":0}]},"llmsFull":false,"issues":[{"level":"error","code":"broken_link","line":3,"message":"m"}],"summary":{"errors":1,"warnings":0},"checkedAt":"t"}`)
	})
	r, err := c.Check(context.Background(), "a.b")
	if err != nil {
		t.Fatal(err)
	}
	if r.Title != nil || r.Links.Broken[0].URL != "x" || *r.Issues[0].Line != 3 || r.Summary.Errors != 1 || *r.Site != "https://a.b" || *r.LlmsFull || r.Parsed != nil {
		t.Errorf("check = %+v", r)
	}
}

func TestCheckText(t *testing.T) {
	var method, query, ctype, body string
	c := stub(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		method, query, ctype, body = r.Method+" "+r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), string(b)
		fmt.Fprint(w, `{"site":null,"url":null,"found":true,"status":200,"title":"T","tokens":5,"sections":1,"links":{"total":1,"checked":0,"broken":[]},"llmsFull":null,"issues":[],"summary":{"errors":0,"warnings":0},"parsed":{"title":"T","summary":"S","details":null,"sections":[{"name":"Optional","optional":true,"text":null,"links":[{"title":"A","url":"/a","notes":"n","line":5}]}]},"checkedAt":"t"}`)
	})
	r, err := c.CheckText(context.Background(), "# T\n", CheckTextOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if method != "POST /api/v0/check" || query != "" || ctype != "text/plain; charset=utf-8" || body != "# T\n" {
		t.Errorf("request = %s ?%s %s %q", method, query, ctype, body)
	}
	p := r.Parsed
	if r.Site != nil || r.URL != nil || r.LlmsFull != nil || p == nil || *p.Title != "T" || *p.Summary != "S" || p.Details != nil || !p.Sections[0].Optional || p.Sections[0].Text != nil {
		t.Fatalf("result = %+v", r)
	}
	if l := p.Sections[0].Links[0]; l.Title != "A" || l.URL != "/a" || *l.Notes != "n" || l.Line != 5 {
		t.Errorf("link = %+v", l)
	}
	if _, err := c.CheckText(context.Background(), "x", CheckTextOptions{Base: "https://a.b", Links: true}); err != nil {
		t.Fatal(err)
	}
	if query != "base=https%3A%2F%2Fa.b&links=1" {
		t.Errorf("query = %s", query)
	}
}

func TestErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		is     func(error) bool
	}{
		{401, "invalid_key", IsInvalidKey},
		{429, "rate_limited", IsRateLimited},
		{429, "quota_exceeded", IsQuotaExceeded},
	} {
		c := stub(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/problem+json")
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(tc.status)
			fmt.Fprintf(w, `{"type":"https://llms.ink/docs/#errors","title":"T","status":%d,"code":%q,"detail":"D"}`, tc.status, tc.code)
		})
		_, err := c.Llms(context.Background(), "x", LlmsOptions{})
		if !tc.is(fmt.Errorf("wrapped: %w", err)) {
			t.Errorf("%s: predicate false for %v", tc.code, err)
		}
		var e *Error
		if !errors.As(err, &e) || e.Status != tc.status || e.Code != tc.code || e.Title != "T" || e.Detail != "D" || e.RetryAfter != 30*time.Second {
			t.Errorf("%s: %+v", tc.code, e)
		}
		if IsInvalidKey(err) != (tc.code == "invalid_key") {
			t.Errorf("%s: IsInvalidKey mismatch", tc.code)
		}
	}
}

func TestNonJSONError(t *testing.T) {
	c := stub(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 502) })
	_, err := c.Check(context.Background(), "x")
	var e *Error
	if !errors.As(err, &e) || e.Status != 502 || e.Code != "" || e.Detail != "Bad Gateway" {
		t.Errorf("err = %#v", err)
	}
}
