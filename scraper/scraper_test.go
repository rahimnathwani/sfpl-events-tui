package scraper

import (
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
	// Must be sorted by next date ascending (Chess Club +24h before Anza Book Club +72h)
	if groups[0].Name != "Chess Club" || groups[1].Name != "Anza Book Club" {
		t.Errorf("groups not sorted by next date: got %v, %v", groups[0].Name, groups[1].Name)
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
