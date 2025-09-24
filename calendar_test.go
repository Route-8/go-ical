package go_ical

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRecurringEventInProgress(t *testing.T) {
	// This test verifies the bug where recurring events that are in progress
	// (started before the query window but are still ongoing) are filtered out

	// Create a timezone for consistent testing
	tz := time.UTC

	// Set up test times:
	// - Event starts at 10:00 AM and runs for 2 hours (until 12:00 PM)
	// - Query window starts at 11:00 AM (middle of the event) and ends at 1:00 PM
	eventStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz) // 10:00 AM
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz) // 11:00 AM (during event)
	queryEndTime := time.Date(2023, 10, 15, 13, 0, 0, 0, tz)   // 1:00 PM

	// Create an iCal with a daily recurring event
	icalData := createRecurringEventICal(eventStartTime, 2*time.Hour, "DAILY", "BYDAY=SU,MO,TU,WE,TH,FR,SA", 7)

	// Create calendar with query window starting in the middle of the event
	calendar := NewICalendar(queryStartTime, queryEndTime, tz)

	// Parse the iCal data
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	// Process events
	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Check if the in-progress event is included
	// The event started at 10:00 AM but the query starts at 11:00 AM
	// This event should still be included because it's ongoing during the query window

	foundInProgressEvent := false
	for _, event := range calendar.Events {
		// Check if this is the event that started before our query window
		if event.Start.Equal(eventStartTime) {
			foundInProgressEvent = true

			// Verify the event has the correct end time
			expectedEndTime := eventStartTime.Add(2 * time.Hour)
			if !event.End.Equal(expectedEndTime) {
				t.Errorf("Expected event end time %v, got %v", expectedEndTime, event.End)
			}

			// Verify the duration is correct
			if event.Duration != 2*time.Hour {
				t.Errorf("Expected event duration %v, got %v", 2*time.Hour, event.Duration)
			}

			t.Logf("Found in-progress event: Start=%v, End=%v, Duration=%v",
				event.Start, event.End, event.Duration)
			break
		}
	}

	if !foundInProgressEvent {
		t.Error("BUG CONFIRMED: In-progress recurring event was filtered out. " +
			"Event that started at 10:00 AM should be included in query window starting at 11:00 AM " +
			"because the event is still ongoing (ends at 12:00 PM)")

		// Log all found events for debugging
		t.Logf("Found %d events in total:", len(calendar.Events))
		for id, event := range calendar.Events {
			t.Logf("  Event ID: %s, Start: %v, End: %v", id, event.Start, event.End)
		}
	} else {
		t.Log("SUCCESS: In-progress recurring event was correctly included")
	}
}

func TestRecurringEventFullyWithinWindow(t *testing.T) {
	// This test verifies that normal recurring events (fully within the query window) still work

	tz := time.UTC

	// Event starts at 11:30 AM (within query window)
	eventStartTime := time.Date(2023, 10, 15, 11, 30, 0, 0, tz)
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz)  // 11:00 AM
	queryEndTime := time.Date(2023, 10, 15, 13, 0, 0, 0, tz)    // 1:00 PM

	icalData := createRecurringEventICal(eventStartTime, 1*time.Hour, "DAILY", "BYDAY=SU,MO,TU,WE,TH,FR,SA", 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)

	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Should find the event that starts at 11:30 AM
	foundEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundEvent = true
			break
		}
	}

	if !foundEvent {
		t.Error("Normal recurring event (fully within window) was not found")
	}
}

