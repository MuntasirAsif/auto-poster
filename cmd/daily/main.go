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

	count := flag.Int("count", 3, "number of posts to generate and publish")
	withImage := flag.Bool("image", false, "generate a cover image for each post (free, via AI Horde)")
	flag.Parse()

	if *count < 1 {
		*count = 1
	}

	provider := ai.GeminiProviderFromEnv()
	if provider.APIKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	blogPub := publisher.DjangoBlogPublisherFromEnv()

	category := "blog"
	footer := ""
	if settings, err := publisher.FetchSettings(ctx, blogPub.BaseURL, blogPub.Token); err != nil {
		log.Printf("fetch settings (continuing without): %v", err)
	} else {
		if !settings.Enabled {
			log.Println("auto poster disabled in settings, exiting")
			return
		}
		if settings.DefaultCategory != "" {
			category = settings.DefaultCategory
		}
		footer = settings.ContentFooter
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

	published := 0
	failures := 0
	for published < *count && failures < 5 {
		topics, err := content.GenerateTopics(ctx, provider, 1)
		if err != nil {
			failures++
			log.Printf("topic generation failed (%d/5): %v", failures, err)
			time.Sleep(15 * time.Second)
			continue
		}
		topic := topics[0]
		log.Printf("[%d/%d] %s", published+1, *count, topic)

		post, err := content.GeneratePost(ctx, provider, topic)
		if err != nil {
			failures++
			log.Printf("  generate failed (%d/5): %v", failures, err)
			time.Sleep(15 * time.Second)
			continue
		}

		var imageBytes []byte
		imageURL := ""
		if *withImage {
			imgProvider := ai.AIHordeProviderFromEnv()
			imgPrompt := post.ImagePrompt
			if strings.TrimSpace(imgPrompt) == "" {
				imgPrompt = fmt.Sprintf("Blog cover illustration for an article titled %q. Modern technology theme, professional, high quality, no text.", post.Title)
			}
			if img, err := imgProvider.Generate(ctx, imgPrompt); err != nil {
				log.Printf("  image failed: %v (publishing without cover)", err)
			} else {
				imageBytes = img.Data
				imageURL = img.URL
			}
		}

		now := time.Now()
		created, err := repo.Create(ctx, post.Title, post.ShortDescription, imageURL, "blog", post.Content, nil)
		if err != nil {
			log.Printf("  record failed: %v", err)
		}

		body := post.Content
		if footer != "" {
			body += "\n\n" + footer
		}

		url, err := blogPub.Publish(ctx, publisher.PublishRequest{
			Title:            post.Title,
			Content:          body,
			Category:         category,
			ShortDescription: post.ShortDescription,
			Image:            imageBytes,
		})
		if err != nil {
			if created != nil {
				msg := err.Error()
				repo.UpdateStatus(ctx, created.ID, "failed", &now, &msg)
			}
			failures++
			log.Printf("  publish failed (%d/5): %v", failures, err)
			time.Sleep(15 * time.Second)
			continue
		}
		if created != nil {
			if err := repo.UpdateStatus(ctx, created.ID, "published", &now, nil); err != nil {
				log.Printf("  mark published failed: %v", err)
			}
		}
		published++
		log.Printf("  published: %s%s", blogPub.BaseURL, url)
	}

	log.Printf("done: %d/%d post(s) published", published, *count)
	if published == 0 {
		log.Fatal("no posts were published")
	}
}
