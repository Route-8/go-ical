package go_ical

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
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
	// Read into string for processing
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	calString := string(b)

	// Check for invalid timezone input from Twitch
	calString = strings.ReplaceAll(calString, "TZID=/", "TZID=")

	// Convert back to io.Reader and parse
	sr := strings.NewReader(calString)
	c.Calendar, err = ics.ParseCalendar(sr)
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

	fmt.Println("============ Building Base Event:")
	fmt.Printf("%+v\n\n", evt)

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

	lastModifiedTime, err := getComponentPropertyTimeSafe(evt, ics.ComponentPropertyLastModified)
	if err != nil {
		log.Println("Error: lastModifiedTime")
		return Event{}, err
	}

	// Build base event
	return Event{
		UID:         getComponentPropertyStringSafe(evt, ics.ComponentPropertyUniqueId),
		Summary:     getComponentPropertyStringSafe(evt, ics.ComponentPropertySummary),
		Description: getComponentPropertyStringSafe(evt, ics.ComponentPropertyDescription),
		// Categories
		AllDay:   allDay,
		Start:    startTime,
		End:      endTime,
		Duration: endTime.Sub(startTime),
		// Stamp
		// Created          *time.Time
		LastModified: lastModifiedTime,
		Location:     getComponentPropertyStringSafe(evt, ics.ComponentPropertyLocation),
		// URL              string
		Status: getComponentPropertyStringSafe(evt, ics.ComponentPropertyStatus),
		// RecurrenceID     string
		// ExcludeDates     []time.Time
		// Sequence         int
		// CustomAttributes map[string]string
	}, nil
}

func (c *ICalendar) addEvent(event Event) {
	c.Events[event.GetID()] = event
}

// Helper functions when retrieving ics.VEvent component properties

func getComponentPropertyStringSafe(evt *ics.VEvent, compProp ics.ComponentProperty) string {
	prop := evt.GetProperty(compProp)
	if prop != nil {
		return prop.Value
	}
	return ""
}

func getComponentPropertyTimeSafe(evt *ics.VEvent, compProp ics.ComponentProperty) (time.Time, error) {
	prop := evt.GetProperty(compProp)
	if prop != nil {
		return time.Parse(icalTimestampFormatUtc, prop.Value)
	}
	return time.Time{}, nil
}
