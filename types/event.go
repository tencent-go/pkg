package types

type EventType string

const (
	EventTypeCreated EventType = "created"
	EventTypeUpdated EventType = "updated"
	EventTypeDeleted EventType = "deleted"
)

func (e EventType) Enum() Enum {
	return RegisterEnum(EventTypeCreated, EventTypeUpdated, EventTypeDeleted)
}
