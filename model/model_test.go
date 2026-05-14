package model_test

import (
	"testing"
	"time"

	"sfpl-events-tui/model"
)

func TestNextDate_returnsEarliestInstance(t *testing.T) {
	now := time.Now()
	instances := []model.EventInstance{
		{Date: now.Add(48 * time.Hour)},
		{Date: now.Add(24 * time.Hour)},
	}
	model.SortInstances(instances)
	g := model.EventGroup{
		Name:      "Chess Club",
		Instances: instances,
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
	if !instances[0].Date.Equal(now.Add(24 * time.Hour)) ||
		!instances[1].Date.Equal(now.Add(48 * time.Hour)) ||
		!instances[2].Date.Equal(now.Add(72 * time.Hour)) {
		t.Error("SortInstances did not sort all elements correctly")
	}
}
