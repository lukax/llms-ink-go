package llmsink_test

import (
	"context"
	"errors"
	"fmt"
	"log"

	llmsink "github.com/lukax/llms-ink-go"
)

func ExampleClient_Llms() {
	c := llmsink.New() // uses LLMS_INK_KEY if set
	r, err := c.Llms(context.Background(), "https://llmstxt.org", llmsink.LlmsOptions{Mode: llmsink.ModeFull})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r.Title, r.Tokens)
}

func ExampleClient_Check() {
	c := llmsink.New()
	r, err := c.Check(context.Background(), "llmstxt.org")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(r.Found, r.Summary.Errors, r.Summary.Warnings)
}

func ExampleError() {
	c := llmsink.New(llmsink.WithKey("lk_..."))
	_, err := c.Llms(context.Background(), "example.com", llmsink.LlmsOptions{})
	var e *llmsink.Error
	switch {
	case llmsink.IsInvalidKey(err):
		log.Fatal("unknown or revoked API key")
	case llmsink.IsRateLimited(err), llmsink.IsQuotaExceeded(err):
		errors.As(err, &e)
		log.Printf("slow down; retry in %s", e.RetryAfter)
	case err != nil:
		log.Fatal(err)
	}
}
