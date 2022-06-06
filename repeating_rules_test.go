package go_ical

import (
	"reflect"
	"testing"
	"time"

	"github.com/teambition/rrule-go"
)

// Examples from https://icalendar.org/iCalendar-RFC-5545/3-8-5-3-recurrence-rule.html
var rruleTests = []struct {
	optionString string
	roptionsWant rrule.ROption
}{
	{"FREQ=WEEKLY;WKST=SU;COUNT=10;BYDAY=MO", rrule.ROption{
		Freq:      rrule.WEEKLY,
		Wkst:      rrule.SU,
		Count:     10,
		Byweekday: []rrule.Weekday{rrule.MO},
	}},
	{"FREQ=YEARLY;INTERVAL=2;BYMONTH=1;BYDAY=SU;BYHOUR=8,9",
		rrule.ROption{
			Freq:      rrule.YEARLY,
			Interval:  2,
			Bymonth:   []int{1},
			Byweekday: []rrule.Weekday{rrule.SU},
			Byhour:    []int{8, 9},
		}},
	{"FREQ=DAILY;COUNT=10",
		rrule.ROption{
			Freq:  rrule.DAILY,
			Count: 10,
		}},
	{"FREQ=DAILY;UNTIL=19971224T000000Z",
		rrule.ROption{
			Freq:  rrule.DAILY,
			Until: time.Date(1997, 12, 24, 0, 0, 0, 0, time.UTC),
		}},
	{"FREQ=DAILY;INTERVAL=10;COUNT=5",
		rrule.ROption{
			Freq:     rrule.DAILY,
			Interval: 10,
			Count:    5,
		}},
	{"FREQ=YEARLY;UNTIL=20000131T140000Z;BYMONTH=1;BYDAY=SU,MO,TU,WE,TH,FR,SA",
		rrule.ROption{
			Freq:      rrule.YEARLY,
			Until:     time.Date(2000, 1, 31, 14, 0, 0, 0, time.UTC),
			Bymonth:   []int{1},
			Byweekday: []rrule.Weekday{rrule.SU, rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA},
		}},
	{"FREQ=DAILY;UNTIL=20000131T140000Z;BYMONTH=1",
		rrule.ROption{
			Freq:    rrule.DAILY,
			Until:   time.Date(2000, 1, 31, 14, 0, 0, 0, time.UTC),
			Bymonth: []int{1},
		}},
	{"FREQ=WEEKLY;COUNT=10",
		rrule.ROption{
			Freq:  rrule.WEEKLY,
			Count: 10,
		}},
	{"FREQ=WEEKLY;INTERVAL=2;WKST=SU",
		rrule.ROption{
			Freq:     rrule.WEEKLY,
			Interval: 2,
			Wkst:     rrule.SU,
		}},
	{"FREQ=WEEKLY;UNTIL=19971007T000000Z;WKST=SU;BYDAY=TU,TH",
		rrule.ROption{
			Freq:      rrule.WEEKLY,
			Until:     time.Date(1997, 10, 7, 0, 0, 0, 0, time.UTC),
			Wkst:      rrule.SU,
			Byweekday: []rrule.Weekday{rrule.TU, rrule.TH},
		}},
	{"FREQ=MONTHLY;COUNT=10;BYMONTHDAY=1,-1",
		rrule.ROption{
			Freq:       rrule.MONTHLY,
			Count:      10,
			Bymonthday: []int{1, -1},
		}},
	{"FREQ=YEARLY;INTERVAL=3;COUNT=10;BYYEARDAY=1,100,200",
		rrule.ROption{
			Freq:      rrule.YEARLY,
			Interval:  3,
			Count:     10,
			Byyearday: []int{1, 100, 200},
		}},
	// {"FREQ=YEARLY;BYDAY=20MO", // Does not work yet, same as the one below
	// 	rrule.ROption{
	// 		Freq:      rrule.YEARLY,
	// 		Byweekno:  []int{20},
	// 		Byweekday: []rrule.Weekday{rrule.MO},
	// 	}},
	{"FREQ=YEARLY;BYWEEKNO=20;BYDAY=MO", //Same as the one above
		rrule.ROption{
			Freq:      rrule.YEARLY,
			Byweekno:  []int{20},
			Byweekday: []rrule.Weekday{rrule.MO},
		}},
	{"FREQ=MONTHLY;COUNT=3;BYDAY=TU,WE,TH;BYSETPOS=3",
		rrule.ROption{
			Freq:      rrule.MONTHLY,
			Count:     3,
			Byweekday: []rrule.Weekday{rrule.TU, rrule.WE, rrule.TH},
			Bysetpos:  []int{3},
		}},
	// {"",
	// 	rrule.ROption{}},
}

func TestROptionsFromString(t *testing.T) {
	for _, tt := range rruleTests {
		t.Run(tt.optionString, func(t *testing.T) {
			roptions, err := ROptionsFromString(tt.optionString)
			if err != nil {
				t.Fatalf("Error in creating ROptions '%s': %v", tt.optionString, err)
			}

			if !reflect.DeepEqual(roptions, tt.roptionsWant) {
				t.Fatalf("ROptions not set correctly '%s'", tt.optionString)
			}
		})
	}
}

// func Test(t *testing.T) {
// 	// roptions.Dtstart = time.Date(2000, 1, 1, 8, 0, 0, 0, nil)
// 	// r, _ := rrule.NewRRule(roptions)
// 	// fmt.Println("Events: ", len(r.All()))
// 	// fmt.Println(r.All())
// }
