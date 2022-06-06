package go_ical

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

func ROptionsFromString(rruleString string) (roption rrule.ROption, err error) {
	tokens := strings.Split(rruleString, ";")
	for _, token := range tokens {
		s := strings.SplitN(token, "=", 2)
		if s[0] == "" {
			continue
		}
		err = addRule(&roption, s[0], s[1])
		if err != nil {
			break
		}
	}

	return
}

// Example rules from:
// https://icalendar.org/iCalendar-RFC-5545/3-3-10-recurrence-rule.html
func addRule(roption *rrule.ROption, key string, value string) (err error) {
	switch strings.ToLower(key) {
	case "freq":
		roption.Freq = freqRule(value)
	case "until":
		roption.Until, err = time.Parse(rrule.DateTimeFormat, value)
		if err != nil {
			fmt.Println("Error, could not convert until string to time: ", value)
		}
	case "count":
		roption.Count, err = strconv.Atoi(value)
		if err != nil {
			fmt.Println("Error, could not convert count string to int: ", value)
		}
	case "interval":
		roption.Interval, err = strconv.Atoi(value)
		if err != nil {
			fmt.Println("Error, could not convert interval string to int: ", value)
		}
	case "bysetpos":
		roption.Bysetpos = byIntArray(value)
	case "byeaster":
		roption.Byeaster = byIntArray(value)
	case "byday":
		// TODO - need to handle values such as '20MO'
		roption.Byweekday, err = byWeekdayArray(value)
	case "bymonth":
		roption.Bymonth = byIntArray(value)
	case "bymonthday":
		roption.Bymonthday = byIntArray(value)
	case "byyearday":
		roption.Byyearday = byIntArray(value)
	case "byweekno":
		roption.Byweekno = byIntArray(value)
	case "byhour":
		roption.Byhour = byIntArray(value)
	case "byminute":
		roption.Byminute = byIntArray(value)
	case "bysecond":
		roption.Bysecond = byIntArray(value)
	case "byweekdaylist":
		roption.Byweekday, err = byWeekdayArray(value)
	case "wkst":
		roption.Wkst, err = weekday(value)
	default:
		return fmt.Errorf("unhandled addRule key: %v = %v", key, value)
	}

	return
}

// TODO - Something like this will be needed in the future to support "byday" entries that contain an int
// func byDay(roption *rrule.ROption, value string) {
// 	tokens := strings.Split(value, ",")
// 	weekDays := make([]rrule.Weekday, 0)
//
// 	var err error
// 	for _, token := range tokens {
// 		if len(token) == 2 {
// 			weekDays = append(weekDays, weekday(token))
// 		} else {
// 			// TODO - handle these cases
// 		}
// 	}
// 	if len(weekDays) > 0 {
// 		roption.Byweekday = weekDays
// 	}
// }

func weekday(value string) (rrule.Weekday, error) {
	switch strings.ToLower(value) {
	case "su":
		return rrule.SU, nil
	case "mo":
		return rrule.MO, nil
	case "tu":
		return rrule.TU, nil
	case "we":
		return rrule.WE, nil
	case "th":
		return rrule.TH, nil
	case "fr":
		return rrule.FR, nil
	case "sa":
		return rrule.SA, nil
	}
	return rrule.MO, errors.New("Weekday not found: " + value)
}

func byIntArray(value string) []int {
	tokens := strings.Split(value, ",")
	values := make([]int, len(tokens))

	var err error
	for i, token := range tokens {
		values[i], err = strconv.Atoi(token)
		if err != nil {
			fmt.Println("Error, could not convert string to int: ", token)
		}
	}
	return values
}

func byWeekdayArray(value string) ([]rrule.Weekday, error) {
	tokens := strings.Split(value, ",")
	values := make([]rrule.Weekday, len(tokens))

	var err error
	for i, token := range tokens {
		values[i], err = weekday(token)
		if err != nil {
			break
		}
	}
	return values, err
}

func freqRule(value string) (freq rrule.Frequency) {
	switch strings.ToLower(value) {
	case "secondly":
		freq = rrule.SECONDLY
	case "minutely":
		freq = rrule.MINUTELY
	case "hourly":
		freq = rrule.HOURLY
	case "daily":
		freq = rrule.DAILY
	case "weekly":
		freq = rrule.WEEKLY
	case "monthly":
		freq = rrule.MONTHLY
	case "yearly":
		freq = rrule.YEARLY
	default:
		freq = rrule.YEARLY
	}
	return
}
