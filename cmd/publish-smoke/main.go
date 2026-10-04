package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"auto-poster/internal/publisher"
)

func main() {
	baseURL := os.Getenv("BLOG_API_URL")
	token := os.Getenv("BLOG_API_TOKEN")

	p := publisher.NewDjangoBlogPublisher(baseURL, token)
	url, err := p.Publish(context.Background(), publisher.PublishRequest{
		Title:            "Go Auto Poster Smoke Test",
		Content:          "<p>Published from the Go auto-poster publisher.</p>",
		Category:         "tech",
		ShortDescription: "End-to-end smoke test",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Published:", url)
}