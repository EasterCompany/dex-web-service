package endpoints

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

// WebViewResponse represents the data extracted from a headless browser session.
type WebViewResponse struct {
	URL        string `json:"url"`
	Title      string `json:"title,omitempty"`
	Content    string `json:"content,omitempty"`    // Rendered HTML content
	Screenshot string `json:"screenshot,omitempty"` // Base64 encoded screenshot
	Error      string `json:"error,omitempty"`
}

// WebViewHandler handles requests to view a page in a headless browser.
func WebViewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	// Create context with timeout
	// 30 seconds should be enough for most pages to load
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize chromedp
	// We use the default allocator which tries to find Chrome/Chromium.
	// If it fails to find a browser, it will return an error during Run.
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var title, content string
	var buf []byte

	// Run tasks
	// 1. Navigate
	// 2. Wait for body (ensures at least basic rendering)
	// 3. Capture Title
	// 4. Capture HTML (DOM state after JS)
	// 5. Capture Screenshot (Visual state)
	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.OuterHTML(`html`, &content, chromedp.ByQuery),
		chromedp.CaptureScreenshot(&buf),
	)

	response := WebViewResponse{
		URL: targetURL,
	}

	if err != nil {
		log.Printf("Chromedp error for %s: %v", targetURL, err)
		response.Error = fmt.Sprintf("Failed to browse page: %v", err)
	} else {
		response.Title = title
		response.Content = content
		response.Screenshot = base64.StdEncoding.EncodeToString(buf)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding webview response: %v", err)
	}
}
