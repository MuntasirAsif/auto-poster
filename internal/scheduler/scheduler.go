package scheduler

import (
	"context"
	"time"

	"auto-poster/internal/database"
)

type Scheduler interface {
	Schedule(ctx context.Context, post *database.Post, at time.Time) error
}