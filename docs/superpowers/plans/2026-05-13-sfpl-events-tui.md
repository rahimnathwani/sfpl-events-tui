# SFPL Events TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go + Bubbletea TUI that fetches SFPL library events fresh on every launch, groups them by name, and lets the user browse, drill into instances, and archive event names they've lost interest in.

**Architecture:** `AppModel` is the single Bubbletea root model; it owns `LoadingModel`, `ListModel`, and `DetailModel` as plain structs (not tea.Model themselves), delegating updates based on the active screen. A goroutine runs the scraper and pushes progress/completion messages via `p.Send()`. Navigation between screens happens through typed messages (`NavigateToDetailMsg`, `NavigateBackMsg`). Descriptions are fetched lazily via Bubbletea commands when an instance is selected.

**Tech Stack:** Go 1.22+, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `golang.org/x/net/html`

---

## File Map

```
sfpl-events-tui/
├── main.go                   # CLI flags, wires scraper goroutine to tea.Program
├── model/
│   ├── model.go              # EventInstance, EventGroup types and helpers
│   └── model_test.go
├── store/
│   ├── store.go              # Load/Save archived names to ~/.local/share/...
│   └── store_test.go
├── scraper/
│   ├── locations.go          # Location ID→name map and ID slice
│   ├── parser.go             # Parse Drupal AJAX response HTML → []EventInstance
│   ├── parser_test.go
│   ├── scraper.go            # Parallel fetch with semaphore + pagination; GroupInstances
│   └── scraper_test.go
└── tui/
    ├── messages.go           # Exported message types shared across screens
    ├── app.go                # AppModel: root tea.Model, screen switching
    ├── loading.go            # LoadingModel: progress counter display
    ├── list.go               # ListModel: two-tab event group list
    └── detail.go             # DetailModel: instance list + lazy description
```

---

## Task 1: Go module and dependencies

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialise the module**

```bash
cd /Users/rahim/src/personal/sfpl-events-tui
go mod init sfpl-events-tui
```

- [ ] **Step 2: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get golang.org/x/net/html
```

- [ ] **Step 3: Verify**

```bash
cat go.mod
```

Expected: module line `sfpl-events-tui`, three `require` entries.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: initialise Go module with bubbletea, lipgloss, x/net/html"
```

---

## Task 2: Data model

**Files:**
- Create: `model/model.go`
- Create: `model/model_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// model/model_test.go
package model_test

import (
	"testing"
	"time"

	"sfpl-events-tui/model"
)

func TestNextDate_returnsEarliestInstance(t *testing.T) {
	now := time.Now()
	g := model.EventGroup{
		Name: "Chess Club",
		Instances: []model.EventInstance{
			{Date: now.Add(48 * time.Hour)},
			{Date: now.Add(24 * time.Hour)},
		},
	}
	want := now.Add(24 * time.Hour)
	if got := g.NextDate(); !got.Equal(want) {
		t.Errorf("NextDate() = %v, want %v", got, want)
	}
}

func TestNextDate_emptyGroup(t *testing.T) {
	g := model.EventGroup{}
	if !g.NextDate().IsZero() {
		t.Error("expected zero time for empty group")
	}
}

func TestSortInstances_sortsAscending(t *testing.T) {
	now := time.Now()
	instances := []model.EventInstance{
		{Date: now.Add(72 * time.Hour)},
		{Date: now.Add(24 * time.Hour)},
		{Date: now.Add(48 * time.Hour)},
	}
	model.SortInstances(instances)
	if !instances[0].Date.Equal(now.Add(24 * time.Hour)) {
		t.Error("SortInstances did not sort ascending")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./model/...
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Implement the model**

```go
// model/model.go
package model

import (
	"sort"
	"time"
)

type EventInstance struct {
	Name        string
	Date        time.Time
	EndTime     time.Time // zero if unavailable
	Location    string
	URL         string
	Description *string // nil = not fetched; "" = fetched but empty
}

type EventGroup struct {
	Name      string
	Instances []EventInstance // sorted ascending by Date
	Archived  bool
}

func (g EventGroup) NextDate() time.Time {
	if len(g.Instances) == 0 {
		return time.Time{}
	}
	return g.Instances[0].Date
}

