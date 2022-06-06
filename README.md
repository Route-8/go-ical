# go-ical Helper Library

## About the repo

When looking for a good golang ical library we found several existing library which did parts of what is needed, but nothing that put everything together.

This repo uses the following libraries to create a complete event list with reoccuring events unrolled into individual entries.

- [arran4/golang-ical](https://github.com/arran4/golang-ical)
- [teambition/rrule-go](https://github.com/teambition/rrule-go)

## Status

The repo is in alpha status. Very little testing has been done, use at your own risk.

## Example Usage

This is a quick example how to read in an iCalendar file, parse the events, and print each event back out.

```go
file, err := os.Open("calendar.ics")
if err != nil {
  log.Fatal(err)
}
defer file.Close()

// Only events between these two times will be returned
startTime := time.Now()
endTime := startTime.Add(time.Hour * time.Duration(30*24))

icalendar := go_ical.NewICalendar(startTime, endTime, nil)
err = icalendar.Parse(file)
if err != nil {
  log.Fatal(err)
}

fmt.Println("Calendar Event: ", len(icalendar.Events))
for i, e := range icalendar.Events {
  fmt.Printf("Event %v:\n%+v\n\n", i, e)
}
```