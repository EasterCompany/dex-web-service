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
	// 90 seconds should be enough for most pages to load even on slow hardware
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Initialize chromedp
	// We use the default allocator which tries to find Chrome/Chromium.
	// If it fails to find a browser, it will return an error during Run.
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var title, content string
	var buf []byte
	var pageHeight float64

	// Run tasks
	// 1. Emulate Mobile (iPhone X width)
	// 2. Navigate & Wait
	// 3. Capture Meta
	// 4. Measure Height
	// 5. Resize Viewport to content height (capped at 5000px)
	// 6. Capture Screenshot
	err := chromedp.Run(ctx,
		chromedp.EmulateViewport(375, 812, chromedp.EmulateMobile),
		chromedp.Navigate(targetURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Title(&title),
		chromedp.OuterHTML(`html`, &content, chromedp.ByQuery),
		chromedp.Evaluate(`document.documentElement.scrollHeight`, &pageHeight),
		chromedp.ActionFunc(func(ctx context.Context) error {
			h := int64(pageHeight)
			if h > 5000 {
				h = 5000
			}
			if h < 812 {
				h = 812
			}
			return chromedp.EmulateViewport(375, h, chromedp.EmulateMobile).Do(ctx)
		}),
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