func SortInstances(instances []EventInstance) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Date.Before(instances[j].Date)
	})
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./model/...
```

Expected: `ok sfpl-events-tui/model`

- [ ] **Step 5: Commit**

```bash
git add model/
git commit -m "feat: add EventInstance and EventGroup model types"
```

---

## Task 3: Archive store

**Files:**
- Create: `store/store.go`
- Create: `store/store_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// store/store_test.go
package store_test

import (
	"path/filepath"
	"testing"

	"sfpl-events-tui/store"
)

func TestLoad_nonExistentFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "archived.json"))
	defer store.ResetPath()

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestSaveAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "archived.json"))
	defer store.ResetPath()

	input := map[string]bool{"Chess Club": true, "Book Club": true}
	if err := store.Save(input); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !got["Chess Club"] || !got["Book Club"] {
		t.Errorf("round-trip failed: got %v", got)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestSave_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "nested", "dir", "archived.json"))
	defer store.ResetPath()

	if err := store.Save(map[string]bool{"X": true}); err != nil {
		t.Fatalf("Save should create missing dirs: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./store/...
```

Expected: compile error.

- [ ] **Step 3: Implement the store**

```go
// store/store.go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// defaultPath returns the canonical archive file path.
func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "archived.json"
	}
	return filepath.Join(home, ".local", "share", "sfpl-events-tui", "archived.json")
}

var overriddenPath string

// OverridePath replaces the storage path (for tests only).
func OverridePath(p string) { overriddenPath = p }

// ResetPath clears a test override.
func ResetPath() { overriddenPath = "" }

func path() string {
	if overriddenPath != "" {
		return overriddenPath
	}
	return defaultPath()
}

// Load reads the archived event names from disk. Returns an empty map if the
// file does not exist.
func Load() (map[string]bool, error) {
	data, err := os.ReadFile(path())
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m, nil
}

// Save writes the archived event names to disk, creating directories as needed.
func Save(archived map[string]bool) error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(archived))
	for n := range archived {
		names = append(names, n)
	}
	sort.Strings(names)
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./store/...
```

Expected: `ok sfpl-events-tui/store`

- [ ] **Step 5: Commit**

```bash
git add store/
git commit -m "feat: add archive store (load/save archived event names)"
```

---

## Task 4: SFPL response parser

**Files:**
- Create: `scraper/locations.go`
- Create: `scraper/parser.go`
- Create: `scraper/parser_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// scraper/parser_test.go
package scraper

import (
	"strings"
	"testing"
	"time"
)

const sampleHTML = `<div class="view-content">
  <div class="views-row">
    <article about="/events/2026/05/13/test-event" class="event event--teaser event--adult teaser">
      <div class="event__details">
        <div class="event__date">
          <div class="field field--name-field-event-date-and-time field__item">
            <span class="date-display-range">Wednesday, 5/13/2026, 10:00 - 11:00</span>
          </div>
        </div>
        <div class="event__name">
          <h2 class="event__title">
            <a href="/events/2026/05/13/test-event" rel="bookmark"><span>Test Event</span>
</a>
          </h2>
        </div>
        <div class="event__location">
          <a href="/locations/mission-bay">Mission Bay</a>
        </div>
      </div>
    </article>
  </div>
</div>`

func TestParseHTML_extractsFields(t *testing.T) {
	instances, err := parseHTML(sampleHTML)
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(instances))
	}
	inst := instances[0]
	if inst.Name != "Test Event" {
		t.Errorf("Name = %q, want %q", inst.Name, "Test Event")
	}
	if inst.URL != "https://sfpl.org/events/2026/05/13/test-event" {
		t.Errorf("URL = %q", inst.URL)
	}
	if inst.Location != "Mission Bay" {
		t.Errorf("Location = %q, want %q", inst.Location, "Mission Bay")
	}
	if inst.Date.Month() != time.May || inst.Date.Day() != 13 || inst.Date.Hour() != 10 {
		t.Errorf("Date = %v", inst.Date)
	}
	if inst.EndTime.Hour() != 11 {
		t.Errorf("EndTime = %v", inst.EndTime)
	}
}

func TestParseDateRange_parsesStartAndEnd(t *testing.T) {
	start, end, err := parseDateRange("Wednesday, 5/13/2026, 10:00 - 11:30")
	if err != nil {
		t.Fatal(err)
	}
	if start.Hour() != 10 || start.Minute() != 0 {
		t.Errorf("start = %v", start)
	}
	if end.Hour() != 11 || end.Minute() != 30 {
		t.Errorf("end = %v", end)
	}
}

