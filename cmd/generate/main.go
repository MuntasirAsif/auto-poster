package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"auto-poster/internal/ai"
	"auto-poster/internal/content"
	"auto-poster/internal/database"
	"auto-poster/internal/publisher"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	topic := flag.String("topic", "", "topic for the AI-generated post")
	at := flag.String("at", "", "schedule time (RFC3339 or HH:MM); default now")
	publishNow := flag.Bool("publish", false, "publish immediately to the blog instead of just scheduling")
	withImage := flag.Bool("image", false, "also generate a cover image (free, via AI Horde)")
	flag.Parse()

	if strings.TrimSpace(*topic) == "" {
		log.Fatal("-topic is required, e.g. -topic \"Go concurrency best practices\"")
	}

	provider := ai.GeminiProviderFromEnv()
	if provider.APIKey == "" {
		log.Fatal("GEMINI_API_KEY not set (add it to .env)")
	}

	log.Printf("generating AI post for topic: %q (model %s)...", *topic, provider.Model)
	raw, err := provider.Generate(context.Background(), content.BuildPrompt(*topic))
	if err != nil {
		log.Fatal(err)
	}

	post, err := content.ParsePost(raw)
	if err != nil {
		log.Fatalf("generated content not parseable: %v\nraw output:\n%s", err, raw)
	}

	fmt.Println("=== Generated post ===")
	fmt.Println("Title:", post.Title)
	fmt.Println("Description:", post.ShortDescription)
	fmt.Println("Content:", post.Content)
	fmt.Println("======================")

	imageURL := ""
	var imageBytes []byte
	if *withImage {
		fmt.Println("Generating cover image via AI Horde (may take ~1-3 min)...")
		imgProvider := ai.AIHordeProviderFromEnv()
		imgPrompt := post.ImagePrompt
		if strings.TrimSpace(imgPrompt) == "" {
			imgPrompt = fmt.Sprintf("Blog cover illustration for an article titled %q. Modern technology theme, professional, high quality, no text.", post.Title)
		}
		fmt.Println("Image prompt:", imgPrompt)
		img, err := imgProvider.Generate(context.Background(), imgPrompt)
		if err != nil {
			log.Fatalf("image generation failed: %v", err)
		}
		imageURL = img.URL
		imageBytes = img.Data
		fmt.Printf("cover image ready (%d bytes, %s)\n", len(imageBytes), imageURL)
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

	var scheduledAt *time.Time
	if *at != "" {
		t, err := parseTime(*at)
		if err != nil {
			log.Fatal(err)
		}
		scheduledAt = &t
	} else {
		now := time.Now()
		scheduledAt = &now
	}

	created, err := repo.Create(context.Background(), post.Title, post.ShortDescription, imageURL, "blog", post.Content, scheduledAt)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("queued post #%d (status=%s, scheduled=%s)", created.ID, created.Status, created.ScheduledAt.Format(time.RFC3339))

	if *publishNow {
		blogPub := publisher.DjangoBlogPublisherFromEnv()
		url, err := blogPub.Publish(context.Background(), publisher.PublishRequest{
			Title:            created.Title,
			Content:          created.Content,
			ShortDescription: created.ShortDescription,
			Image:            imageBytes,
		})
		if err != nil {
			msg := err.Error()
			now := time.Now()
			repo.UpdateStatus(context.Background(), created.ID, "failed", &now, &msg)
			log.Fatal(err)
		}
		now := time.Now()
		if err := repo.UpdateStatus(context.Background(), created.ID, "published", &now, nil); err != nil {
			log.Fatal(err)
		}
		log.Printf("published: https://www.muntasirashif.com%s", url)
	}
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02T15:04:05-0700", s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("15:04", s, time.Local); err == nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("invalid time %q: use RFC3339 or HH:MM", s)
}
