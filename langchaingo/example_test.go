package langchaingo_test

import (
	"context"
	"fmt"
	"log"

	llmsink "github.com/lukax/llms-ink-go"
	"github.com/lukax/llms-ink-go/langchaingo"
)

func ExampleNewLoader() {
	l := langchaingo.NewLoader(nil, llmsink.LlmsOptions{Mode: llmsink.ModeFull}, "https://llmstxt.org")
	docs, err := l.Load(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(docs[0].Metadata["title"])
}
