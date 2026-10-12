package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dronm/gobizapp/api"
	"github.com/dronm/gobizapp/controllers"
	"github.com/dronm/gobizapp/database"

	"github.com/dronm/gobizapp/logger"
	"github.com/dronm/session"
	"github.com/gin-gonic/gin"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

// Init provides an access to a websocket server, it upgrades
// http to a websocket, mantains a constant connection.
func (s *WSServer) Init(c *gin.Context) {
	funcName := "WebSocketInit"
	sess := controllers.GetSession(c, funcName)
	if sess == nil {
		return
	}
	if respCode, err := s.HandleConnection(c.Writer, c.Request, sess, c); err != nil {
		controllers.ServeError(c, respCode, funcName+" ws.Init()", err)
	}
}

type ClientMessage struct {
	Func    string          `json:"f"`
	QueryID string          `json:"q"`
	Payload json.RawMessage `json:"p"`
}

func (s *WSServer) HandleConnection(w http.ResponseWriter, r *http.Request, sess session.Session, c *gin.Context) (int, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("upgrader.Upgrade(): %v", err)
	}

	clientID := sess.SessionID()
	logger.Logger.Warnf("WSServer HandleConnection: adding new client with ID: %s", clientID)

	client := NewClient(clientID, conn, s.EventServer)

	s.clientsMx.Lock()
	s.clients[clientID] = append(s.clients[clientID], client)
	s.clientsMx.Unlock()

	defer func() {
		logger.Logger.Warnf("ws: closing connection %s, removing from session client list", clientID)

		s.clientsMx.Lock()
		clients := s.clients[clientID]
		for i, c := range clients {
			if c == client {
				c.RemoveAllEvents()
				s.clients[clientID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}

		if len(s.clients[clientID]) == 0 {
			delete(s.clients, clientID)
		}
		s.clientsMx.Unlock()

		conn.Close()
	}()

	done := make(chan struct{})
	go func() {
		<-r.Context().Done()
		logger.Logger.Info("ws: request context closed, terminating connection")
		conn.Close()
		close(done)
	}()

	for {
		select {
		case <-done:
			return http.StatusOK, nil

		default:
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
				) {
					return http.StatusOK, nil
				}
				return http.StatusInternalServerError, fmt.Errorf("conn.ReadMessage(): %v", err)
			}

			logger.Logger.Debugf("Received: type:%d, msg:%s\n", msgType, string(msg))

			// update client visit time
			client.mx.Lock()
			client.VisitedAt = time.Now()
			client.mx.Unlock()

			resp := SrvResponse{EventID: "Response"}

			clientMsg := ClientMessage{}
			if err := json.Unmarshal(msg, &clientMsg); err != nil {
				resp.Error = NewSrvResponseError(
					http.StatusInternalServerError,
					"json.Unmarshal client message",
					s.IsProduction, err,
				)
				_ = s.SendMessage(client, &resp)
				continue
			}

			if clientMsg.Func == "" {
				resp.Error = NewSrvResponseError(
					http.StatusInternalServerError,
					"func is not defined",
					s.IsProduction, nil,
				)
				_ = s.SendMessage(client, &resp)
				continue
			}

			// Service.Method
			service := strings.Split(clientMsg.Func, ".")
			if len(service) != 2 {
				resp.Error = NewSrvResponseError(
					http.StatusInternalServerError,
					"func structure is bad",
					s.IsProduction, nil,
				)
				_ = s.SendMessage(client, &resp)
				continue
			}

			params, err := api.UnmarshalParams(clientMsg.Payload)
			if err != nil {
				resp.Error = NewSrvResponseError(
					http.StatusInternalServerError,
					"api.UnmarshalParams",
					s.IsProduction, err,
				)
				_ = s.SendMessage(client, &resp)
				continue
			}

			// permission check
			if s.isMethodAllowed != nil {
				if err := s.isMethodAllowed(sess, service[0]+service[1]); err != nil {
					conn.WriteMessage(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "method is not allowed"),
					)
					return http.StatusUnauthorized,
						fmt.Errorf("isMethodAllowed(%s.%s)", service[0], service[1])
				}
			}

			methDuration := defMaxMethodCallDuration
			if s.MaxMethodCallDuration != 0 {
				methDuration = s.MaxMethodCallDuration
			}
			ctx, cancel := context.WithTimeout(r.Context(), methDuration)

			var resHTTP int
			resHTTP, resp.Payload, err = controllers.CallServiceMethod(
				ctx,
				service[0],
				service[1],
				params,
				&api.ServiceContext{
					DB:      database.DB,
					Session: sess,
					QueryID: clientMsg.QueryID,
				},
			)

			if err != nil {
				resp.Error = NewSrvResponseError(
					resHTTP,
					"controllers.CallServiceMethod",
					s.IsProduction, err,
				)
				_ = s.SendMessage(client, &resp)
				cancel()
				continue
			}

			if resp.Payload != nil {
				if err := s.SendMessage(client, &resp); err != nil {
					logger.Logger.Errorf("error sending response: %v", err)
				}
			}

			cancel()
		}
	}
}
