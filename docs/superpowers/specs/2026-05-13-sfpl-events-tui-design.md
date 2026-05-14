# SFPL Events TUI — Design Spec

## Overview

A terminal user interface (TUI) for browsing San Francisco Public Library events. Written in Go using the Bubbletea framework. On every launch it fetches fresh event data from the SFPL website, groups events by name, and lets the user archive event names they're not interested in.

---

## Data Model

```go
type EventInstance struct {
    Name        string
    Date        time.Time
    Location    string
    URL         string
    Description *string  // nil = not yet fetched; "" = fetched but empty
}

type EventGroup struct {
    Name      string
    Instances []EventInstance  // sorted ascending by date
    Archived  bool
}
```

**Grouping key:** exact event title string. Two instances with the same title (different dates or locations) collapse into one `EventGroup`.

**Next date:** `EventGroup.Instances[0].Date` — the earliest upcoming occurrence.

Archive state is held in memory as `map[string]bool`, keyed by event name. Loaded from disk at startup, written on every toggle.

---

## Scraping & Startup

### Source

SFPL's Drupal AJAX endpoint:
```
https://sfpl.org/views/ajax?_wrapper_format=drupal_ajax&items_per_page=50
  &view_name=events&view_display_id=page_events_list
  &date-end-after=<MM/DD/YYYY>&field_event_location_target_id=<ID>
  &_drupal_ajax=1&...
```

### Locations (33 total)

| ID  | Name                              | ID   | Name                    |
|-----|-----------------------------------|------|-------------------------|
| 31  | Main Library                      | 47   | Park                    |
| 41  | Anza                              | 55   | Parkside                |
| 38  | Bayview/Linda Brooks-Burton       | 58   | Portola                 |
| 53  | Bernal Heights                    | 56   | Potrero                 |
| 43  | Chinatown/Him Mark Lai            | 59   | Presidio                |
| 54  | Eureka Valley/Harvey Milk Memorial| 57   | Richmond/Senator Milton Marks |
| 44  | Excelsior                         | 48   | Sunset                  |
| 45  | Glen Park                         | 49   | Visitacion Valley       |
| 39  | Golden Gate Valley                | 50   | West Portal             |
| 40  | Ingleside                         | 52   | Western Addition        |
| 35  | Marina                            | 60   | Bookmobiles/Mobile Outreach |
| 36  | Merced                            | 667  | Virtual Library         |
| 34  | Mission                           | 915  | Treasure Island kiosk   |
| 32  | Mission Bay                       | 1019 | Dogpatch kiosk          |
| 33  | Noe Valley/Sally Brunn            | 1021 | Sunnydale kiosk         |
| 37  | North Beach                       | 46   | Ortega                  |
| 42  | Ocean View                        |      |                         |

### Date windows

6 monthly windows from today forward (e.g. 05/13/2026, 06/13/2026, … 10/13/2026). Dates are capped at day 28 to avoid month-end edge cases. Total requests: ~200 (33 locations × 6 dates).

### Execution

- All requests fire in parallel, capped at 20 concurrent goroutines (semaphore).
- Each response is a Drupal AJAX JSON envelope; the `insert` command contains HTML with event teasers.
- Parsed fields per instance: title (`h2.event__title a span`), date (`span.date-display-range`), location (`div.event__location a`), URL (`article[about]`).
- Deduplication by URL across all responses.
- Results grouped into `EventGroup`s, sorted alphabetically by name.

### Startup sequence

1. Read `~/.local/share/sfpl-events-tui/archived.json` → populate archive map.
2. Show loading screen; fire all scrape requests.
3. Each completed request sends a progress message to the Bubbletea update loop.
4. When all requests finish, transition to the main list view.

---

## Archive Persistence

**Path:** `~/.local/share/sfpl-events-tui/archived.json`

**Format:** JSON array of event name strings.
```json
["Chess Club", "Learn Mandarin", "Storytime: For Babies"]
```

- Directory created on first run if absent.
- Loaded into `map[string]bool` at startup.
- Full file rewrite on every `e` toggle (file is small).
- No other state is persisted; event data is always fetched fresh.

---

## TUI Structure

Three Bubbletea models, each implementing `tea.Model`:

| Model | File | Purpose |
|---|---|---|
| `LoadingModel` | `tui/loading.go` | Startup progress counter |
| `ListModel` | `tui/list.go` | Main tabbed event list |
| `DetailModel` | `tui/detail.go` | Per-event-name instance drill-down |

### Loading screen

```
  Fetching SFPL events: 142 / 180
```

Transitions automatically to `ListModel` when all requests complete.

### Main list

```
  [ Events ]  [ All Events ]

  Book Club (Adults)          May 15
▶ Chess Club                  May 16
  Coding for Kids             May 17

  [e] archive   [tab] switch tab   [enter] view dates
```

- **Events tab:** non-archived groups only, sorted alphabetically.
- **All Events tab:** all groups, archived ones visually dimmed, sorted alphabetically.
- `Tab` switches between tabs.
- `↑`/`↓` navigate the list.
- `e` toggles archive on the selected group; immediately removes from Events tab if newly archived (or restores if unarchived). Writes to disk.
- `Enter` transitions to `DetailModel` for the selected group.

### Detail view

```
  ◀ Chess Club

  May 16  10:00–11:00  Mission Bay
▶ May 23  10:00–11:00  Chinatown
  May 30  10:00–11:00  Mission Bay

  ──────────────────────────────────
  Loading description…
  ──────────────────────────────────

  [enter] open in browser   [esc] back
```

- Transition from main list is instant — no waiting.
- First instance is auto-selected on entry.
- Description for the selected instance is fetched as a background Bubbletea command immediately on entry and on each selection change.
- Fetched descriptions are cached on `EventInstance.Description`; re-selecting a cached instance does not re-fetch.
- `Enter` opens the instance URL using `open` (macOS) / `xdg-open` (Linux).
- `Esc` returns to the main list, restoring the previous scroll position.

---

## Package Structure

```
sfpl-events-tui/
├── main.go                  # Entry point, wires together scraper + TUI
├── scraper/
│   ├── scraper.go           # Parallel HTTP fetch, semaphore, progress reporting
│   └── parser.go            # HTML parsing of SFPL AJAX responses
├── model/
│   └── model.go             # EventInstance, EventGroup types
├── store/
│   └── store.go             # Archive load/save
└── tui/
    ├── tui.go               # Top-level Bubbletea app, model switching
    ├── loading.go           # Loading screen model
    ├── list.go              # Main list model
    └── detail.go            # Detail/drill-down model
```

---

## Key Decisions

- **No caching of event data.** Fresh fetch on every launch per the requirement.
- **Description lazy-loaded per instance.** The detail screen is instant; descriptions appear once the individual event page responds.
- **Grouping is exact title match.** No fuzzy matching — "Chess Club" and "Chess Club (Adults)" are separate groups.
- **URL deduplication.** The same event can appear in multiple location+date query responses; the URL is the canonical identity.
- **macOS `open` for browser.** The target platform is macOS (Darwin); `open <url>` is the standard invocation.
