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
