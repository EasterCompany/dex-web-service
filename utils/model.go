package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/EasterCompany/dex-web-service/config"
)

// GenerateSummary calls the Model Hub to summarize the provided content.
func GenerateSummary(content string) (string, error) {
	if content == "" {
		return "", nil
	}

	// Resolve Model Hub address
	hubHost, err := config.ResolveServiceHost("dex-model-service")
	if err != nil {
		return "", fmt.Errorf("failed to resolve model hub: %w", err)
	}

	// Limit content length to avoid overwhelming the model
	if len(content) > 12000 {
		content = content[:12000]
	}

	reqBody := map[string]interface{}{
		"model":  "dex-scraper-model",
		"prompt": content,
		"stream": false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Model Hub
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(fmt.Sprintf("http://%s/model/run", hubHost), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to call model hub at %s: %w", hubHost, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("model hub returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Response, nil
}
