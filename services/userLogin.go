package services

import (
	"context"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	crud "github.com/dronm/crudifier/v2"
	"github.com/dronm/ds/v4"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/logger"

	"github.com/dronm/eurobeton/models"
)

const (
	USER_LOGIN_Q         = "users_login_q"
	USER_LOGIN_ID_Q      = "users_login_id_q"
	USER_ADD_Q           = "users_add_q"
	USER_LOGIN_PUB_KEY_Q = "users_login_pub_key"
	USER_LOGIN_UPDATE_Q  = "users_login_update"
	USER_LOGIN_INSERT_Q  = "users_login_insert"
	USER_LOGIN_CLOSE_Q   = "users_login_close"
	USER_LOGIN_TOKEN_Q   = "users_login_token"
	USER_RESET_PWD_Q     = "users_reset_pwd"

	PUB_KEY_LEN = 15
)

func loginUser(ctx context.Context, sess session.Session, userInf UserLoginInf, conn ds.Conn, userRow *models.UserLogin) error {
	if err := prepareUserLoginSQL(ctx, conn); err != nil {
		return fmt.Errorf("sprepareUserLoginSQL() failed: %v", err)
	}

	if userRow.Banned {
		return crud.NewValidationError(ER_USER_BANNED)
	}

	sessID := sess.SessionID()

	// user agent
	var err error
	var userHeaders []byte
	userAgentField := []byte("{}")

	userIP := userInf.RemoteAddr
	userIPPortPos := strings.Index(userIP, ":")
	if userIPPortPos >= 0 {
		userIP = userIP[:userIPPortPos]
	}
	userHeaders, err = json.Marshal(userInf.Headers)
	if err != nil {
		logger.Logger.Errorf("json.Marshal() failed: %v", err)
	}

	userAgent := userInf.Headers.Get("User-Agent")
	if userAgent != "" {
		userAgentField, err = userAgentFieldValue(userAgent)
		if err != nil {
			logger.Logger.Errorf("userAgentFieldValue() failed: %v", err)
		}
	}

	if userRow.RoleID != "admin" {
		deviceHash := ""
		err := conn.QueryRow(ctx,
			`SELECT md5(login_devices_uniq($1::jsonb)) AS hash`,
			string(userAgentField)).Scan(&deviceHash)
		if err != nil {
			// just log
			logger.Logger.Errorf("login_devices_uniq() failed: %v", err)
		} else {
			if userRow.BanHash != nil && strings.Contains(*userRow.BanHash, deviceHash) {
				return crud.NewValidationError("user device is banned")
			}

			// No banned. New device? было ло ли в logins такое устройство?
			var timeIn time.Time
			err := conn.QueryRow(ctx,
				`SELECT
					date_time_in
				FROM login_devices_list
				WHERE user_id = $1 AND ban_hash = $2`,
				userRow.ID, deviceHash).Scan(&timeIn)

			if err != nil && err != ds.ErrNoRows {
				return fmt.Errorf("first select from login_devices_list failed: %v", err)
			}

			/*
				if err == ds.ErrNoRows || date_time_in.GetIsNull() {
					//first login - ban!
					if _, err := conn.Exec(ctx,
						`INSERT INTO login_device_bans
							(user_id, hash) VALUES ($1, $2) ON CONFLICT (user_id, hash) DO NOTHING`,
						user_id, device_hash); err != nil {
							return "", gobizap.NewPublicMethodError(response.RESP_ER_INTERNAL, fmt.Sprintf("doLoginUser first login ban pgx.Conn.Exec(): %v",err))
					}
					if _, err := conn.Exec(ctx,
						`INSERT INTO logins
							(date_time_in, date_time_out, ip, session_id, pub_key, user_id, headers, user_agent)
							VALUES(now(), now(), $1, $2, NULL, $3, $4::json, $5::jsonb)`,
						user_ip, sessID,
						user_id, string(userHeaders), string(user_a_s)); err != nil {
							return "", gobizap.NewPublicMethodError(response.RESP_ER_INTERNAL, fmt.Sprintf("doLoginUser first login ban pgx.Conn.Exec(): %v",err))
					}
					return "", gobizap.NewPublicMethodError(RESP_ER_BANNED_DEV_CODE, RESP_ER_BANNED_DEV_DESCR)
				}
			*/
		}
	}

	err = conn.QueryRow(ctx, USER_LOGIN_PUB_KEY_Q, sessID, userRow.ID).Scan(&userRow.PubKey, &userRow.LoginID)

	if err == ds.ErrNoRows {
		// no user login
		userRow.PubKey, err = genPublicKey()
		if err != nil {
			return fmt.Errorf("genPublicKey() failed: %v", err)
		}

		err := conn.QueryRow(ctx, USER_LOGIN_UPDATE_Q,
			userRow.ID,
			userRow.PubKey,
			string(userAgentField), userIP, string(userHeaders),
			sessID,
		).Scan(&userRow.LoginID)

		// fmt.Println("userIP:", userIP)
		// fmt.Println("userPubKey:", userRow.PubKey)
		if err == ds.ErrNoRows {
			// no user
			if err = conn.QueryRow(ctx,
				USER_LOGIN_INSERT_Q,
				userIP,
				sessID,
				userRow.PubKey,
				userRow.ID,
				string(userAgentField),
				string(userHeaders),
			).Scan(&userRow.LoginID); err != nil {
				return fmt.Errorf("users_login_insert pgx.Rows.Scan(): %v", err)
			}
		} else if err != nil {
			return fmt.Errorf("users_login_update pgx.Rows.Scan(): %v", err)
		}

	} else if err != nil {
		return fmt.Errorf("USER_LOGIN_PUB_KEY_Q pgx.Rows.Scan(): %v", err)
	}

	// Session data
	gob.Register(models.UserLogin{})
	if err := sess.Set("user", *userRow); err != nil {
		return fmt.Errorf("sess.Set('user') failed: %v", err)
	}

	if err := sess.Flush(); err != nil {
		return fmt.Errorf("sess.Flush() failed: %v", err)
	}
	//******* Preset filters for all users*******************

	return nil
}

