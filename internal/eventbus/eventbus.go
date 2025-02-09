package eventbus

import (
	"encoding/json"
	"sync"
)

type (
	Bus interface {
		Register(identifier string) chan Event
		Broadcast(identifier string, evType Type, message string)
		BroadcastWithData(identifier string, evType Type, message string, data []byte)
	}

	Event struct {
		Type    Type            `json:"type"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	Type string
)

const (
	Error    Type = "error"
	Info     Type = "info"
	Success  Type = "success"
	Complete Type = "complete"
)

type eventPublisher struct {
	events map[string][]chan Event
	lock   sync.Mutex
}

func New() Bus {
	return &eventPublisher{
		events: make(map[string][]chan Event),
	}
}

func (e *eventPublisher) Register(identifier string) chan Event {
	e.lock.Lock()
	defer e.lock.Unlock()

	ch := make(chan Event, 1000)
	e.events[identifier] = append(e.events[identifier], ch)
	return ch
}

func (e *eventPublisher) Broadcast(identifier string, evType Type, message string) {
	e.lock.Lock()
	clients, ok := e.events[identifier]
	e.lock.Unlock()

	if ok && len(clients) > 0 {
		for _, ch := range clients {
			ch <- Event{
				Type:    evType,
				Message: message,
			}
		}
	}
}

func (e *eventPublisher) BroadcastWithData(identifier string, evType Type, message string, data []byte) {
	e.lock.Lock()
	clients, ok := e.events[identifier]
	e.lock.Unlock()

	ev := Event{
		Type:    evType,
		Message: message,
		Data:    data,
	}

	if ok && len(clients) > 0 {
		for _, ch := range clients {
			ch <- ev
		}
	}
}
