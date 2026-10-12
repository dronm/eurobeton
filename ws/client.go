package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client holds information about client connections.
// Client events are stored in events map where key is an event ID.
type Client struct {
	ID          string
	Conn        *websocket.Conn
	EventServer EventPubSub

	writeMu   sync.Mutex
	mx        sync.Mutex	// events & visited
	VisitedAt time.Time
	events    map[string]struct{}
}

func NewClient(id string, conn *websocket.Conn, evSrv EventPubSub) *Client {
	return &Client{ID: id, Conn: conn, events: make(map[string]struct{}), EventServer: evSrv}
}

// AddEvent registeres a new event to the Client by its ID.
// EventSrv variable should be set, otherwise
// the registration silently fails.
func (c *Client) AddEvent(ID string) {
	if c.EventServer == nil {
		return
	}
	c.mx.Lock()
	if _, ok := c.events[ID]; !ok {
		c.EventServer.AddEvent(ID)
		c.events[ID] = struct{}{}
	}
	c.mx.Unlock()
}

// RemoveEvent unregisters an event by ID.
// EventSrv variable should be set, otherwise
// the registration silently fails.
func (c *Client) RemoveEvent(ID string) {
	if c.EventServer == nil {
		return
	}
	c.mx.Lock()
	if _, ok := c.events[ID]; ok {
		delete(c.events, ID)
		c.EventServer.RemoveEvent(ID)
	}
	c.mx.Unlock()
}

func (c *Client) RemoveAllEvents() {
	if c.EventServer == nil {
		return
	}
	c.mx.Lock()
	for evID := range c.events {
		c.EventServer.RemoveEvent(evID)
	}
	c.events = make(map[string]struct{})
	c.mx.Unlock()
}
