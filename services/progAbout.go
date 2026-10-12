// Package services
package services

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/dronm/eurobeton/models"

	"github.com/dronm/gobizapp/logger"

	"github.com/dronm/ds/v4"
	"github.com/dronm/session"
)

var ProgAbout = models.ProgAbout{}
var progAboutMx sync.Mutex

type ProgAboutService struct {
	DB      ds.Provider
	Session session.Session
}

func NewProgAboutService(db ds.Provider, sess session.Session) *ProgAboutService {
	return &ProgAboutService{DB: db, Session: sess}
}

func (s *ProgAboutService) Info(ctx context.Context, primaryConn string) (*models.ProgAbout, error) {
	if ProgAbout.DB == nil {

		poolConn, connID, err := s.DB.GetSecondary(ctx, "")
		if err != nil {
			return nil, fmt.Errorf("GetSecondary() failed: %w", err)
		}
		defer s.DB.Release(poolConn, connID)
		conn := poolConn.Conn()

		var vers string
		if err := conn.QueryRow(ctx,
			`SELECT
				version(),
				(SELECT updated_at FROM deploy_updates ORDER BY updated_at DESC LIMIT 1)`,
		).Scan(&vers, &ProgAbout.UpdatedAt); err != nil {
			return nil, fmt.Errorf("conn.QueryRow().Scan() for version information: %v", err)
		}

		migVers, err := migrationVersion(primaryConn)
		if err != nil {
			return nil, fmt.Errorf("migrationVersion: %v", err)
		}

		ip, err := externalIP()
		if err != nil {
			return nil, fmt.Errorf("externalIP: %v", err)
		}

		progAboutMx.Lock()
		ProgAbout.Backend.IP = ip

		ProgAbout.DB = &models.ProgAboutDB{ServerType: "Postgresql",
			ServerVersion: vers,
			Migration:     migVers,
		}
		progAboutMx.Unlock()
	}

	return &ProgAbout, nil
}

func migrationVersion(connStr string) (string, error) {
	cmd := exec.Command(
		"migrate",
		"-database", connStr+"?sslmode=disable",
		"-path", "migrations",
		"version",
	)

	// Run and capture stdout
	out, err := cmd.Output()
	if err != nil {
		logger.Logger.Errorf("migrationVersion(): %v", err)
		// If command fails, capture stderr too
		if exitErr, ok := err.(*exec.ExitError); ok {
			logger.Logger.Errorf("stderr: %s", string(exitErr.Stderr))
		}
		return "error", nil
	}

	return string(out), nil
}

func externalIP() (string, error) {
	cmd := exec.Command("curl", "-s", "ifconfig.me")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
