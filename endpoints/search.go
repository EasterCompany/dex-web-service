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

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// SearchHandler performs a search via DuckDuckGo HTML
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// DuckDuckGo HTML Search URL
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", searchURL, nil)
	// Important: DuckDuckGo HTML needs a real-looking User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Failed to perform search", http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("DuckDuckGo returned status: %d", resp.StatusCode), http.StatusServiceUnavailable)
		return
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse search results", http.StatusInternalServerError)
		return
	}

	var results []SearchResult
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			// DuckDuckGo HTML results are in divs with class 'result' or 'web-result'
			isResult := false
			for _, attr := range n.Attr {
				if attr.Key == "class" && (strings.Contains(attr.Val, "result") || strings.Contains(attr.Val, "web-result")) {
					isResult = true
					break
				}
			}

			if isResult {
				res := parseSearchResult(n)
				if res.URL != "" && !strings.Contains(res.URL, "duckduckgo.com/y.js") { // Filter ads
					results = append(results, res)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(doc)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func parseSearchResult(n *html.Node) SearchResult {
	var res SearchResult
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "a" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result__a") {
						// This is the title and link
						res.Title = getText(n)
						for _, a := range n.Attr {
							if a.Key == "href" {
								// DDG uses relative links or redirects sometimes, but HTML version is usually clean
								u, err := url.Parse(a.Val)
								if err == nil {
									// Extract real URL from DDG redirect if needed
									if u.Path == "/l/" {
										res.URL = u.Query().Get("uddg")
									} else {
										res.URL = a.Val
									}
								}
							}
						}
					}
				}
			}
			if n.Data == "a" && res.Snippet == "" {
				for _, attr := range n.Attr {
					if attr.Key == "class" && strings.Contains(attr.Val, "result__snippet") {
						res.Snippet = getText(n)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return res
}

func getText(n *html.Node) string {
	var sb strings.Builder
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(n)
	return strings.TrimSpace(sb.String())
}

// ScrapeHandler performs a high-fidelity scrape of a URL
func ScrapeHandler(w http.ResponseWriter, r *http.Request) {
	// For now, we can reuse the MetadataHandler logic but maybe with different parameters if needed
	// Or we just proxy it to MetadataHandler for now as it already does text extraction.
	MetadataHandler(w, r)
}
