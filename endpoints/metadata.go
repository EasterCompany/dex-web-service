package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/EasterCompany/dex-web-service/utils"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// MetadataResponse represents the structured data extracted from a URL.
type MetadataResponse struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
	Content     string `json:"content,omitempty"`
	Summary     string `json:"summary,omitempty"`
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

	ctx := r.Context()
	var rawHTML string

	// Try cache first
	rawHTML, err = utils.GetCachedPage(ctx, targetURL)
	if err != nil {
		// Fetch the URL content
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			log.Printf("Error creating request for URL %s: %v", targetURL, err)
			http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
			return
		}
		// Use a standard Desktop User-Agent to avoid mobile versions or blocking
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching URL %s: %v", targetURL, err)
			http.Error(w, fmt.Sprintf("Failed to fetch URL: %v", err), http.StatusInternalServerError)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNonAuthoritativeInfo {
			log.Printf("Received non-OK status for URL %s: %d", targetURL, resp.StatusCode)
			http.Error(w, fmt.Sprintf("Failed to fetch URL, status code: %d", resp.StatusCode), http.StatusInternalServerError)
			return
		}

		// Detect and convert charset to UTF-8
		utf8Reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
		if err != nil {
			log.Printf("Charset detection failed: %v", err)
			utf8Reader = resp.Body // Fallback
		}

		bodyBytes, err := io.ReadAll(utf8Reader)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return
		}
		rawHTML = string(bodyBytes)

		// Store in cache
		_ = utils.SetCachedPage(ctx, targetURL, rawHTML)
	}

	// Parse HTML from string
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		log.Printf("Error parsing HTML for URL %s: %v", targetURL, err)
		http.Error(w, fmt.Sprintf("Failed to parse HTML: %v", err), http.StatusInternalServerError)
		return
	}

	metadata := make(map[string]string)
	var title string

	// Traverse for metadata
	var traverseMetadata func(*html.Node)
	traverseMetadata = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "meta" {
				var property, name, content string
				for _, attr := range n.Attr {
					if attr.Key == "property" {
						property = attr.Val
					}
					if attr.Key == "name" {
						name = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if strings.HasPrefix(property, "og:") {
					metadata[property] = content
				}
				if strings.HasPrefix(name, "twitter:") {
					metadata[name] = content
				}
			} else if n.Data == "title" && title == "" {
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					title = n.FirstChild.Data
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverseMetadata(c)
		}
	}
	traverseMetadata(doc)

	// Prioritize Open Graph, then Twitter Card, then generic HTML elements
	response := MetadataResponse{
		URL: targetURL,
	}
	response.Title = metadata["og:title"]
	if response.Title == "" {
		response.Title = metadata["twitter:title"]
	}
	if response.Title == "" {
		response.Title = title
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
		return
	}

	// Update global Web View state
	go utils.UpdateWebViewState(context.Background(), utils.GetRedisClient(), targetURL, "metadata", response)
}