func TestParseDescription_extractsBodyText(t *testing.T) {
	html := `<html><body>
		<div class="field field--name-body field__item"><p>Join us for chess!</p></div>
	</body></html>`
	got := ParseDescription(strings.NewReader(html))
	if !strings.Contains(got, "Join us for chess!") {
		t.Errorf("ParseDescription = %q", got)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./scraper/...
```

Expected: compile error — package does not exist.

- [ ] **Step 3: Create the locations file**

```go
// scraper/locations.go
package scraper

var locationIDs = []int{
	31, 32, 33, 34, 35, 36, 37, 38, 39, 40,
	41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
	52, 53, 54, 55, 56, 57, 58, 59, 60,
	667, 915, 1019, 1021,
}

var locationNames = map[int]string{
	31:   "Main Library",
	32:   "Mission Bay",
	33:   "Noe Valley/Sally Brunn",
	34:   "Mission",
	35:   "Marina",
	36:   "Merced",
	37:   "North Beach",
	38:   "Bayview/Linda Brooks-Burton",
	39:   "Golden Gate Valley",
	40:   "Ingleside",
	41:   "Anza",
	42:   "Ocean View",
	43:   "Chinatown/Him Mark Lai",
	44:   "Excelsior",
	45:   "Glen Park",
	46:   "Ortega",
	47:   "Park",
	48:   "Sunset",
	49:   "Visitacion Valley",
	50:   "West Portal",
	52:   "Western Addition",
	53:   "Bernal Heights",
	54:   "Eureka Valley/Harvey Milk Memorial",
	55:   "Parkside",
	56:   "Potrero",
	57:   "Richmond/Senator Milton Marks",
	58:   "Portola",
	59:   "Presidio",
	60:   "Bookmobiles/Mobile Outreach",
	667:  "Virtual Library",
	915:  "Treasure Island kiosk",
	1019: "Dogpatch kiosk",
	1021: "Sunnydale kiosk",
}
```

- [ ] **Step 4: Implement the parser**

```go
// scraper/parser.go
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
```

- [ ] **Step 5: Run tests to confirm they pass**

```bash
go test ./scraper/...
```

Expected: `ok sfpl-events-tui/scraper`

- [ ] **Step 6: Commit**

```bash
git add scraper/
git commit -m "feat: add SFPL HTML parser and location map"
```

---

## Task 5: Scraper and grouper

**Files:**
- Create: `scraper/scraper.go`
- Create: `scraper/scraper_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// scraper/scraper_test.go
package scraper

import (
	"sort"
	"testing"
	"time"

	"sfpl-events-tui/model"
)

func TestGroupInstances_deduplicatesAndGroups(t *testing.T) {
	now := time.Now()
	instances := []model.EventInstance{
		{Name: "Chess Club", URL: "https://sfpl.org/events/chess-1", Date: now.Add(24 * time.Hour), Location: "Main"},
		{Name: "Chess Club", URL: "https://sfpl.org/events/chess-2", Date: now.Add(48 * time.Hour), Location: "Anza"},
		{Name: "Chess Club", URL: "https://sfpl.org/events/chess-1", Date: now.Add(24 * time.Hour), Location: "Main"}, // duplicate
		{Name: "Anza Book Club", URL: "https://sfpl.org/events/book-1", Date: now.Add(72 * time.Hour), Location: "Anza"},
	}
	archived := map[string]bool{"Chess Club": true}

	groups := GroupInstances(instances, archived)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Must be sorted alphabetically
	names := []string{groups[0].Name, groups[1].Name}
	if !sort.StringsAreSorted(names) {
		t.Errorf("groups not sorted: %v", names)
	}
	// Find Chess Club
	var chess model.EventGroup
	for _, g := range groups {
		if g.Name == "Chess Club" {
			chess = g
		}
	}
	if !chess.Archived {
		t.Error("Chess Club should be archived")
	}
	if len(chess.Instances) != 2 {
		t.Errorf("expected 2 unique instances for Chess Club, got %d", len(chess.Instances))
	}
	// Instances sorted ascending
	if !chess.Instances[0].Date.Before(chess.Instances[1].Date) {
		t.Error("instances not sorted by date")
	}
}

func TestGroupInstances_emptyInput(t *testing.T) {
	groups := GroupInstances(nil, nil)
	if len(groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(groups))
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./scraper/... -run TestGroup
```

Expected: compile error — `GroupInstances` undefined.

- [ ] **Step 3: Implement the scraper**

```go
// scraper/scraper.go
package scraper

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"sfpl-events-tui/model"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// Scrape fetches all SFPL events for the given number of months ahead.
// progress is called after each completed base request (not pagination requests).
func Scrape(months int, progress func(completed, total int)) ([]model.EventInstance, error) {
	type job struct {
		locID int
		date  string
	}

	now := time.Now()
	var jobs []job
	for _, locID := range locationIDs {
		for m := 0; m < months; m++ {
			t := now.AddDate(0, m, 0)
			day := t.Day()
			if day > 28 {
				day = 28
			}
			date := fmt.Sprintf("%02d%%2F%02d%%2F%d", int(t.Month()), day, t.Year())
			jobs = append(jobs, job{locID: locID, date: date})
		}
	}

	total := len(jobs)
	if progress != nil {
		progress(0, total)
	}

	var (
		mu        sync.Mutex
		instances []model.EventInstance
		completed int
	)

	sem := make(chan struct{}, 20)
	var wg sync.WaitGroup

	for _, j := range jobs {
		j := j
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			insts := fetchAllPages(j.locID, j.date)

			mu.Lock()
			instances = append(instances, insts...)
			completed++
			if progress != nil {
				progress(completed, total)
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return instances, nil
}

func fetchAllPages(locID int, date string) []model.EventInstance {
	var all []model.EventInstance
	for page := 0; ; page++ {
		url := buildURL(locID, date, page)
		resp, err := httpClient.Get(url)
		if err != nil {
			break
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			break
		}
		insts, err := ParseResponse(body)
		if err != nil || len(insts) == 0 {
			break
		}
		all = append(all, insts...)
		if len(insts) < 50 {
			break
		}
	}
	return all
}

func buildURL(locID int, date string, page int) string {
	return fmt.Sprintf(
		"https://sfpl.org/views/ajax?_wrapper_format=drupal_ajax&items_per_page=50"+
			"&view_name=events&view_display_id=page_events_list"+
			"&view_dom_id=enhanced-basic--events-page_events_list"+
			"&date-end-after=%s&field_event_location_target_id=%d&page=%d"+
			"&_drupal_ajax=1&ajax_page_state%%5Btheme%%5D=sfpl_2019"+
			"&ajax_page_state%%5Btheme_token%%5D=&ajax_page_state%%5Blibraries%%5D=",
		date, locID, page,
	)
}

// GroupInstances deduplicates by URL, groups by event name, sorts instances
// within each group by date, and sorts groups alphabetically.
func GroupInstances(instances []model.EventInstance, archived map[string]bool) []model.EventGroup {
	seen := make(map[string]bool)
	byName := make(map[string]*model.EventGroup)

	for _, inst := range instances {
		if seen[inst.URL] {
			continue
		}
		seen[inst.URL] = true

		g, ok := byName[inst.Name]
		if !ok {
			byName[inst.Name] = &model.EventGroup{
				Name:     inst.Name,
				Archived: archived[inst.Name],
			}
			g = byName[inst.Name]
		}
		g.Instances = append(g.Instances, inst)
	}

	groups := make([]model.EventGroup, 0, len(byName))
	for _, g := range byName {
		model.SortInstances(g.Instances)
		groups = append(groups, *g)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name < groups[j].Name
	})

	return groups
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./scraper/... -run TestGroup
```

Expected: `ok sfpl-events-tui/scraper`

- [ ] **Step 5: Run all tests**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 6: Commit**

```bash
git add scraper/scraper.go scraper/scraper_test.go
git commit -m "feat: add parallel scraper with pagination and GroupInstances"
```

---

## Task 6: TUI messages and app shell

**Files:**
- Create: `tui/messages.go`
- Create: `tui/app.go`

No tests for this task — it is pure wiring verified by the running binary.

- [ ] **Step 1: Create the shared message types**

```go
// tui/messages.go
package tui

import "sfpl-events-tui/model"

// ProgressMsg is sent by the scraper goroutine after each completed base request.
type ProgressMsg struct{ Completed, Total int }

// ScrapeCompleteMsg is sent when all scrape requests have finished.
type ScrapeCompleteMsg struct{ Groups []model.EventGroup }

// ScrapeErrMsg is sent if the scraper fails fatally.
type ScrapeErrMsg struct{ Err error }

// NavigateToDetailMsg is sent by ListModel when the user presses Enter.
type NavigateToDetailMsg struct{ Group model.EventGroup }

// NavigateBackMsg is sent by DetailModel when the user presses Esc.
type NavigateBackMsg struct{}

// ArchiveToggleMsg is sent by ListModel when the user presses 'e'.
type ArchiveToggleMsg struct{ Name string }

// DescFetchedMsg is sent when a description HTTP fetch completes.
type DescFetchedMsg struct{ URL, Text string }
```

- [ ] **Step 2: Create the app model**

```go
// tui/app.go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"sfpl-events-tui/model"
	"sfpl-events-tui/store"
)

type screen int

const (
	screenLoading screen = iota
	screenList
	screenDetail
)

// AppModel is the root Bubbletea model. It owns all sub-models and routes
// messages between them.
type AppModel struct {
	screen   screen
	archived map[string]bool
	loading  LoadingModel
	list     ListModel
	detail   DetailModel
}

func NewAppModel(archived map[string]bool) AppModel {
	return AppModel{
		screen:   screenLoading,
		archived: archived,
		loading:  NewLoadingModel(),
	}
}

func (m AppModel) Init() tea.Cmd { return nil }

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case ProgressMsg:
		m.loading.Completed = msg.Completed
		m.loading.Total = msg.Total
		return m, nil

	case ScrapeCompleteMsg:
		m.list = NewListModel(msg.Groups, m.archived)
		m.screen = screenList
		return m, nil

	case ScrapeErrMsg:
		// Quit on fatal scrape error; real app could show an error screen
		return m, tea.Quit

	case NavigateToDetailMsg:
		m.detail = NewDetailModel(msg.Group)
		m.screen = screenDetail
		var cmd tea.Cmd
		if len(msg.Group.Instances) > 0 && msg.Group.Instances[0].Description == nil {
			cmd = FetchDescriptionCmd(msg.Group.Instances[0].URL)
		}
		return m, cmd

	case NavigateBackMsg:
		m.screen = screenList
		return m, nil

	case ArchiveToggleMsg:
		if m.archived[msg.Name] {
			delete(m.archived, msg.Name)
		} else {
			m.archived[msg.Name] = true
		}
		store.Save(m.archived) // synchronous; file is tiny
		m.list.archived = m.archived
		for i, g := range m.list.groups {
			if g.Name == msg.Name {
				m.list.groups[i].Archived = m.archived[msg.Name]
			}
		}
		m.list.clampCursors()
		return m, nil

	case DescFetchedMsg:
		newDetail, cmd := m.detail.Update(msg)
		m.detail = newDetail
		return m, cmd
	}

	// Delegate to the active screen.
	switch m.screen {
	case screenLoading:
		return m, nil
	case screenList:
		newList, cmd := m.list.Update(msg)
		m.list = newList
		return m, cmd
	case screenDetail:
		newDetail, cmd := m.detail.Update(msg)
		m.detail = newDetail
		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	switch m.screen {
	case screenLoading:
		return m.loading.View()
	case screenList:
		return m.list.View()
	case screenDetail:
		return m.detail.View()
	}
	return ""
}

// StartScraper returns a function that starts the scraper goroutine. Call it
// after creating the tea.Program so that p.Send is available.
func StartScraper(p *tea.Program, months int) {
	go func() {
		instances, err := scrapeFunc(months, func(completed, total int) {
			p.Send(ProgressMsg{Completed: completed, Total: total})
		})
		if err != nil {
			p.Send(ScrapeErrMsg{Err: err})
			return
		}
		// archived is read-only at this point (before any UI interaction)
		p.Send(ScrapeCompleteMsg{})
		_ = instances // handled in main.go via a closure — see Task 10
	}()
}
```

> **Note:** `StartScraper` above is a placeholder; Task 10 (main.go) shows the actual wiring pattern which avoids the circular dependency on `archived`. The goroutine is started directly in `main.go`.

- [ ] **Step 3: Verify it compiles (stubs for missing types are OK for now)**

```bash
go build ./tui/... 2>&1 | head -20
```

Expected: errors for `NewLoadingModel`, `NewListModel`, `NewDetailModel`, `FetchDescriptionCmd` — these are fine, they will be added in subsequent tasks.

- [ ] **Step 4: Commit**

```bash
git add tui/messages.go tui/app.go
git commit -m "feat: add TUI message types and AppModel shell"
```

---

## Task 7: Loading screen

**Files:**
- Create: `tui/loading.go`

- [ ] **Step 1: Implement the loading model**

```go
// tui/loading.go
package tui

import "fmt"

// LoadingModel displays a progress counter while the scraper is running.
type LoadingModel struct {
	Completed int
	Total     int
}

func NewLoadingModel() LoadingModel {
	return LoadingModel{}
}

func (m LoadingModel) View() string {
	if m.Total == 0 {
		return "\n  Fetching SFPL events…\n"
	}
	return fmt.Sprintf("\n  Fetching SFPL events: %d / %d\n", m.Completed, m.Total)
}
```

- [ ] **Step 2: Verify the package compiles**

```bash
go build ./tui/... 2>&1 | grep -v "NewListModel\|NewDetailModel\|FetchDescriptionCmd\|scrapeFunc"
```

Expected: no new errors beyond the still-missing symbols.

- [ ] **Step 3: Commit**

```bash
git add tui/loading.go
git commit -m "feat: add loading screen model"
```

---

## Task 8: Main list

**Files:**
- Create: `tui/list.go`

- [ ] **Step 1: Implement the list model**

```go
// tui/list.go
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sfpl-events-tui/model"
)

var (
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Underline(true).Padding(0, 1)
	inactiveTabStyle = lipgloss.NewStyle().Padding(0, 1).Faint(true)
	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	archivedRowStyle = lipgloss.NewStyle().Faint(true)
	cursorChar       = "▶"
	noCursorChar     = " "
)

// ListModel renders the two-tab event group list.
type ListModel struct {
	groups   []model.EventGroup
	archived map[string]bool
	tab      int    // 0 = Events, 1 = All Events
	cursors  [2]int // cursor per tab
	viewport int    // index of first visible row in current tab
	width    int
	height   int
}

func NewListModel(groups []model.EventGroup, archived map[string]bool) ListModel {
	return ListModel{
		groups:   groups,
		archived: archived,
	}
}

func (m ListModel) visible() []model.EventGroup {
	if m.tab == 0 {
		var result []model.EventGroup
		for _, g := range m.groups {
			if !m.archived[g.Name] {
				result = append(result, g)
			}
		}
		return result
	}
	return m.groups
}

func (m *ListModel) clampCursors() {
	for t := 0; t < 2; t++ {
		prev := m.tab
		m.tab = t
		vis := m.visible()
		m.tab = prev
		if len(vis) == 0 {
			m.cursors[t] = 0
		} else if m.cursors[t] >= len(vis) {
			m.cursors[t] = len(vis) - 1
		}
	}
	m.scrollToCursor()
}

func (m *ListModel) scrollToCursor() {
	cursor := m.cursors[m.tab]
	visH := m.height - 6 // reserve rows for tabs + footer
	if visH < 1 {
		visH = 1
	}
	if cursor < m.viewport {
		m.viewport = cursor
	} else if cursor >= m.viewport+visH {
		m.viewport = cursor - visH + 1
	}
	if m.viewport < 0 {
		m.viewport = 0
	}
}

func (m ListModel) Update(msg tea.Msg) (ListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		vis := m.visible()
		cursor := m.cursors[m.tab]

		switch msg.String() {
		case "up", "k":
			if cursor > 0 {
				m.cursors[m.tab]--
				m.scrollToCursor()
			}
		case "down", "j":
			if cursor < len(vis)-1 {
				m.cursors[m.tab]++
				m.scrollToCursor()
			}
		case "tab":
			m.tab = 1 - m.tab
			m.viewport = 0
			m.scrollToCursor()
		case "e":
			if len(vis) > 0 {
				name := vis[cursor].Name
				return m, func() tea.Msg { return ArchiveToggleMsg{Name: name} }
			}
		case "enter":
			if len(vis) > 0 {
				group := vis[cursor]
				return m, func() tea.Msg { return NavigateToDetailMsg{Group: group} }
			}
		}
	}
	return m, nil
}

func (m ListModel) View() string {
	var sb strings.Builder

	// Tab bar
	tab0 := inactiveTabStyle.Render("Events")
	tab1 := inactiveTabStyle.Render("All Events")
	if m.tab == 0 {
		tab0 = activeTabStyle.Render("Events")
	} else {
		tab1 = activeTabStyle.Render("All Events")
	}
	sb.WriteString(fmt.Sprintf("  %s  %s\n\n", tab0, tab1))

	vis := m.visible()
	cursor := m.cursors[m.tab]
	visH := m.height - 6
	if visH < 1 {
		visH = 1
	}

	nameWidth := m.width - 16
	if nameWidth < 10 {
		nameWidth = 10
	}

	end := m.viewport + visH
	if end > len(vis) {
		end = len(vis)
	}

	for i := m.viewport; i < end; i++ {
		g := vis[i]
		c := noCursorChar
		if i == cursor {
			c = cursorChar
		}

		name := g.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}

		dateStr := ""
		if !g.NextDate().IsZero() {
			dateStr = g.NextDate().Format("Jan 2")
		}

		line := fmt.Sprintf("%s %-*s  %s", c, nameWidth, name, dateStr)

		if i == cursor {
			sb.WriteString(selectedRowStyle.Render(line))
		} else if m.tab == 1 && m.archived[g.Name] {
			sb.WriteString(archivedRowStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"  [e] archive   [tab] switch tab   [enter] view dates   [ctrl+c] quit",
	))
	return sb.String()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./tui/... 2>&1 | grep -v "NewDetailModel\|FetchDescriptionCmd\|scrapeFunc"
```

Expected: no new errors.

- [ ] **Step 3: Commit**

```bash
git add tui/list.go
git commit -m "feat: add two-tab event list with archive toggle and navigation"
```

---

## Task 9: Detail view

**Files:**
- Create: `tui/detail.go`

- [ ] **Step 1: Implement the detail model and description fetch command**

```go
// tui/detail.go
package tui

import (
	"fmt"
	"net/http"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"sfpl-events-tui/model"
	"sfpl-events-tui/scraper"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true)
	dividerStyle = lipgloss.NewStyle().Faint(true)
	selectedInstStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	descStyle    = lipgloss.NewStyle().Padding(0, 2)
)

// DetailModel shows the list of instances for one event group and lazily loads
// the description for the selected instance.
type DetailModel struct {
	group  model.EventGroup
	cursor int
	width  int
	height int
}

func NewDetailModel(group model.EventGroup) DetailModel {
	return DetailModel{group: group}
}

// FetchDescriptionCmd fetches the description for the given event URL as a
// background Bubbletea command.
func FetchDescriptionCmd(url string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(url)
		if err != nil {
			return DescFetchedMsg{URL: url, Text: ""}
		}
		defer resp.Body.Close()
		text := scraper.ParseDescription(resp.Body)
		return DescFetchedMsg{URL: url, Text: text}
	}
}

