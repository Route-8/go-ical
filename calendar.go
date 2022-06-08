package go_ical

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	ics "github.com/Route-8/golang-ical"
	rrule "github.com/teambition/rrule-go"
)

const icalTimestampFormatUtc = "20060102T150405Z"

type ICalendar struct {
	Calendar  *ics.Calendar
	Events    map[string]Event
	RRuleSets map[string]rrule.Set
	StartTime time.Time
	EndTime   time.Time
	TimeZone  *time.Location
}

func NewICalendar(startTime time.Time, endTime time.Time, timezone *time.Location) ICalendar {
	return ICalendar{
		StartTime: startTime,
		EndTime:   endTime,
		TimeZone:  timezone,
		RRuleSets: make(map[string]rrule.Set, 20),
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

	// Process Recurring Events First
	for i, evt := range c.Calendar.Events() {
		rr := evt.GetProperty(ics.ComponentPropertyRrule)
		if rr == nil {
			// Not Recurring, skip this entry for now
			continue
		}

		err := c.AddRecurringEvents(evt)
		if err != nil {
			fmt.Printf("%v: Error adding Recurring Events = %v\n", i, err)
			continue
		}
	}

	// Process Single Events Second
	for i, evt := range c.Calendar.Events() {
		rr := evt.GetProperty(ics.ComponentPropertyRrule)
		if rr != nil {
			// Recurring, skip this entry
			continue
		}

		err := c.AddSingleEvent(evt)
		if err != nil {
			fmt.Printf("%v: Error processEvents = %v\n", i, err)
			continue
		}
	}
	return
}

func (c *ICalendar) AddRecurringEvents(evt *ics.VEvent) (err error) {
	event, err := getBaseEvent(evt)
	if err != nil {
		return
	}
	event.IsRecurring = true

	// Build the options
	rr := evt.GetProperty(ics.ComponentPropertyRrule)
	options, err := rrule.StrToROption(rr.Value)
	if err != nil {
		return errors.New("Error building ROptions: " + err.Error())
	}
	options.Dtstart = event.Start

	// Build the RRule
	rule, err := rrule.NewRRule(*options)
	if err != nil {
		return errors.New("Error evaluating RRule: " + err.Error())
	}

	// Build the RRuleSet
	set := rrule.Set{}
	set.RRule(rule)

	// Add all ExDate entries to RRuleSet
	for _, prop := range evt.Properties {
		if prop.IANAToken == string(ics.ComponentPropertyExdate) {
			exDate, err := ics.GetTimeFromProp(&prop, false)
			if err != nil {
				return err
			}
			set.ExDate(exDate)
		}
	}

	// Get all RDate entries to RRuleSet
	for _, prop := range evt.Properties {
		if prop.IANAToken == string(ics.ComponentPropertyRdate) {
			rDate, err := ics.GetTimeFromProp(&prop, false)
			if err != nil {
				return err
			}
			set.RDate(rDate)
		}
	}

	// Save rule for future processing of exceptions
	c.RRuleSets[event.UID] = set

	// Add each Recurring event to the main event list
	for _, rStart := range set.Between(c.StartTime, c.EndTime, true) {
		rEvent := event
		rEvent.Start = rStart
		rEvent.End = rEvent.Start.Add(rEvent.Duration)
		rEvent.RecurrenceID = rStart.Format(icalTimestampFormatUtc)
		c.addEvent(rEvent)
	}
	return nil
}

func (c *ICalendar) AddSingleEvent(evt *ics.VEvent) (err error) {
	event, err := getBaseEvent(evt)
	if err != nil {
		return
	}

	// Check if this is a Recurring event exception
	if recurringIdTime, err := evt.GetRecurringIDAt(); err == nil {
		// This is an exception to a recurring event
		event.RecurrenceID = recurringIdTime.Format(icalTimestampFormatUtc)
	}

	// Add event to the main event list
	c.Events[event.GetID()] = event
	return
}

func getBaseEvent(evt *ics.VEvent) (Event, error) {
	var err error
	var allDay = false
	var startTime, endTime time.Time

	dtStart := evt.GetProperty(ics.ComponentPropertyDtStart)
	if len(dtStart.ICalParameters["VALUE"]) > 0 && dtStart.ICalParameters["VALUE"][0] == "DATE" {
		// This is an all day event
		allDay = true
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
			return Event{}, errors.New("Error with event GetStartAt: " + err.Error())
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
		return Event{}, err
	}

	// Build base event
	return Event{
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
	}, nil
}

func (c *ICalendar) addEvent(event Event) {
	c.Events[event.GetID()] = event
}
