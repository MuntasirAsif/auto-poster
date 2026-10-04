package scheduler

import (
	"context"
	"log"
	"time"

	"auto-poster/internal/database"
	"auto-poster/internal/publisher"
)

type Options struct {
	DefaultCategory string
	ContentFooter   string
}

func ProcessDue(ctx context.Context, repo *database.PostRepository, publishers map[publisher.Platform]publisher.Publisher, opts Options) (int, error) {
	due, err := repo.ListDue(ctx, time.Now())
	if err != nil {
		return 0, err
	}

	published := 0
	for i := range due {
		post := due[i]
		log.Printf("publishing post %d (%s): %s", post.ID, post.Platform, post.Title)

		pub, ok := publishers[publisher.Platform(post.Platform)]
		if !ok {
			msg := "no publisher for platform " + post.Platform
			now := time.Now()
			if err := repo.UpdateStatus(ctx, post.ID, "failed", &now, &msg); err != nil {
				log.Printf("post %d: mark failed: %v", post.ID, err)
			}
			continue
		}

		now := time.Now()
		category := post.Platform
		if opts.DefaultCategory != "" {
			category = opts.DefaultCategory
		}
		content := post.Content
		if opts.ContentFooter != "" {
			content += "\n\n" + opts.ContentFooter
		}

		_, err := pub.Publish(ctx, publisher.PublishRequest{
			Title:    post.Title,
			Content:  content,
			Category: category,
		})
		if err != nil {
			msg := err.Error()
			if err := repo.UpdateStatus(ctx, post.ID, "failed", &now, &msg); err != nil {
				log.Printf("post %d: mark failed: %v", post.ID, err)
			}
			continue
		}

		if err := repo.UpdateStatus(ctx, post.ID, "published", &now, nil); err != nil {
			log.Printf("post %d: mark published: %v", post.ID, err)
		}
		published++
	}

	return published, nil
}

type Worker struct {
	repo       *database.PostRepository
	publishers map[publisher.Platform]publisher.Publisher
	interval   time.Duration
	baseURL    string
	token      string
}

func NewWorker(repo *database.PostRepository, publishers map[publisher.Platform]publisher.Publisher, interval time.Duration, baseURL, token string) *Worker {
	return &Worker{repo: repo, publishers: publishers, interval: interval, baseURL: baseURL, token: token}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler stopped")
			return
		case <-ticker.C:
			opts := Options{}
			if settings, err := publisher.FetchSettings(ctx, w.baseURL, w.token); err == nil {
				if !settings.Enabled {
					log.Println("scheduler: auto poster disabled in settings, skipping")
					continue
				}
				opts = Options{DefaultCategory: settings.DefaultCategory, ContentFooter: settings.ContentFooter}
			} else {
				log.Printf("scheduler: fetch settings: %v", err)
			}

			if _, err := ProcessDue(ctx, w.repo, w.publishers, opts); err != nil {
				log.Printf("scheduler: process due: %v", err)
			}
		}
	}
}