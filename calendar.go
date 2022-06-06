package go_ical

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	ics "github.com/arran4/golang-ical"
	rrule "github.com/teambition/rrule-go"
)

const icalTimestampFormatUtc = "20060102T150405Z"

type ICalendar struct {
	Calendar  *ics.Calendar
	Events    map[string]Event
	StartTime time.Time
	EndTime   time.Time
	TimeZone  *time.Location
}

func NewICalendar(startTime time.Time, endTime time.Time, timezone *time.Location) ICalendar {
	return ICalendar{
		StartTime: startTime,
		EndTime:   endTime,
		TimeZone:  timezone,
	}
}

func (c *ICalendar) Parse(r io.Reader) (err error) {
	c.Calendar, err = ics.ParseCalendar(r)
	if err != nil {
		return
	}

	c.processEvents()
	return
}

func (c *ICalendar) processEvents() (err error) {
	c.Events = make(map[string]Event, len(c.Calendar.Events())*2)
	for i, evt := range c.Calendar.Events() {
		err := c.AddEventFromICS(evt)
		if err != nil {
			fmt.Printf("%v: Error processEvents = %v\n", i, err)
			continue
		}
	}
	return
}

func (c *ICalendar) AddEventFromICS(evt *ics.VEvent) (err error) {
	var allDay = false
	var startTime, endTime time.Time

	dtStart := evt.GetProperty(ics.ComponentPropertyDtStart)
	if len(dtStart.ICalParameters["VALUE"]) > 0 && dtStart.ICalParameters["VALUE"][0] == "DATE" {
		// This is an all day event
		startTime, err = evt.GetAllDayStartAt()
		if err != nil {
			log.Println("Error: GetAllDayStartAt")
		}

		endTime, err = evt.GetAllDayEndAt()
		if err != nil {
			// Handle zero time events
			endTime = startTime
			err = nil
		}
	} else {
		startTime, err = evt.GetStartAt()
		if err != nil {
			return errors.New("Error with event GetStartAt: " + err.Error())
		}

		endTime, err = evt.GetEndAt()
		if err != nil {
			// Handle zero time events
			endTime = startTime
			err = nil
		}
	}

	lastModifiedProp := evt.GetProperty(ics.ComponentPropertyLastModified)
	lastModifiedTime, err := time.Parse(icalTimestampFormatUtc, lastModifiedProp.Value)
	if err != nil {
		log.Println("Error: lastModifiedTime")
		return
	}

	// Build base event
	event := Event{
		UID:         evt.GetProperty(ics.ComponentPropertyUniqueId).Value,
		Summary:     evt.GetProperty(ics.ComponentPropertySummary).Value,
		Description: evt.GetProperty(ics.ComponentPropertyDescription).Value,
		// Categories
		AllDay:   allDay,
		Start:    startTime,
		End:      endTime,
		Duration: endTime.Sub(startTime),
		// Stamp
		// Created          *time.Time
		LastModified: lastModifiedTime,
		Location:     evt.GetProperty(ics.ComponentPropertyLocation).Value,
		// URL              string
		Status: evt.GetProperty(ics.ComponentPropertyStatus).Value,
		// RecurrenceID     string
		// ExcludeDates     []time.Time
		// Sequence         int
		// CustomAttributes map[string]string
	}

	// Check if this is a reoccuring event
	rr := evt.GetProperty(ics.ComponentPropertyRrule)
	if rr != nil {
		event.IsRecurring = true

		// Build rrule options
		roptions, err := ROptionsFromString(rr.Value)
		if err != nil {
			return errors.New("Error building ROptions: " + err.Error())
		}
		roptions.Dtstart = startTime

		// Process the rule
		r, err := rrule.NewRRule(roptions)
		if err != nil {
			return errors.New("Error evaluating RRule: " + err.Error())
		}

		// Add each reoccuring event

		for _, rStart := range r.Between(c.StartTime, c.EndTime, true) {
			// fmt.Println(i, rStart)
			rEvent := event
			rEvent.Start = rStart
			rEvent.End = rEvent.Start.Add(rEvent.Duration)
			c.addEvent(rEvent)
		}
	} else {
		// Add Single Event
		if event.End.After(c.StartTime) && event.Start.Before(c.EndTime) {
			c.addEvent(event)
		}
	}

	return
}

func (c *ICalendar) addEvent(event Event) {
	key := fmt.Sprintf("%v-%v", event.UID, event.Start.Unix())
	c.Events[key] = event
}
