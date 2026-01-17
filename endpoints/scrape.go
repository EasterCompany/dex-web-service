package endpoints

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/EasterCompany/dex-web-service/utils"
	"golang.org/x/net/html"
)

// ScrapeResponse holds the high-fidelity scraped content
type ScrapeResponse struct {
	URL     string `json:"url"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// ScrapeHandler performs a high-fidelity "Smart Scrape" of a URL
func ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	// Fetch URL
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Scrape fetch error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch URL: %v", err), http.StatusServiceUnavailable)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("URL returned status: %d", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse HTML", http.StatusInternalServerError)
		return
	}

	// Perform Smart Extraction
	content, err := utils.ExtractMainContent(doc, targetURL)
	if err != nil {
		// Fallback to empty content if extraction fails (should be rare with fallback to body)
		content = ""
	}

	response := ScrapeResponse{
		URL:     targetURL,
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding scrape response: %v", err)
	}
}
