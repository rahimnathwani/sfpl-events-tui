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
