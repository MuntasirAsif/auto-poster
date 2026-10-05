package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const geminiOpenAIURL = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
const defaultModel = "gemini-3.8-flash"

type GeminiProvider struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = defaultModel
	}
	return &GeminiProvider{
		APIKey: apiKey,
		Model:  model,
		Client: &http.Client{Timeout: 60 * time.Second},
	}
}

func GeminiProviderFromEnv() *GeminiProvider {
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = defaultModel
	}
	return NewGeminiProvider(os.Getenv("GEMINI_API_KEY"), model)
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("gemini error (status %d): %s", e.StatusCode, e.Message)
}

func (e *HTTPError) retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", err
	}

	var lastErr error
	backoff := 2 * time.Second
	for attempt := 1; attempt <= 5; attempt++ {
		if attempt > 1 {
			wait := backoff
			var he *HTTPError
			if errors.As(lastErr, &he) && he.StatusCode == http.StatusTooManyRequests {
				// Rate limited: back off hard instead of hammering the quota.
				wait = 60 * time.Second
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return "", ctx.Err()
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}

		resp, err := p.post(ctx, body)
		if err != nil {
			var he *HTTPError
			if errors.As(err, &he) && !he.retryable() {
				return "", err
			}
			lastErr = err
			continue
		}
		if resp == "" {
			continue
		}
		return resp, nil
	}

	return "", fmt.Errorf("gemini request failed after retries: %w", lastErr)
}

func (p *GeminiProvider) post(ctx context.Context, body []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, geminiOpenAIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		raw := make([]byte, 0)
		raw, _ = io.ReadAll(resp.Body)
		json.Unmarshal(raw, &errBody)
		return "", &HTTPError{StatusCode: resp.StatusCode, Message: errBody.Error.Message}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("gemini decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("gemini returned no choices")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
