package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultBlogAPIURL = "http://127.0.0.1:8000"

func DjangoBlogPublisherFromEnv() *DjangoBlogPublisher {
	baseURL := os.Getenv("BLOG_API_URL")
	if baseURL == "" {
		baseURL = defaultBlogAPIURL
	}
	return NewDjangoBlogPublisher(baseURL, os.Getenv("BLOG_API_TOKEN"))
}

type DjangoBlogPublisher struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewDjangoBlogPublisher(baseURL, token string) *DjangoBlogPublisher {
	return &DjangoBlogPublisher{
		BaseURL: baseURL,
		Token:   token,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *DjangoBlogPublisher) Platform() Platform {
	return PlatformBlog
}

func (p *DjangoBlogPublisher) Publish(ctx context.Context, req PublishRequest) (string, error) {
	payload, err := json.Marshal(map[string]string{
		"title":             req.Title,
		"content":           req.Content,
		"category":          req.Category,
		"short_description": req.ShortDescription,
	})
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/blog/posts/", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.Token)

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("blog publish: read response: %w", err)
	}

	var result struct {
		ID    int64  `json:"id"`
		Title string `json:"title"`
		Slug  string `json:"slug"`
		URL   string `json:"url"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("blog publish: decode response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("blog publish failed (status %d): %s", resp.StatusCode, result.Error)
	}

	return result.URL, nil
}