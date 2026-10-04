package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ImageResult struct {
	URL  string
	Data []byte
}

type ImageProvider interface {
	Generate(ctx context.Context, prompt string) (*ImageResult, error)
}

const anonymousHordeKey = "0000000000"

type AIHordeProvider struct {
	APIKey   string
	Client   *http.Client
	Timeout  time.Duration
}

func NewAIHordeProvider(apiKey string) *AIHordeProvider {
	if apiKey == "" {
		apiKey = anonymousHordeKey
	}
	return &AIHordeProvider{
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: 30 * time.Second},
		Timeout: 6 * time.Minute,
	}
}

func AIHordeProviderFromEnv() *AIHordeProvider {
	return NewAIHordeProvider(os.Getenv("AI_HORDE_API_KEY"))
}

func (p *AIHordeProvider) Generate(ctx context.Context, prompt string) (*ImageResult, error) {
	payload, err := json.Marshal(map[string]any{
		"prompt": prompt,
		"params": map[string]any{
			"width":  512,
			"height": 512,
			"steps":  20,
			"cfg_scale": 7,
		},
		"nsfw": false,
		"r2":   true,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://stablehorde.net/api/v2/generate/async", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("horde submit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("horde submit error (status %d): %s", resp.StatusCode, string(body))
	}

	var submitted struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&submitted); err != nil {
		return nil, fmt.Errorf("horde submit decode: %w", err)
	}
	if submitted.ID == "" {
		return nil, fmt.Errorf("horde submit returned no job id")
	}

	deadline := time.Now().Add(p.Timeout)
	checkURL := "https://stablehorde.net/api/v2/generate/check/" + submitted.ID

	lastLog := time.Time{}
	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("horde image generation timed out after %s", p.Timeout)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}

		done, pos, err := p.check(ctx, checkURL)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		if time.Since(lastLog) > 20*time.Second {
			lastLog = time.Now()
			fmt.Printf("  horde: job %s still processing (queue position ~%d)...\n", submitted.ID, pos)
		}
	}

	return p.fetchResult(ctx, submitted.ID)
}

func (p *AIHordeProvider) check(ctx context.Context, url string) (bool, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, 0, err
	}
	req.Header.Set("apikey", p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	var r struct {
		Done          bool `json:"done"`
		QueuePosition int  `json:"queue_position"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return false, 0, fmt.Errorf("horde check decode: %w", err)
	}
	return r.Done, r.QueuePosition, nil
}

func (p *AIHordeProvider) fetchResult(ctx context.Context, id string) (*ImageResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://stablehorde.net/api/v2/generate/status/"+id, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", p.APIKey)

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var r struct {
		Generations []struct {
			Img string `json:"img"`
		} `json:"generations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("horde status decode: %w", err)
	}
	if len(r.Generations) == 0 {
		return nil, fmt.Errorf("horde returned no generations")
	}

	img := r.Generations[0].Img
	var data []byte
	url := ""

	if strings.HasPrefix(img, "http") {
		url = img
		data, err = DownloadImage(ctx, img)
		if err != nil {
			return nil, fmt.Errorf("horde download: %w", err)
		}
	} else {
		data, err = base64.StdEncoding.DecodeString(img)
		if err != nil {
			return nil, fmt.Errorf("horde decode image: %w", err)
		}
	}

	return &ImageResult{URL: url, Data: data}, nil
}

func DownloadImage(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download image: status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}