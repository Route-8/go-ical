package go_ical

import (
	"fmt"
	"time"
)

type Event struct {
	UID         string
	Summary     string
	Description string
	// Categories  []string
	AllDay   bool
	Start    time.Time
	End      time.Time
	Duration time.Duration
	// Stamp            time.Time
	// Created          time.Time
	LastModified time.Time
	Location     string
	// URL              string
	Status       string
	IsRecurring  bool
	RecurrenceID string
	// ExcludeDates     []time.Time
	// Sequence         int
	// CustomAttributes map[string]string
}

func (e Event) GetID() string {
	return fmt.Sprintf("%v-%v", e.UID, e.RecurrenceID)
}
