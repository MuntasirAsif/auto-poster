package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"auto-poster/internal/database"
	"auto-poster/internal/publisher"
	"auto-poster/internal/scheduler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.ApplyMigrations(); err != nil {
		log.Fatal(err)
	}

	repo := database.NewPostRepository(db)
	blogPub := publisher.DjangoBlogPublisherFromEnv()
	publishers := map[publisher.Platform]publisher.Publisher{
		publisher.PlatformBlog: blogPub,
	}

	ctx := context.Background()
	opts := scheduler.Options{}
	if settings, err := publisher.FetchSettings(ctx, blogPub.BaseURL, blogPub.Token); err == nil {
		log.Printf("autoposter settings: enabled=%v start=%s interval=%d tz=%s category=%q",
			settings.Enabled, settings.DailyPublishTime, settings.ScheduleIntervalMin, settings.Timezone, settings.DefaultCategory)
		if !settings.Enabled {
			log.Println("auto poster disabled in settings, exiting")
			os.Exit(0)
		}
		lastPublished, err := repo.LastPublishedAt(ctx)
		if err != nil {
			log.Printf("last published lookup: %v", err)
		}
		if !scheduler.PublishAllowed(settings, time.Now(), lastPublished) {
			log.Println("outside the configured publish window, skipping")
			os.Exit(0)
		}
		opts = scheduler.Options{DefaultCategory: settings.DefaultCategory, ContentFooter: settings.ContentFooter}
	} else {
		log.Printf("fetch settings (continuing without options): %v", err)
	}

	published, err := scheduler.ProcessDue(ctx, repo, publishers, opts)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("published %d due post(s)", published)

	if published == 0 {
		os.Exit(0)
	}
}
