# llms-ink-go

[![Go Reference](https://pkg.go.dev/badge/github.com/lukax/llms-ink-go.svg)](https://pkg.go.dev/github.com/lukax/llms-ink-go)

Go client for [llms.ink](https://llms.ink): turn any public link — a site, docs page, GitHub repo, PDF or video — into `llms.txt` or `llms-full.txt`, and check a site's own `llms.txt`. Standard library only, Go 1.21+.

```sh
go get github.com/lukax/llms-ink-go
```

```go
import llmsink "github.com/lukax/llms-ink-go"

c := llmsink.New() // reads LLMS_INK_KEY if set; anonymous otherwise
r, err := c.Llms(ctx, "https://llmstxt.org", llmsink.LlmsOptions{Mode: llmsink.ModeFull})
fmt.Println(r.Title, r.Tokens, r.Text)

chk, err := c.Check(ctx, "llmstxt.org")
fmt.Println(chk.Found, chk.Summary.Errors, chk.Summary.Warnings)
```

`LlmsOptions{Mode, Fresh}`: `ModeLLMS` (default, the index) or `ModeFull` (the content); `Fresh: true` skips the cache.

Options: `WithKey(key)`, `WithBaseURL(url)`, `WithHTTPClient(*http.Client)`.

## API key (optional)

Without a key, requests are anonymous and share per-IP limits. A key (`Authorization: Bearer lk_…`) gets its own daily quota. Set `LLMS_INK_KEY` or pass `WithKey`. An invalid or revoked key fails with 401 `invalid_key`; it never falls back to anonymous. [Request a key](https://lucasespindola.com/message?text=llms.ink%20API%20key).

## Errors

API errors are `*llmsink.Error` (`Status`, `Code`, `Title`, `Detail`, `RetryAfter`), from the API's [problem details](https://llms.ink/docs/#errors):

```go
var e *llmsink.Error
switch {
case llmsink.IsInvalidKey(err):
case llmsink.IsRateLimited(err), llmsink.IsQuotaExceeded(err):
	errors.As(err, &e)
	time.Sleep(e.RetryAfter)
case errors.As(err, &e):
	log.Print(e.Code, e.Detail)
}
```

## langchaingo

A [langchaingo](https://github.com/tmc/langchaingo) document loader lives in a separate module so the root stays dependency-free:

```sh
go get github.com/lukax/llms-ink-go/langchaingo
```

```go
l := langchaingo.NewLoader(nil, llmsink.LlmsOptions{Mode: llmsink.ModeFull}, "https://llmstxt.org")
docs, err := l.Load(ctx)
```

## Development

```sh
go vet ./... && go test ./...
go test -tags live -run Live -v .   # hits https://llms.ink
(cd langchaingo && go test ./...)
```

Release: tag `vX.Y.Z` for the root and `langchaingo/vX.Y.Z` for the loader; then drop the `replace` in `langchaingo/go.mod`.

MIT License.
