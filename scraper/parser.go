package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/net/html"

	"sfpl-events-tui/model"
)

const baseURL = "https://sfpl.org"

type ajaxCommand struct {
	Command string `json:"command"`
	Data    string `json:"data"`
}

// ParseResponse parses a Drupal AJAX JSON response body and returns all event
// instances found in the embedded HTML.
func ParseResponse(body []byte) ([]model.EventInstance, error) {
	var commands []ajaxCommand
	if err := json.Unmarshal(body, &commands); err != nil {
		return nil, err
	}
	for _, cmd := range commands {
		if cmd.Command == "insert" {
			return parseHTML(cmd.Data)
		}
	}
	return nil, nil
}

// ParseDescription parses an individual SFPL event page and returns the plain
// text body description.
func ParseDescription(r io.Reader) string {
	doc, err := html.Parse(r)
	if err != nil {
		return ""
	}
	node := findNode(doc, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "field--name-body")
	})
	if node == nil {
		return ""
	}
	return strings.TrimSpace(textContent(node))
}

func parseHTML(htmlStr string) ([]model.EventInstance, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}
	var instances []model.EventInstance
	walkNodes(doc, func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "article" && hasClass(n, "event--teaser") {
			if inst := extractInstance(n); inst != nil {
				instances = append(instances, *inst)
			}
			return false // don't descend into nested articles
		}
		return true
	})
	return instances, nil
}

func extractInstance(article *html.Node) *model.EventInstance {
	about := attr(article, "about")
	if about == "" {
		return nil
	}

	titleNode := findNode(article, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "h2" && hasClass(n, "event__title")
	})
	dateNode := findNode(article, func(n *html.Node) bool {
		return n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "date-display-range")
	})
	locNode := findNode(article, func(n *html.Node) bool {
		return n.Type == html.ElementNode && hasClass(n, "event__location")
	})

	if titleNode == nil || dateNode == nil {
		return nil
	}

	name := strings.TrimSpace(textContent(titleNode))
	dateStr := strings.TrimSpace(textContent(dateNode))
	location := ""
	if locNode != nil {
		a := findNode(locNode, func(n *html.Node) bool {
			return n.Type == html.ElementNode && n.Data == "a"
		})
		if a != nil {
			location = strings.TrimSpace(textContent(a))
		}
	}

	start, end, err := parseDateRange(dateStr)
	if err != nil {
		return nil
	}

	return &model.EventInstance{
		Name:     name,
		Date:     start,
		EndTime:  end,
		Location: location,
		URL:      baseURL + about,
	}
}

// parseDateRange parses "Wednesday, 5/13/2026, 10:00 - 11:00" into start and end times.
func parseDateRange(s string) (start, end time.Time, err error) {
	parts := strings.SplitN(s, ", ", 3)
	if len(parts) < 3 {
		return time.Time{}, time.Time{}, fmt.Errorf("unexpected date format: %q", s)
	}
	dateStr := parts[1]   // "5/13/2026"
	timeRange := parts[2] // "10:00 - 11:00"

	timeParts := strings.SplitN(timeRange, " - ", 2)
	start, err = time.ParseInLocation("1/2/2006 15:04", dateStr+" "+strings.TrimSpace(timeParts[0]), time.Local)
	if err != nil {
		return
	}
	if len(timeParts) == 2 {
		end, _ = time.ParseInLocation("1/2/2006 15:04", dateStr+" "+strings.TrimSpace(timeParts[1]), time.Local)
	}
	return
}

// --- HTML helpers ---

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	for _, c := range strings.Fields(attr(n, "class")) {
		if c == class {
			return true
		}
	}
	return false
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	walkNodes(n, func(n *html.Node) bool {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		return true
	})
	return sb.String()
}

// findNode returns the first node for which match returns true (depth-first).
func findNode(n *html.Node, match func(*html.Node) bool) *html.Node {
	if match(n) {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, match); found != nil {
			return found
		}
	}
	return nil
}

// walkNodes calls fn for every node. If fn returns false the node's children
// are skipped.
func walkNodes(n *html.Node, fn func(*html.Node) bool) {
	if !fn(n) {
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkNodes(c, fn)
	}
}
