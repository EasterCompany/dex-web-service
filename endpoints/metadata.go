package endpoints

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// MetadataResponse represents the structured data extracted from a URL.
type MetadataResponse struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	ContentType string `json:"content_type,omitempty"` // e.g., "image/gif", "text/html"
	Provider    string `json:"provider,omitempty"`     // e.g., "Tenor", "Giphy"
	Error       string `json:"error,omitempty"`
}

// MetadataHandler fetches a URL, extracts Open Graph/Twitter Card metadata, and returns it.
func MetadataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", targetURL, err)
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Fetch the URL content
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		log.Printf("Error fetching URL %s: %v", targetURL, err)
		http.Error(w, fmt.Sprintf("Failed to fetch URL: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Received non-OK status for URL %s: %d", targetURL, resp.StatusCode)
		http.Error(w, fmt.Sprintf("Failed to fetch URL, status code: %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	// Parse HTML to extract metadata
	tokenizer := html.NewTokenizer(resp.Body)
	metadata := make(map[string]string)

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			token := tokenizer.Token()
			if token.Data == "meta" {
				var property, name, content string
				for _, attr := range token.Attr {
					if attr.Key == "property" {
						property = attr.Val
					}
					if attr.Key == "name" { // For Twitter cards
						name = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}

				if strings.HasPrefix(property, "og:") {
					metadata[property] = content
				}
				if strings.HasPrefix(name, "twitter:") { // Also capture Twitter card data
					metadata[name] = content
				}
			}
			if token.Data == "title" { // Fallback for title
				tokenizer.Next() // Get to the text token
				titleToken := tokenizer.Token()
				if titleToken.Type == html.TextToken {
					if _, ok := metadata["og:title"]; !ok { // Prefer OG title if available
						metadata["og:title"] = titleToken.Data
					}
				}
			}
		}
	}

	// Construct the response
	response := MetadataResponse{
		URL: targetURL,
	}

	// Prioritize Open Graph, then Twitter Card, then generic HTML elements
	response.Title = metadata["og:title"]
	if response.Title == "" {
		response.Title = metadata["twitter:title"]
	}

	response.Description = metadata["og:description"]
	if response.Description == "" {
		response.Description = metadata["twitter:description"]
	}

	response.ImageURL = metadata["og:image"]
	if response.ImageURL == "" {
		response.ImageURL = metadata["twitter:image"]
	}

	// Try to infer content type and provider from the URL
	if response.ImageURL != "" {
		if strings.Contains(strings.ToLower(response.ImageURL), ".gif") {
			response.ContentType = "image/gif"
		} else if strings.Contains(strings.ToLower(response.ImageURL), ".jpg") || strings.Contains(strings.ToLower(response.ImageURL), ".jpeg") {
			response.ContentType = "image/jpeg"
		} else if strings.Contains(strings.ToLower(response.ImageURL), ".png") {
			response.ContentType = "image/png"
		}

		host := parsedURL.Host
		if strings.Contains(host, "tenor.com") {
			response.Provider = "Tenor"
		} else if strings.Contains(host, "giphy.com") {
			response.Provider = "Giphy"
		} else {
			response.Provider = strings.Split(host, ".")[0] // e.g., "media.giphy.com" -> "media"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding metadata response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