func TestRecurringEventSpansMultipleDays(t *testing.T) {
	// Test case where recurring event spans multiple query windows

	tz := time.UTC

	// Event starts at 11:00 PM and runs for 2 hours (ends at 1:00 AM next day)
	eventStartTime := time.Date(2023, 10, 15, 23, 0, 0, 0, tz)
	// Query window is the next day from 12:30 AM to 2:00 AM (during the event)
	queryStartTime := time.Date(2023, 10, 16, 0, 30, 0, 0, tz)
	queryEndTime := time.Date(2023, 10, 16, 2, 0, 0, 0, tz)

	icalData := createRecurringEventICal(eventStartTime, 2*time.Hour, "DAILY", "BYDAY=SU,MO,TU,WE,TH,FR,SA", 5)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)

	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Check if the event that started the previous day is included
	foundCrossDayEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundCrossDayEvent = true

			expectedEndTime := eventStartTime.Add(2 * time.Hour) // 1:00 AM next day
			if !event.End.Equal(expectedEndTime) {
				t.Errorf("Expected cross-day event end time %v, got %v", expectedEndTime, event.End)
			}

			t.Logf("Found cross-day event: Start=%v, End=%v", event.Start, event.End)
			break
		}
	}

	if !foundCrossDayEvent {
		t.Error("BUG CONFIRMED: Cross-day recurring event was filtered out. " +
			"Event that started at 11:00 PM should be included in query window starting at 12:30 AM " +
			"because the event spans across days and is ongoing during the query")
	}
}

func TestRecurringEventWeekly(t *testing.T) {
	// Test weekly recurring events with the in-progress bug scenario
	tz := time.UTC

	// Weekly event on Sundays, starts at 2:00 PM, runs for 3 hours
	// Query starts during the event (at 3:00 PM Sunday)
	eventStartTime := time.Date(2023, 10, 15, 14, 0, 0, 0, tz) // 2:00 PM Sunday
	queryStartTime := time.Date(2023, 10, 15, 15, 0, 0, 0, tz) // 3:00 PM Sunday (during event)
	queryEndTime := time.Date(2023, 10, 15, 18, 0, 0, 0, tz)   // 6:00 PM Sunday

	icalData := createRecurringEventICal(eventStartTime, 3*time.Hour, "WEEKLY", "BYDAY=SU", 4)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	foundInProgressEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundInProgressEvent = true
			t.Logf("Found weekly in-progress event: Start=%v, End=%v", event.Start, event.End)
			break
		}
	}

	if !foundInProgressEvent {
		t.Error("BUG CONFIRMED: Weekly recurring event in progress was filtered out")
	}
}

func TestRecurringEventWithExDate(t *testing.T) {
	// Test recurring events with exclusion dates
	tz := time.UTC

	// Event starts at 10:00 AM, recurring daily, with one exclusion
	eventStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz)
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz) // Query starts during first occurrence
	queryEndTime := time.Date(2023, 10, 17, 12, 0, 0, 0, tz)   // Query covers 2 days

	// Exclude the second occurrence (Oct 16)
	excludeDate := time.Date(2023, 10, 16, 10, 0, 0, 0, tz)
	icalData := createRecurringEventWithExDate(eventStartTime, 2*time.Hour, "DAILY", excludeDate, 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Should find first occurrence (in progress) but not the excluded one
	foundFirstEvent := false
	foundExcludedEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundFirstEvent = true
		} else if event.Start.Equal(excludeDate) {
			foundExcludedEvent = true
		}
	}

	if !foundFirstEvent {
		t.Error("BUG CONFIRMED: First recurring event (in progress) with EXDATE was filtered out")
	}
	if foundExcludedEvent {
		t.Error("Excluded event was incorrectly included despite EXDATE")
	}
}

func TestRecurringEventWithRDate(t *testing.T) {
	// Test recurring events with additional dates
	tz := time.UTC

	eventStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz)
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz) // During first event
	queryEndTime := time.Date(2023, 10, 18, 12, 0, 0, 0, tz)

	// Add an extra occurrence on Oct 17
	additionalDate := time.Date(2023, 10, 17, 10, 0, 0, 0, tz)
	icalData := createRecurringEventWithRDate(eventStartTime, 2*time.Hour, "DAILY", additionalDate, 2)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	foundOriginalEvent := false
	foundRDateEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundOriginalEvent = true
		} else if event.Start.Equal(additionalDate) {
			foundRDateEvent = true
		}
	}

	if !foundOriginalEvent {
		t.Error("BUG CONFIRMED: Original recurring event (in progress) with RDATE was filtered out")
	}
	if !foundRDateEvent {
		t.Error("RDATE additional occurrence was not found")
	}
}

