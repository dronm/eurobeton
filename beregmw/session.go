// Package beregmw
package beregmw

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dronm/gobizapp/database"
	"github.com/dronm/gobizapp/logger"
	sess "github.com/dronm/session"
)

func extractCookieSessionID(c *gin.Context, cookieKey string) string {
	// cookie value if exists
	if vCookie, err := c.Cookie(cookieKey); err == nil {
		return vCookie
	}
	return ""
}

func extractHeaderSessionID(c *gin.Context) string {
	// cookie valuer
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

type SessionManager interface {
	SessionStart(string) (sess.Session, error)
	GetMaxLifeTime() int64
}

func SessionMiddleware(
	manager SessionManager,
	cookieKey string,
	isProduction bool,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/favicon.ico" {
			c.Next()
			return
		}
		sessID := extractCookieSessionID(c, cookieKey)
		if sessID == "" {
			sessID = extractHeaderSessionID(c) // for API calls without cookies
		}
		if sessID == "" {
			sessID = c.Param("sessID")
			if sessID != "" && sessID[0:1] == "/" {
				sessID = sessID[1:]
			}
		}
		var sessionAlreadyExists bool
		if sessID != "" {
			sessionAlreadyExists = true
			sessID = BeregSessionID(c.Request.Context(), sessID)
		}

		c.Set("session_loader", func() (sess.Session, error) {
			sess, err := manager.SessionStart(sessID)
			if err != nil {
				return nil, fmt.Errorf("manager.SessionStart(): %v", err)
			}

			if !sessionAlreadyExists {
				now := time.Now()
				_ = sess.Put("time_created", now)
			}

			// set cookie
			c.SetCookie(
				cookieKey,
				sess.SessionID(),
				int(manager.GetMaxLifeTime()),
				"/",
				"",
				isProduction,
				true,
			)
			c.Set("session", sess)
			return sess, nil
		})

		c.Next()
	}
}

// BeregSessionID loggs all errrors, returns corrected session ID or empty on error
func BeregSessionID(ctx context.Context, sessID string) string {
	idParts := strings.Split(sessID, ":")
	if len(idParts) < 2 {
		return sessID // as is
	}

	// retrieve session ID from database by its pubkey idParts[0]
	poolConn, connID, err := database.DB.GetSecondary(ctx, "")
	if err != nil {
		logger.Logger.Errorf("GetSecondary(): %v", err)
		return ""
	}

	defer database.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if err := conn.QueryRow(ctx,
		`SELECT 
			trim(session_id)
		FROM logins 
		WHERE pub_key=$1 AND date_time_out IS NULL
		LIMIT 1`,
		idParts[0],
	).Scan(&sessID); err != nil {
		logger.Logger.Errorf("BeregSessionID select session_id Scan(): %v, for pub_key: %s", err, sessID)
		sessID = ""
	}

	return sessID
}
