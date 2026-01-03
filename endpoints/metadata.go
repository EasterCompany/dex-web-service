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
	Content     string `json:"content,omitempty"`
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

	// Parse HTML
	doc, err := html.Parse(resp.Body)
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

	// Helper to resolve relative URLs
	resolveURL := func(base, ref string) string {
		u, err := url.Parse(ref)
		if err != nil {
			return ref
		}
		if u.IsAbs() {
			return ref
		}
		baseURL, err := url.Parse(base)
		if err != nil {
			return ref
		}
		return baseURL.ResolveReference(u).String()
	}

	// Extract text content from body
	var textBuilder strings.Builder
	var currentHref string
	var traverseText func(*html.Node)
	traverseText = func(n *html.Node) {
		if n.Type == html.ElementNode {
			// Skip script, style, and noscript tags
			if n.Data == "script" || n.Data == "style" || n.Data == "noscript" || n.Data == "iframe" {
				return
			}

			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						currentHref = resolveURL(targetURL, attr.Val)
						break
					}
				}
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if currentHref != "" {
					// Basic deduplication for text nodes within same link
					textBuilder.WriteString(fmt.Sprintf("[%s](%s) ", text, currentHref))
				} else {
					textBuilder.WriteString(text)
					textBuilder.WriteString(" ")
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverseText(c)
		}

		if n.Type == html.ElementNode && n.Data == "a" {
			currentHref = ""
		}
	}

	// Find body node to start text extraction
	var bodyNode *html.Node
	var findBody func(*html.Node)
	findBody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			bodyNode = n
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findBody(c)
			if bodyNode != nil {
				return
			}
		}
	}
	findBody(doc)

	if bodyNode != nil {
		traverseText(bodyNode)
	}

	// Construct the response
	response := MetadataResponse{
		URL:     targetURL,
		Content: strings.TrimSpace(textBuilder.String()),
	}

	// Prioritize Open Graph, then Twitter Card, then generic HTML elements
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
	}
}
