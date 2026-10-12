// Package session is a redis session implementation.
package session

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/dronm/session"
	_ "github.com/dronm/session/bereg"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dronm/gobizapp/logger"
)

const SessCookieKey = "token"

var (
	SessManager    *session.Manager
	//SessQueryParam = "token"
)

type SessConfiger interface {
	GetMaxLifeTime() int64
	GetMaxIdleTime() int64
	GetDestroyAllTime() string
	GetEncKey() string
}

func Initialize(sessConf SessConfiger, pool *pgxpool.Pool, lv string) error {
	var err error
	SessManager, err = session.NewManager("bereg",
		sessConf.GetMaxLifeTime(),
		sessConf.GetMaxIdleTime(),
		sessConf.GetDestroyAllTime(),
		pool,
		sessConf.GetEncKey(),
	)
	if err != nil {
		panic(fmt.Sprintf("session.NewManager: %v", err))
	}

	var sessLv session.LogLevel
	switch lv {
	case "debug":
		sessLv = session.LOG_LEVEL_DEBUG
	case "warn":
		sessLv = session.LOG_LEVEL_WARN
	default:
		sessLv = session.LOG_LEVEL_ERROR
	}

	lw := logger.NewLogWriter()
	SessManager.StartGC(lw, sessLv)

	return nil
}

func GenSessionID() (string, error) {
	sessionID := make([]byte, SessManager.GetSessionIDLen())
	_, err := rand.Read(sessionID)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(sessionID), nil
}