func TestRecurringEventBoundaryConditions(t *testing.T) {
	// Test various boundary conditions
	tz := time.UTC

	// Test 1: Event ends exactly at query start time
	t.Run("EventEndsAtQueryStart", func(t *testing.T) {
		eventStartTime := time.Date(2023, 10, 15, 9, 0, 0, 0, tz)  // 9:00 AM
		queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz) // 11:00 AM
		queryEndTime := time.Date(2023, 10, 15, 13, 0, 0, 0, tz)   // 1:00 PM
		// Event duration is 2 hours (9-11 AM), ends exactly when query starts

		icalData := createRecurringEventICal(eventStartTime, 2*time.Hour, "DAILY", "BYDAY=SU,MO,TU,WE,TH,FR,SA", 3)
		calendar := NewICalendar(queryStartTime, queryEndTime, tz)
		reader := strings.NewReader(icalData)
		err := calendar.Parse(reader)
		if err != nil {
			t.Fatalf("Failed to parse iCal data: %v", err)
		}

		err = calendar.processEvents()
		if err != nil {
			t.Fatalf("Failed to process events: %v", err)
		}

		// Event that ends exactly at query start should NOT be included
		foundEvent := false
		for _, event := range calendar.Events {
			if event.Start.Equal(eventStartTime) {
				foundEvent = true
				break
			}
		}

		if foundEvent {
			t.Error("Event that ends exactly at query start was incorrectly included")
		}
	})

	// Test 2: Event completely encompasses query window
	t.Run("EventEncompassesQuery", func(t *testing.T) {
		eventStartTime := time.Date(2023, 10, 15, 8, 0, 0, 0, tz)  // 8:00 AM
		queryStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz) // 10:00 AM
		queryEndTime := time.Date(2023, 10, 15, 12, 0, 0, 0, tz)   // 12:00 PM
		// Event runs 8 AM to 2 PM (6 hours), completely encompasses query window

		icalData := createRecurringEventICal(eventStartTime, 6*time.Hour, "DAILY", "", 2)
		calendar := NewICalendar(queryStartTime, queryEndTime, tz)
		reader := strings.NewReader(icalData)
		err := calendar.Parse(reader)
		if err != nil {
			t.Fatalf("Failed to parse iCal data: %v", err)
		}

		err = calendar.processEvents()
		if err != nil {
			t.Fatalf("Failed to process events: %v", err)
		}

		foundEvent := false
		for _, event := range calendar.Events {
			if event.Start.Equal(eventStartTime) {
				foundEvent = true
				break
			}
		}

		if !foundEvent {
			t.Error("BUG CONFIRMED: Event that completely encompasses query window was filtered out")
		}
	})
}

func TestMultipleOverlappingRecurringEvents(t *testing.T) {
	// Test multiple recurring events that overlap
	tz := time.UTC

	// Two overlapping events
	event1StartTime := time.Date(2023, 10, 15, 9, 0, 0, 0, tz)  // 9:00 AM, 3 hours
	event2StartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz) // 10:00 AM, 2 hours
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz)  // 11:00 AM (during both)
	queryEndTime := time.Date(2023, 10, 15, 14, 0, 0, 0, tz)    // 2:00 PM

	ical1 := createRecurringEventICal(event1StartTime, 3*time.Hour, "DAILY", "", 3)
	ical2 := createRecurringEventICalWithUID(event2StartTime, 2*time.Hour, "DAILY", "", 3, "event2@example.com")

	// Combine both iCal strings
	combinedICal := combineICalEvents(ical1, ical2)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(combinedICal)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse combined iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	foundEvent1 := false
	foundEvent2 := false
	for _, event := range calendar.Events {
		if event.Start.Equal(event1StartTime) {
			foundEvent1 = true
		} else if event.Start.Equal(event2StartTime) {
			foundEvent2 = true
		}
	}

	if !foundEvent1 {
		t.Error("BUG CONFIRMED: First overlapping recurring event (in progress) was filtered out")
	}
	if !foundEvent2 {
		t.Error("BUG CONFIRMED: Second overlapping recurring event (in progress) was filtered out")
	}
}

