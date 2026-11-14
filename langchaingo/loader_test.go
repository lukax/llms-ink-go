package langchaingo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	llmsink "github.com/lukax/llms-ink-go"
	"github.com/tmc/langchaingo/textsplitter"
)

func TestLoader(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"url":"https://%s/","kind":"site","source":"native","mode":"llms","title":"T","text":"# T\n\nbody text here","tokens":3,"pages":0,"model":null,"cached":true,"generatedAt":"t"}`, r.URL.Query().Get("url"))
	}))
	defer s.Close()
	l := NewLoader(llmsink.New(llmsink.WithBaseURL(s.URL), llmsink.WithKey("")), llmsink.LlmsOptions{}, "a.com", "b.com")
	docs, err := l.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 || docs[1].Metadata["source"] != "https://b.com/" || docs[0].PageContent == "" {
		t.Fatalf("docs = %+v", docs)
	}
	split, err := l.LoadAndSplit(context.Background(), textsplitter.NewRecursiveCharacter(textsplitter.WithChunkSize(8), textsplitter.WithChunkOverlap(0)))
	if err != nil || len(split) <= 2 {
		t.Fatalf("split = %d, %v", len(split), err)
	}
}
