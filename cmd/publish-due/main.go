package main

import (
	"context"
	"log"
	"os"

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
	publishers := map[publisher.Platform]publisher.Publisher{
		publisher.PlatformBlog: publisher.DjangoBlogPublisherFromEnv(),
	}

	published, err := scheduler.ProcessDue(context.Background(), repo, publishers)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("published %d due post(s)", published)

	if published == 0 {
		os.Exit(0)
	}
}