package content

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"auto-poster/internal/ai"
)

type Post struct {
	Title            string `json:"title"`
	ShortDescription string `json:"short_description"`
	Content          string `json:"content"`
	ImagePrompt      string `json:"image_prompt"`
}

func BuildPrompt(topic string) string {
	return fmt.Sprintf(`Write a high-quality, original blog post about: %s

Return ONLY valid JSON, no markdown, in exactly this shape:
{"title": "A compelling title", "short_description": "A 1-2 sentence teaser in plain text", "content": "<p>HTML content...</p>", "image_prompt": "a detailed visual description for the blog cover image"}

Requirements:
- short_description must be plain text, no HTML, max ~30 words.
- Content must be valid HTML using <p>, <h2>, <ul>, <li>, <strong> tags.
- Make it 4-6 paragraphs, useful, and well-structured.
- Content should read naturally for a professional developer's blog.
- image_prompt: write ONE detailed sentence describing a relevant illustration for this specific topic (the subject matter, not generic "abstract tech"). Example for a CI/CD article: "a colorful pipeline diagram with code and robots automating builds on a dark background". No text or words in the image.`, topic)
}

func stripCodeFence(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	return strings.TrimSpace(raw)
}

func ParsePost(raw string) (*Post, error) {
	raw = stripCodeFence(raw)

	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	var p Post
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, fmt.Errorf("parse AI JSON: %w", err)
	}
	if strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Content) == "" {
		return nil, fmt.Errorf("AI returned empty title/content")
	}
	return &p, nil
}

func GeneratePost(ctx context.Context, provider ai.Provider, topic string) (*Post, error) {
	raw, err := provider.Generate(ctx, BuildPrompt(topic))
	if err != nil {
		return nil, err
	}
	return ParsePost(raw)
}

func GenerateTopics(ctx context.Context, provider ai.Provider, n int) ([]string, error) {
	if n < 1 {
		n = 1
	}

	prompt := fmt.Sprintf(`You are the editor of a professional software engineer's blog.
Suggest %d distinct, specific, original blog post topics. Focus on Go, backend engineering, PostgreSQL, DevOps, cloud infrastructure, and practical AI tooling.
Avoid generic listicles and overly broad subjects. Return ONLY a JSON array of %d strings, for example ["topic one", "topic two"].`, n, n)

	raw, err := provider.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}
	raw = stripCodeFence(raw)

	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	var topics []string
	if err := json.Unmarshal([]byte(raw), &topics); err != nil {
		return nil, fmt.Errorf("parse AI topics: %w", err)
	}

	clean := make([]string, 0, len(topics))
	for _, t := range topics {
		if t = strings.TrimSpace(t); t != "" {
			clean = append(clean, t)
		}
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("AI returned no topics")
	}
	if len(clean) > n {
		clean = clean[:n]
	}
	return clean, nil
}