func TestAllDayRecurringEvents(t *testing.T) {
	// Test all-day recurring events
	tz := time.UTC

	// All-day event on Oct 15, query starts at 2 PM that day
	eventStartTime := time.Date(2023, 10, 15, 0, 0, 0, 0, tz)  // Midnight (all-day)
	queryStartTime := time.Date(2023, 10, 15, 14, 0, 0, 0, tz) // 2:00 PM same day
	queryEndTime := time.Date(2023, 10, 16, 2, 0, 0, 0, tz)    // Next day

	icalData := createAllDayRecurringEvent(eventStartTime, "DAILY", 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	foundAllDayEvent := false
	for _, event := range calendar.Events {
		if event.AllDay && event.Start.YearDay() == eventStartTime.YearDay() {
			foundAllDayEvent = true
			t.Logf("Found all-day event: Start=%v, End=%v, AllDay=%v", event.Start, event.End, event.AllDay)
			break
		}
	}

	if !foundAllDayEvent {
		t.Error("BUG CONFIRMED: All-day recurring event in progress was filtered out")
	}
}

func TestMonthlyRecurringEvents(t *testing.T) {
	// Test monthly recurring events
	tz := time.UTC

	// Monthly event on the 15th, starts at 10 AM, runs for 3 hours
	eventStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz)
	queryStartTime := time.Date(2023, 10, 15, 11, 30, 0, 0, tz) // Query during first occurrence
	queryEndTime := time.Date(2023, 11, 16, 12, 0, 0, 0, tz)    // Covers next month too

	icalData := createRecurringEventICal(eventStartTime, 3*time.Hour, "MONTHLY", "BYMONTHDAY=15", 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	foundFirstEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundFirstEvent = true
			break
		}
	}

	if !foundFirstEvent {
		t.Error("BUG CONFIRMED: Monthly recurring event in progress was filtered out")
	}
}

func TestRecurringEventAfterQueryWindow(t *testing.T) {
	// Ensure events that start after the query window are not included
	tz := time.UTC

	eventStartTime := time.Date(2023, 10, 15, 14, 0, 0, 0, tz) // 2:00 PM
	queryStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz) // 10:00 AM
	queryEndTime := time.Date(2023, 10, 15, 12, 0, 0, 0, tz)   // 12:00 PM (before event)

	icalData := createRecurringEventICal(eventStartTime, 2*time.Hour, "DAILY", "", 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Should not find any events since they start after query window
	if len(calendar.Events) > 0 {
		t.Errorf("Found %d events that should not be included (start after query window)", len(calendar.Events))
		for id, event := range calendar.Events {
			t.Logf("  Unexpected event ID: %s, Start: %v", id, event.Start)
		}
	}
}

func TestRecurringEventBeforeQueryWindow(t *testing.T) {
	// Ensure events that end before the query window are not included
	tz := time.UTC

	eventStartTime := time.Date(2023, 10, 15, 8, 0, 0, 0, tz)  // 8:00 AM
	queryStartTime := time.Date(2023, 10, 15, 12, 0, 0, 0, tz) // 12:00 PM
	queryEndTime := time.Date(2023, 10, 15, 14, 0, 0, 0, tz)   // 2:00 PM
	// Event runs 8-10 AM (2 hours), ends before query starts

	icalData := createRecurringEventICal(eventStartTime, 2*time.Hour, "DAILY", "", 3)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Should not find the first event since it ends before query starts
	// But might find subsequent daily occurrences that fall within the window
	foundOriginalEvent := false
	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) {
			foundOriginalEvent = true
			break
		}
	}

	if foundOriginalEvent {
		t.Error("Event that ends before query window was incorrectly included")
	}
}

