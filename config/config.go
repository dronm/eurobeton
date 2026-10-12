// Package config
package config

import (
	"bytes"
	"encoding/json"
	"os"
)

const (
	defAppShutdownTimeout = 10
)

// Config is a global variable to hold config value.
var Config AppConfig

// DBStorage describes common db connections: one primary and several secondary servers.
type DBStorage struct {
	Primary     string            `json:"primary"`
	Secondaries map[string]string `json:"secondaries"`
}

func (st DBStorage) GetPrimary() string {
	return st.Primary
}

func (st DBStorage) GetSecondaries() map[string]string {
	return st.Secondaries
}

// Session holds configuration for sessions.
type Session struct {
	MaxLifeTime    int64  `json:"maxLifeTime"`
	MaxIdleTime    int64  `json:"maxIdleTime"`
	EncKey         string `json:"encKey"`
	DestroyAllTime string `json:"destroy_all_time"`
}

func (s Session) GetMaxLifeTime() int64 {
	return s.MaxLifeTime
}

func (s Session) GetMaxIdleTime() int64 {
	return s.MaxIdleTime
}

func (s Session) GetDestroyAllTime() string {
	return s.DestroyAllTime
}

func (s Session) GetEncKey() string {
	return s.EncKey
}

type WebSocketServer struct {
	Start           bool   `json:"start"`
	CleanupInterval int    `json:"cleanupInterval"` // cleanup interval in seconds
	Address         string `json:"address"`
}

type EventServer struct {
	Start         bool `json:"start"`
	PauseInverval int  `json:"pauseInterval"` // loop pause interval in ms
}

type Elkon struct {
	FailoverCheckInterval int `json:"failoverCheckInterval"`
}

type AppConfig struct {
	BaseURL      string          `json:"baseURL"`
	Production   bool            `json:"production"`
	LogLevel     string          `json:"logLevel"` // debug|warn|note|error
	LogToFile    string          `json:"logToFile"`
	DB           DBStorage       `json:"db"`
	WSServer     WebSocketServer `json:"wsServer"`    // start ws server
	EventServer  EventServer     `json:"eventServer"` // start event server
	AppID        string          `json:"appId"`       // Application ID
	Session      Session         `json:"session"`
	ReportErrors bool            `json:"reportErrors"` // If set to true public method error will be send to client,
	// otherwise error will be logged, short text will be sent to client
	HTTPServer         string `json:"httpServer"`
	MiddlewareTimeout  int    `json:"middlewareTimeout"`  // timeout in milliseconds
	ClientQueryTimeout int    `json:"clientQueryTimeout"` // timeout in milliseconds

	DebugQueries       bool `json:"debugQueries"`
	AppShutdownTimeout int  `json:"appShutdownTimeout"` // application shutdown timeout in seconds

	DefaultLocale string `json:"defaultLocale"`

	TechMail string `json:"techMail"` // Author name && email
	Author   string `json:"author"`

	PublicStoreDir string `json:"publicPictureDir"`

	Elkon Elkon `json:"elkon"`

	HTTPProgDir           string `json:"httpProdDir"`
	PHPDir                string `json:"phpDir"`
	IsClusterMasterScript string `json:"IsClusterMasterScript"`
}

func (c *AppConfig) GetDebugQueries() bool {
	return c.DebugQueries
}

func (c *AppConfig) GetDBPrimary() string {
	return c.DB.Primary
}

// ReadConf reads configiration from json file
func (c *AppConfig) ReadConf(fileName string) error {
	file, err := os.ReadFile(fileName)
	if err == nil {
		file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))
		err = json.Unmarshal([]byte(file), c)
	}

	// defaults
	if c.AppShutdownTimeout == 0 {
		c.AppShutdownTimeout = defAppShutdownTimeout
	}

	return err
}

// WriteConf writes configiration from arbitary struct to json file
// func WriteConf(fileName string, c interface{}) error {
// 	cont_b, err := json.Marshal(c)
// 	if err == nil {
// 		err = os.WriteFile(fileName, cont_b, 0644)
// 	}
// 	return err
// }
