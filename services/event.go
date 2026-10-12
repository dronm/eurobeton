package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/dronm/ds/v4"
	"github.com/dronm/session"
)

var ErrEvHandlerNotDefined = errors.New("EventService: EvHandler not set")

type EventHandler interface {
	SubscribeToEvent(sessionID string, eventID string) error
	UnsubscribeFromEvent(sessionID string, eventID string) error
	PublishEvent(publisherID, eventID string, payload any) error
}

var EvHandler EventHandler

type EventService struct {
	DB      ds.Provider
	Session session.Session
	QueryID string
}

func (s *EventService) SetDB(db ds.Provider) {
	s.DB = db
}

func (s *EventService) SetSession(sess session.Session) {
	s.Session = sess
}

func (s *EventService) SetQueryID(queryID string) {
	s.QueryID = queryID
}

func NewEventService(db ds.Provider, sess session.Session) *EventService {
	return &EventService{DB: db, Session: sess}
}

func (s *EventService) Subscribe(ctx context.Context, events []map[string]string) error {
	if EvHandler == nil {
		return ErrEvHandlerNotDefined
	}
	sessID := s.Session.SessionID()
	for _, ev := range events {
		id, ok := ev["id"]
		if !ok {
			return fmt.Errorf("id not found in events structure")
		}
		if err := EvHandler.SubscribeToEvent(sessID, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *EventService) Unsubscribe(ctx context.Context, events []map[string]string) error {
	if EvHandler == nil {
		return ErrEvHandlerNotDefined
	}

	sessID := s.Session.SessionID()
	for _, ev := range events {
		id, ok := ev["id"]
		if !ok {
			return fmt.Errorf("id not found in events structure")
		}
		if err := EvHandler.UnsubscribeFromEvent(sessID, id); err != nil {
			return err
		}
	}
	return nil
}
