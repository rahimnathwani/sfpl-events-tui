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
	earliest := g.Instances[0].Date
	for _, inst := range g.Instances[1:] {
		if inst.Date.Before(earliest) {
			earliest = inst.Date
		}
	}
	return earliest
}

func SortInstances(instances []EventInstance) {
	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Date.Before(instances[j].Date)
	})
}
