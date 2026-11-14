// Package langchaingo is a langchaingo document loader for llms.ink: load any
// public link — a site, docs page, GitHub repo, PDF or video — as
// llms.txt or llms-full.txt Markdown.
package langchaingo

import (
	"context"

	llmsink "github.com/lukax/llms-ink-go"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// Loader loads links through llms.ink, one document per link.
type Loader struct {
	Client *llmsink.Client
	URLs   []string
	Opts   llmsink.LlmsOptions
}

var _ documentloaders.Loader = Loader{}

// NewLoader returns a Loader for urls. A nil client means llmsink.New().
func NewLoader(client *llmsink.Client, opts llmsink.LlmsOptions, urls ...string) Loader {
	if client == nil {
		client = llmsink.New()
	}
	return Loader{Client: client, URLs: urls, Opts: opts}
}

// Load fetches every URL and returns one document each; metadata mirrors llmsink.Result.
func (l Loader) Load(ctx context.Context) ([]schema.Document, error) {
	docs := make([]schema.Document, 0, len(l.URLs))
	for _, u := range l.URLs {
		r, err := l.Client.Llms(ctx, u, l.Opts)
		if err != nil {
			return nil, err
		}
		meta := map[string]any{
			"source": r.URL, "title": r.Title, "kind": r.Kind, "origin": r.Source,
			"mode": string(r.Mode), "tokens": r.Tokens, "pages": r.Pages,
			"cached": r.Cached, "generatedAt": r.GeneratedAt,
		}
		docs = append(docs, schema.Document{PageContent: r.Text, Metadata: meta})
	}
	return docs, nil
}

// LoadAndSplit loads and splits the documents with splitter.
func (l Loader) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	docs, err := l.Load(ctx)
	if err != nil {
		return nil, err
	}
	return textsplitter.SplitDocuments(splitter, docs)
}
