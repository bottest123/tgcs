package user

import "time"

type Event struct {
	EventDateTime time.Time
	EventDescription string
}