package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dronm/crudifier/v2"
	"github.com/dronm/gobizapp/errs"
	"github.com/dronm/gobizapp/logger"
	"github.com/gorilla/websocket"
)

type SrvResponse struct {
	EventID string            `json:"event_id"`
	QueryID string            `json:"query_id"`
	Payload any               `json:"payload"`
	Error   *SrvResponseError `json:"error"`
}

type SrvResponseError struct {
	Code    errs.ErrorCode `json:"code"`
	Message string         `json:"message"`
}

func NewSrvResponseError(httpErr int, fnName string, isProduction bool, err error) *SrvResponseError {
	errText := fmt.Sprintf("%s: %v", fnName, err)

	// log real message here
	logger.Logger.Error(errText)

	resp := SrvResponseError{}

	var pubErr errs.PublicError
	var validErr *crudifier.ValidationError // all validation errors

	if errors.As(err, &pubErr) {
		resp.Message = pubErr.Error()
		resp.Code = pubErr.Code()

	} else if errors.As(err, &validErr) {
		resp.Message = validErr.Error()
		resp.Code = errs.ValidationFailed

	} else {
		switch httpErr {
		case http.StatusInternalServerError:
			resp.Code = errs.InternalError
		case http.StatusBadRequest:
			resp.Code = errs.BadRequest
		default:
			resp.Code = errs.UnknownError
		}
		if !isProduction {
			resp.Message = errText
		} else {
			resp.Message = errs.ErrorDescr(resp.Code)
		}
	}
	return &resp
}

func (s *WSServer) SendMessage(c *Client, resp *SrvResponse) error {
    respData, err := json.Marshal(resp)
    if err != nil {
        logger.Logger.Errorf("WSServer SendMessage json.Marshal(): %v", err)
        return fmt.Errorf("json marshal: %w", err)
    }

    // All websocket writes MUST be serialized
    c.writeMu.Lock()
    err = c.Conn.WriteMessage(websocket.TextMessage, respData)
    c.writeMu.Unlock()

    if err != nil {
        logger.Logger.Errorf("WSServer SendMessage WriteMessage(): %v", err)
        return fmt.Errorf("write message: %w", err)
    }

    return nil
}

func (s *WSServer) SendMessageToClientID(clientID string, msg any) error {
    msgB, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    s.clientsMx.RLock()
    conns := append([]*Client(nil), s.clients[clientID]...) // copy slice
    s.clientsMx.RUnlock()

    if len(conns) == 0 {
        return fmt.Errorf("WSServer.SendMessageToClientID() client not found: %s", clientID)
    }

    for _, c := range conns {
		logger.Logger.Debugf("WSServer.SendMessageToClientID(): clientID:%s, msg: %v", clientID, msg)
        c.writeMu.Lock()
        err := c.Conn.WriteMessage(websocket.TextMessage, msgB)
        c.writeMu.Unlock()

        if err != nil {
            go s.removeConn(clientID, c)
            return err
        }
    }
    return nil
}

func (s *WSServer) HasClientID(clientID string) bool {

	s.clientsMx.RLock()
	defer s.clientsMx.RUnlock()

	_, ok := s.clients[clientID]
	return ok
}

/*
func (s *WSServer) SendMessageToClient(client []*Client, msg any) error {
	msgB, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("WSServer SendMessage json.Marshal(): %v", err)
	}

	s.clientsMx.RLock()
	defer s.clientsMx.RUnlock()

	for _, conn := range client {
		if err := conn.Conn.WriteMessage(websocket.TextMessage, msgB); err != nil {
			return fmt.Errorf("WSServer SendMessageToClient conn.WriteMessage(): %v", err)			
		}		
	}
	return nil
}
*/

// PublishEvent sends SrvResponse with payload and eventID to all clients registered for this event. 
func (s *WSServer) PublishEvent(publisherID, eventID string, payload any) error {
    // 1. Build the message once
	msg := SrvResponse{
		QueryID: "", // Set this if needed
		EventID: eventID,
		Payload: payload,
		Error:   nil,
	}
    msgB, err := json.Marshal(msg)
    if err != nil {
		return fmt.Errorf("json.Marshal(): %v", err)
    }

    // 2. Copy all clients that subscribed to this event
    s.clientsMx.RLock()
    var targets []*Client
    for _, clientDevList := range s.clients {
        for _, c := range clientDevList {
			if c.ID == publisherID {
				continue
			}
			if _, ok := c.events[eventID]; ok {
                targets = append(targets, c)
			}
        }
    }
    s.clientsMx.RUnlock()

    // 3. Send message outside of lock, per-client
    for _, c := range targets {
        c.writeMu.Lock()
        err := c.Conn.WriteMessage(websocket.TextMessage, msgB)
        c.writeMu.Unlock()

        if err != nil {
            go func(clientID string, bad *Client) {
                s.removeConn(clientID, bad)
            }(c.ID, c)
        }
    }

	// s.clientsMx.RLock()
	// defer s.clientsMx.RUnlock()
	//
	// for _, clientDevList := range s.clients {
	// 	for _, clientDev := range clientDevList {
	// 		if clientDev.ID == publisherID {
	// 			continue
	// 		}
	// 		if _, ok := clientDev.events[eventID]; ok {
	// 			s.SendMessage(clientDev.Conn, &msg)
	// 		}
	// 	}
	// }

	return nil
}
