package utils

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	WebViewStateKey = "state:web:view"
)

type WebViewState struct {
	URL       string                 `json:"url"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Scrape    map[string]interface{} `json:"scrape,omitempty"`
	Visual    map[string]interface{} `json:"visual,omitempty"`
}

func UpdateWebViewState(ctx context.Context, rdb *redis.Client, url string, category string, data interface{}) {
	if rdb == nil {
		return
	}

	// 1. Load current state
	var state WebViewState
	val, err := rdb.Get(ctx, WebViewStateKey).Result()
	if err == nil {
		_ = json.Unmarshal([]byte(val), &state)
	}

	// 2. Check for URL change
	if state.URL != url {
		// New URL requested, wipe everything
		state = WebViewState{
			URL:       url,
			Timestamp: time.Now().Unix(),
		}
	}

	// 3. Merge data
	dataJSON, _ := json.Marshal(data)
	var dataMap map[string]interface{}
	_ = json.Unmarshal(dataJSON, &dataMap)

	switch category {
	case "metadata":
		state.Metadata = dataMap
	case "scrape":
		state.Scrape = dataMap
	case "visual":
		// Requirement: Only store the LAST screenshot.
		// We overwrite the entire visual map, which naturally replaces the previous screenshot.
		state.Visual = dataMap
	}

	state.Timestamp = time.Now().Unix()

	// 4. Save back to Redis
	finalJSON, _ := json.Marshal(state)
	rdb.Set(ctx, WebViewStateKey, finalJSON, 24*time.Hour)

	log.Printf("WebViewState: Updated %s for URL: %s", category, url)
}