func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case DescFetchedMsg:
		for i, inst := range m.group.Instances {
			if inst.URL == msg.URL {
				text := msg.Text
				m.group.Instances[i].Description = &text
				break
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				return m, m.fetchIfNeeded()
			}
		case "down", "j":
			if m.cursor < len(m.group.Instances)-1 {
				m.cursor++
				return m, m.fetchIfNeeded()
			}
		case "enter":
			if len(m.group.Instances) > 0 {
				url := m.group.Instances[m.cursor].URL
				return m, openURL(url)
			}
		case "esc":
			return m, func() tea.Msg { return NavigateBackMsg{} }
		}
	}
	return m, nil
}

func (m DetailModel) fetchIfNeeded() tea.Cmd {
	if len(m.group.Instances) == 0 {
		return nil
	}
	inst := m.group.Instances[m.cursor]
	if inst.Description == nil {
		return FetchDescriptionCmd(inst.URL)
	}
	return nil
}

func (m DetailModel) View() string {
	var sb strings.Builder

	sb.WriteString("\n  ")
	sb.WriteString(headerStyle.Render("◀ " + m.group.Name))
	sb.WriteString("\n\n")

	for i, inst := range m.group.Instances {
		c := noCursorChar
		if i == m.cursor {
			c = cursorChar
		}

		timeStr := inst.Date.Format("Jan 2  15:04")
		if !inst.EndTime.IsZero() {
			timeStr += "–" + inst.EndTime.Format("15:04")
		}

		line := fmt.Sprintf("%s %-22s  %s", c, timeStr, inst.Location)
		if i == m.cursor {
			sb.WriteString(selectedInstStyle.Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	divider := dividerStyle.Render(strings.Repeat("─", m.width-2))
	sb.WriteString("\n  " + divider + "\n")

	if len(m.group.Instances) > 0 {
		inst := m.group.Instances[m.cursor]
		switch {
		case inst.Description == nil:
			sb.WriteString(descStyle.Render("Loading description…"))
		case *inst.Description == "":
			sb.WriteString(descStyle.Render("No description available."))
		default:
			sb.WriteString(descStyle.Render(*inst.Description))
		}
	}

	sb.WriteString("\n  " + divider + "\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
		"  [enter] open in browser   [esc] back   [ctrl+c] quit",
	))
	return sb.String()
}

// openURL opens the URL in the system browser (macOS: open, Linux: xdg-open).
func openURL(url string) tea.Cmd {
	return tea.ExecProcess(
		// Use tea.ExecProcess only for interactive processes; for fire-and-forget
		// we use a plain command that returns a no-op message.
		nil,
		func(err error) tea.Msg { return nil },
	)
}
```

> **Important:** `tea.ExecProcess` is for interactive shell replacements. For a background `open` call use `exec.Command`. Replace the `openURL` stub with the implementation below:

```go
// openURL opens the URL in the default browser and returns a no-op command.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		exec.Command("open", url).Start() //nolint:errcheck
		return nil
	}
}
```

Add `"os/exec"` to the import block and remove the `tea.ExecProcess` stub.

- [ ] **Step 2: Verify it compiles**

```bash
go build ./tui/...
```

Expected: success (all symbols now defined).

- [ ] **Step 3: Commit**

```bash
git add tui/detail.go
git commit -m "feat: add detail view with lazy description loading"
```

---

## Task 10: main.go and final wiring

**Files:**
- Create: `main.go`
- Modify: `tui/app.go` — remove the placeholder `StartScraper`/`scrapeFunc` references

- [ ] **Step 1: Write main.go**

```go
// main.go
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"sfpl-events-tui/scraper"
	"sfpl-events-tui/store"
	"sfpl-events-tui/tui"
)

