package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AutoPosterSettings struct {
	Enabled              bool   `json:"enabled"`
	BlogAPIURL           string `json:"blog_api_url"`
	DefaultCategory      string `json:"default_category"`
	ContentFooter        string `json:"content_footer"`
	DailyPublishTime     string `json:"daily_publish_time"`
	ScheduleIntervalMin  int    `json:"schedule_interval_minutes"`
	Timezone             string `json:"timezone"`
}

func FetchSettings(ctx context.Context, baseURL, token string) (*AutoPosterSettings, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/autoposter/settings/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch autoposter settings failed (status %d)", resp.StatusCode)
	}

	var s AutoPosterSettings
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode autoposter settings: %w", err)
	}

	return &s, nil
}