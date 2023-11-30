package domain

import "time"

type EventType string

const (
	InsertEventType EventType = "INSERT"
	UpdateEventType EventType = "UPDATE"
	DeleteEventType EventType = "DELETE"
)

type BaseEvent struct {
	ID        int64     `json:"id,omitempty" db:"id"`
	Method    EventType `json:"method,omitempty" db:"method"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Data      string    `json:"data,omitempty" db:"data"`
}

type Task struct {
	ID int64
}
