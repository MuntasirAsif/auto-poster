package content

import (
	"context"

	"auto-poster/internal/research"
)

type Draft struct {
	Platform string
	Content  string
}

type Generator interface {
	Generate(ctx context.Context, topic string, sources []research.Source) ([]Draft, error)
}