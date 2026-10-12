package services

import (
	"context"

	"github.com/dronm/ds/v4"
)

const permReadQueryID = "permissions_read" //prepared query ID

type PermissionService struct {
	DB ds.Provider
}

func NewPermissionService(db ds.Provider) *PermissionService {
	return &PermissionService{DB: db}
}

func (s *PermissionService) ReadList(ctx context.Context) (map[string]struct{}, error) {
	return map[string]struct{}{
		"Constant.list.owner": {},
		"Constant.set.owner":  {},
		"Constant.get.owner":  {},

		"MainMenu.for_user.owner":  {},
		"MainMenu.list.owner":      {},
		"MainMenu.detail.owner":    {},
		"MainMenu.del.owner":       {},
		"MainMenu.add.owner":       {},
		"MainMenu.update.owner":    {},
		"MainMenu.move_up.owner":   {},
		"MainMenu.move_down.owner": {},

		"ElkonProduction.upload.owner":       {},
		"ElkonProduction.log.owner":          {},
		"ElkonProduction.fetch_config.owner": {},
		"ElkonProduction.get.owner":          {},
		"ElkonProduction.error.owner":        {},
		"ElkonProduction.ping.owner":         {},

		"WS.Init.owner": {},
	}, nil
	/*
		funcName := "ReadList"
		poolConn, connID, err := s.DB.GetSecondary(ctx, "")
		if err != nil {
			return fmt.Errorf("%s GetSecondary() failed: %w", funcName, err)
		}
		defer s.DB.Release(poolConn, connID)
		conn := poolConn.Conn()

		permRow := models.Permission{}
		permFields, permIds, err := MakeStructRowFields(&permRow, "")
		if err != nil {
			return fmt.Errorf("%s MakeStructRowFields() failed: %v", funcName, err)
		}
		if _, err := conn.Prepare(ctx,
			PERM_READ_Q,
			fmt.Sprintf("SELECT %s FROM permissions", permIds)); err != nil {
			return fmt.Errorf("%s conn.Prepare() for PERM_READ_Q failed: %v", funcName, err)
		}

		rows, err := conn.Query(ctx, PERM_READ_Q)
		if err != nil {
			return fmt.Errorf("conn.Query() failed: %v", err)
		}

		for rows.Next() {
			if err := rows.Scan(permFields...); err != nil {
				return fmt.Errorf("pgx.Rows.Scan() failed: %v", err)
			}
			PermissionList = append(PermissionList, permRow)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("pgx.Rows.Err() failed: %v", err)
		}
	*/
}
