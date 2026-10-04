package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
		Client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (p *DjangoBlogPublisher) Platform() Platform {
	return PlatformBlog
}

func (p *DjangoBlogPublisher) Publish(ctx context.Context, req PublishRequest) (string, error) {
	var httpReq *http.Request
	var err error

	if len(req.Image) > 0 {
		httpReq, err = p.multipartRequest(ctx, req)
	} else {
		httpReq, err = p.jsonRequest(ctx, req)
	}
	if err != nil {
		return "", err
	}
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

func (p *DjangoBlogPublisher) jsonRequest(ctx context.Context, req PublishRequest) (*http.Request, error) {
	payload, err := json.Marshal(map[string]string{
		"title":             req.Title,
		"content":           req.Content,
		"category":          req.Category,
		"short_description": req.ShortDescription,
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/blog/posts/", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq, nil
}

func (p *DjangoBlogPublisher) multipartRequest(ctx context.Context, req PublishRequest) (*http.Request, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fields := map[string]string{
		"title":             req.Title,
		"content":           req.Content,
		"category":          req.Category,
		"short_description": req.ShortDescription,
	}
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, err
		}
	}

	part, err := writer.CreateFormFile("image", "cover.jpg")
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(req.Image); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.BaseURL+"/api/blog/posts/", &buf)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	return httpReq, nil
}