func TestRecurringEventWithException(t *testing.T) {
	// Test recurring events with exception instances (RECURRENCE-ID)
	tz := time.UTC

	// Daily recurring event starting Oct 15, 10 AM, 2 hours duration
	eventStartTime := time.Date(2023, 10, 15, 10, 0, 0, 0, tz)
	queryStartTime := time.Date(2023, 10, 15, 11, 0, 0, 0, tz) // During first occurrence
	queryEndTime := time.Date(2023, 10, 17, 12, 0, 0, 0, tz)   // Covers 3 days

	// Exception on Oct 16 - same time but different summary/duration
	exceptionDate := time.Date(2023, 10, 16, 10, 0, 0, 0, tz)
	icalData := createRecurringEventWithException(eventStartTime, 2*time.Hour, "DAILY", exceptionDate, 1*time.Hour, 5)

	calendar := NewICalendar(queryStartTime, queryEndTime, tz)
	reader := strings.NewReader(icalData)
	err := calendar.Parse(reader)
	if err != nil {
		t.Fatalf("Failed to parse iCal data: %v", err)
	}

	err = calendar.processEvents()
	if err != nil {
		t.Fatalf("Failed to process events: %v", err)
	}

	// Should find first occurrence (in progress) and exception, but not regular second occurrence
	foundFirstEvent := false
	foundExceptionEvent := false
	regularSecondEvent := false

	for _, event := range calendar.Events {
		if event.Start.Equal(eventStartTime) && event.Duration == 2*time.Hour {
			foundFirstEvent = true
		} else if event.Start.Equal(exceptionDate) && event.Duration == 1*time.Hour {
			foundExceptionEvent = true
		} else if event.Start.Equal(exceptionDate) && event.Duration == 2*time.Hour {
			regularSecondEvent = true
		}
	}

	if !foundFirstEvent {
		t.Error("BUG CONFIRMED: First recurring event (in progress) with exception was filtered out")
	}
	if !foundExceptionEvent {
		t.Error("Exception event was not found")
	}
	if regularSecondEvent {
		t.Error("Regular recurring event was found instead of exception (exception override failed)")
	}
}

// Helper functions for creating various types of iCal data

func createRecurringEventICal(startTime time.Time, duration time.Duration, freq string, byDay string, count int) string {
	return createRecurringEventICalWithUID(startTime, duration, freq, byDay, count, "test-recurring-event@example.com")
}