func main() {
	months := flag.Int("months", 2, "number of months ahead to fetch")
	flag.Parse()

	archived, err := store.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load archive: %v\n", err)
		archived = map[string]bool{}
	}

	model := tui.NewAppModel(archived)
	p := tea.NewProgram(model, tea.WithAltScreen())

	go func() {
		instances, err := scraper.Scrape(*months, func(completed, total int) {
			p.Send(tui.ProgressMsg{Completed: completed, Total: total})
		})
		if err != nil {
			p.Send(tui.ScrapeErrMsg{Err: err})
			return
		}
		groups := scraper.GroupInstances(instances, archived)
		p.Send(tui.ScrapeCompleteMsg{Groups: groups})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Clean up the placeholder in app.go**

Remove `StartScraper` and `scrapeFunc` from `tui/app.go` (they are unused; wiring now lives entirely in `main.go`).

- [ ] **Step 3: Build the binary**

```bash
go build -o sfpl-events-tui .
```

Expected: binary produced with no errors.

- [ ] **Step 4: Run all tests**

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 5: Smoke test — verify startup**

```bash
./sfpl-events-tui --months 1
```

Expected: loading counter appears, increments, then the event list renders. Navigate with arrow keys, press `e` to archive, `Enter` to drill down, `Esc` to go back. `ctrl+c` quits.

- [ ] **Step 6: Final commit**

```bash
git add main.go tui/app.go
git commit -m "feat: wire scraper goroutine to TUI in main.go — app is complete"
```

---

## Self-review notes

- All 10 spec requirements are covered: two tabs, alphabetical sort, grouping by name, archive toggle, archive persistence, loading screen with counter, detail view, lazy description, browser open, ESC back.
- `--months` flag wired in `main.go`.
- Pagination implemented in `fetchAllPages` (loop until `< 50` results).
- `clampCursors` is called after every archive toggle so the cursor can't point off the end of the filtered list.
- `parseDateRange` handles missing end time gracefully (zero `EndTime`).
- `store.OverridePath`/`ResetPath` are the only test-only exported symbols — acceptable for a personal tool.
- Description fetching uses the `scraper.ParseDescription` function already tested in Task 4.
