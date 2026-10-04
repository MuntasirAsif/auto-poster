package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/joho/godotenv"

	"auto-poster/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	title := flag.String("title", "", "post title")
	content := flag.String("content", "", "post content")
	platform := flag.String("platform", "blog", "target platform (blog)")
	at := flag.String("at", "", "schedule time (RFC3339 or HH:MM); default now+1min")
	flag.Parse()

	if *title == "" || *content == "" {
		log.Fatal("-title and -content are required")
	}

	var scheduledAt *time.Time
	if *at != "" {
		if t, err := time.Parse(time.RFC3339, *at); err == nil {
			scheduledAt = &t
		} else if t, err := time.Parse("2006-01-02T15:04:05-0700", *at); err == nil {
			scheduledAt = &t
		} else if t, err := time.ParseInLocation("15:04", *at, time.Local); err == nil {
			now := time.Now()
			t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
			scheduledAt = &t
		} else {
			log.Fatalf("invalid -at %q: use RFC3339 or HH:MM", *at)
		}
	} else {
		t := time.Now().Add(time.Minute)
		scheduledAt = &t
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := database.NewPostRepository(db)
	post, err := repo.Create(context.Background(), *title, *platform, *content, scheduledAt)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("scheduled post #%d: %q at %s", post.ID, post.Title, post.ScheduledAt.Format(time.RFC3339))
}