func prepareUserLoginSQL(ctx context.Context, conn ds.Conn) error {
	if _, err := conn.Prepare(ctx, USER_LOGIN_PUB_KEY_Q,
		`SELECT pub_key, id
		FROM logins
		WHERE session_id=$1 AND user_id=$2 AND date_time_out IS NULL`); err != nil {
		return fmt.Errorf("USER_LOGIN_PUB_KEY_Q pgx.Conn.Prepare(): %v", err)
	}

	if _, err := conn.Prepare(ctx,
		USER_LOGIN_UPDATE_Q,
		`UPDATE logins
		SET
			user_id = $1,
			pub_key = $2,
			date_time_in = now(),
			set_date_time = now(),
			user_agent = $3::jsonb,
			ip = $4,
			headers = $5::json
			FROM (
				SELECT
					l.id AS id
				FROM logins l
				WHERE l.session_id=$6 AND l.user_id IS NULL
				ORDER BY l.date_time_in DESC
				LIMIT 1										
			) AS s
			WHERE s.id = logins.id
		RETURNING logins.id`); err != nil {
		return fmt.Errorf("USER_LOGIN_UPDATE_Q pgx.Conn.Prepare(): %v", err)
	}

	if _, err := conn.Prepare(ctx,
		USER_LOGIN_INSERT_Q,
		`INSERT INTO logins
		(date_time_in, ip, session_id, pub_key, user_id, user_agent, headers)
		VALUES (now(), $1, $2, $3, $4, $5, $6)				
		RETURNING id`); err != nil {
		return fmt.Errorf("USER_LOGIN_INSERT_Q pgx.Conn.Prepare(): %v", err)
	}

	return nil
}

func genPublicKey() (string, error) {
	sessionID := make([]byte, PUB_KEY_LEN/2)
	_, err := rand.Read(sessionID)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sessionID), nil
}
