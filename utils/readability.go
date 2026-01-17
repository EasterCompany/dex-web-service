package utils

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ReadabilityConfig holds settings for the extraction
type ReadabilityConfig struct {
	MinTextLength int
}

// ExtractMainContent analyzes the HTML doc and returns the main article content as Markdown
func ExtractMainContent(doc *html.Node, pageURL string) (string, error) {
	// 1. Clean the DOM (remove scripts, styles, navs, footers, etc.)
	cleanDOM(doc)

	// 2. Score candidates

	candidates := make(map[*html.Node]float64)
	scoreNode(doc, candidates)

	// 3. Find the winner
	var topNode *html.Node
	var topScore float64

	for node, score := range candidates {
		if score > topScore {
			topScore = score
			topNode = node
		}
	}

	// Fallback: if no winner found (e.g. really flat structure), use body
	if topNode == nil {
		topNode = findBody(doc)
	}
	if topNode == nil {
		return "", fmt.Errorf("could not find content body")
	}

	// 4. Convert top node to Markdown
	markdown := nodeToMarkdown(topNode, pageURL)
	return cleanMarkdown(markdown), nil
}

// cleanDOM removes noise tags and comments
func cleanDOM(n *html.Node) {
	// Tags to aggressively strip
	noisyTags := map[string]bool{
		"script": true, "style": true, "svg": true, "form": true,
		"nav": true, "footer": true, "header": true, "aside": true,
		"noscript": true, "iframe": true, "button": true, "input": true,
		"textarea": true, "select": true, "option": true,
	}

	var toRemove []*html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if noisyTags[n.Data] {
				toRemove = append(toRemove, n)
				return // Don't traverse children of removed nodes
			}

			// Check classes/IDs for noise
			class := getAttr(n, "class")
			id := getAttr(n, "id")
			combined := strings.ToLower(class + " " + id)

			// Simple heuristic filters
			if strings.Contains(combined, "sidebar") ||
				strings.Contains(combined, "comment") ||
				strings.Contains(combined, "popup") ||
				strings.Contains(combined, "cookie") ||
				strings.Contains(combined, "ad-") ||
				strings.Contains(combined, "widget") ||
				strings.Contains(combined, "promo") ||
				strings.Contains(combined, "newsletter") {
				toRemove = append(toRemove, n)
				return
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}

	walk(n)

	// Remove collected nodes
	for _, node := range toRemove {
		if node.Parent != nil {
			node.Parent.RemoveChild(node)
		}
	}
}

// scoreNode traverses and assigns points to potential containers
func scoreNode(n *html.Node, candidates map[*html.Node]float64) {
	if n.Type == html.ElementNode {
		// Only score block-level containers that contain text
		if n.Data == "p" || n.Data == "article" || n.Data == "section" || n.Data == "div" || n.Data == "main" {
			text := getTextContent(n)
			wordCount := len(strings.Fields(text))

			if wordCount < 10 {
				return // Too short to be main content
			}

			score := float64(0)

			// Base points for structure
			switch n.Data {
			case "article":
				score += 20
			case "main":
				score += 15
			case "section":
				score += 5
			}

			// Points for content volume
			score += float64(wordCount) * 0.5

			// Points for commas (indicative of prose)
			score += float64(strings.Count(text, ",")) * 1.5

			// Adjust based on class/id hints
			class := getAttr(n, "class")
			id := getAttr(n, "id")
			hints := strings.ToLower(class + " " + id)

			if strings.Contains(hints, "article") || strings.Contains(hints, "content") || strings.Contains(hints, "post") || strings.Contains(hints, "body") {
				score += 10
			}
			if strings.Contains(hints, "nav") || strings.Contains(hints, "menu") {
				score -= 20
			}

			// Assign score to this node AND bubble up some score to parent
			candidates[n] += score
			if n.Parent != nil {
				candidates[n.Parent] += score * 0.3 // Parent gets partial credit
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		scoreNode(c, candidates)
	}
}

// nodeToMarkdown converts the DOM subtree to Markdown
func nodeToMarkdown(n *html.Node, baseURL string) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				sb.WriteString(text + " ")
			}
			return
		}

		if node.Type == html.ElementNode {
			switch node.Data {
			case "h1":
				sb.WriteString("\n# ")
			case "h2":
				sb.WriteString("\n## ")
			case "h3":
				sb.WriteString("\n### ")
			case "p":
				sb.WriteString("\n\n")
			case "br":
				sb.WriteString("\n")
			case "li":
				sb.WriteString("\n- ")
			case "a":
				href := getAttr(node, "href")
				if href != "" {
					// Resolve relative URLs
					if baseURL != "" {
						u, err := url.Parse(href)
						if err == nil {
							base, err := url.Parse(baseURL)
							if err == nil {
								href = base.ResolveReference(u).String()
							}
						}
					}
					sb.WriteString(" [") // Start link
					for c := node.FirstChild; c != nil; c = c.NextSibling {
						walk(c) // Recurse for link text
					}
					sb.WriteString(fmt.Sprintf("](%s) ", href)) // End link
					return                                      // Don't traverse children again
				}
			}
		}

		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}

		// Post-element formatting
		if node.Type == html.ElementNode {
			if node.Data == "p" || node.Data == "div" || node.Data == "h1" || node.Data == "h2" || node.Data == "h3" {
				sb.WriteString("\n")
			}
		}
	}

	walk(n)
	return sb.String()
}

func findBody(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "body" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		found := findBody(c)
		if found != nil {
			return found
		}
	}
	return nil
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func getTextContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data + " ")
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}

func cleanMarkdown(raw string) string {
	// Collapse multiple newlines
	re := regexp.MustCompile(`\n{3,}`)
	clean := re.ReplaceAllString(raw, "\n\n")

	// Collapse multiple spaces
	reSpaces := regexp.MustCompile(`[ 	]+`)
	clean = reSpaces.ReplaceAllString(clean, " ")

	return strings.TrimSpace(clean)
}
