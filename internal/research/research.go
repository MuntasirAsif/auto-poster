package research

import "context"

type Source struct {
	Title   string
	URL     string
	Snippet string
}

type Service interface {
	Research(ctx context.Context, topic string) ([]Source, error)
}