func createRecurringEventICalWithUID(startTime time.Time, duration time.Duration, freq string, byDay string, count int, uid string) string {
	endTime := startTime.Add(duration)

	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VTIMEZONE
TZID:UTC
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0000
TZOFFSETTO:+0000
TZNAME:UTC
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:` + uid + `
DTSTART:` + startTime.Format("20060102T150405Z") + `
DTEND:` + endTime.Format("20060102T150405Z") + `
SUMMARY:Test Recurring Event
DESCRIPTION:A test recurring event to verify in-progress event handling
RRULE:FREQ=` + freq + `;COUNT=` + fmt.Sprintf("%d", count)

	if byDay != "" {
		ical += `;` + byDay
	}

	ical += `
END:VEVENT
END:VCALENDAR`

	return ical
}

func createRecurringEventWithExDate(startTime time.Time, duration time.Duration, freq string, exDate time.Time, count int) string {
	endTime := startTime.Add(duration)

	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VTIMEZONE
TZID:UTC
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0000
TZOFFSETTO:+0000
TZNAME:UTC
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:test-exdate-event@example.com
DTSTART:` + startTime.Format("20060102T150405Z") + `
DTEND:` + endTime.Format("20060102T150405Z") + `
SUMMARY:Test Recurring Event with ExDate
RRULE:FREQ=` + freq + `;COUNT=` + fmt.Sprintf("%d", count) + `
EXDATE:` + exDate.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`

	return ical
}

func createRecurringEventWithRDate(startTime time.Time, duration time.Duration, freq string, rDate time.Time, count int) string {
	endTime := startTime.Add(duration)

	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VTIMEZONE
TZID:UTC
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0000
TZOFFSETTO:+0000
TZNAME:UTC
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:test-rdate-event@example.com
DTSTART:` + startTime.Format("20060102T150405Z") + `
DTEND:` + endTime.Format("20060102T150405Z") + `
SUMMARY:Test Recurring Event with RDate
RRULE:FREQ=` + freq + `;COUNT=` + fmt.Sprintf("%d", count) + `
RDATE:` + rDate.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`

	return ical
}

func createAllDayRecurringEvent(startTime time.Time, freq string, count int) string {
	// All-day events use DATE format, not DATETIME
	startDate := startTime.Format("20060102")
	endDate := startTime.AddDate(0, 0, 1).Format("20060102") // Next day

	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VTIMEZONE
TZID:UTC
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0000
TZOFFSETTO:+0000
TZNAME:UTC
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:test-allday-event@example.com
DTSTART;VALUE=DATE:` + startDate + `
DTEND;VALUE=DATE:` + endDate + `
SUMMARY:Test All-Day Recurring Event
RRULE:FREQ=` + freq + `;COUNT=` + fmt.Sprintf("%d", count) + `
END:VEVENT
END:VCALENDAR`

	return ical
}

func combineICalEvents(ical1, ical2 string) string {
	// Simple way to combine two iCal strings by extracting events
	lines1 := strings.Split(ical1, "\n")
	lines2 := strings.Split(ical2, "\n")

	// Start with calendar header from first iCal
	var combined []string
	for _, line := range lines1 {
		if line == "END:VCALENDAR" {
			break
		}
		combined = append(combined, line)
	}

	// Add event from second iCal (skip header/footer)
	inEvent := false
	for _, line := range lines2 {
		if line == "BEGIN:VEVENT" {
			inEvent = true
		}
		if inEvent {
			combined = append(combined, line)
		}
		if line == "END:VEVENT" {
			inEvent = false
		}
	}

	// Add calendar footer
	combined = append(combined, "END:VCALENDAR")

	return strings.Join(combined, "\n")
}

func createRecurringEventWithException(startTime time.Time, duration time.Duration, freq string, exceptionDate time.Time, exceptionDuration time.Duration, count int) string {
	endTime := startTime.Add(duration)
	exceptionEndTime := exceptionDate.Add(exceptionDuration)

	ical := `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//Test//EN
BEGIN:VTIMEZONE
TZID:UTC
BEGIN:STANDARD
DTSTART:19700101T000000
TZOFFSETFROM:+0000
TZOFFSETTO:+0000
TZNAME:UTC
END:STANDARD
END:VTIMEZONE
BEGIN:VEVENT
UID:test-exception-event@example.com
DTSTART:` + startTime.Format("20060102T150405Z") + `
DTEND:` + endTime.Format("20060102T150405Z") + `
SUMMARY:Test Recurring Event
RRULE:FREQ=` + freq + `;COUNT=` + fmt.Sprintf("%d", count) + `
END:VEVENT
BEGIN:VEVENT
UID:test-exception-event@example.com
DTSTART:` + exceptionDate.Format("20060102T150405Z") + `
DTEND:` + exceptionEndTime.Format("20060102T150405Z") + `
SUMMARY:Test Recurring Event (Modified)
RECURRENCE-ID:` + exceptionDate.Format("20060102T150405Z") + `
END:VEVENT
END:VCALENDAR`

	return ical
}