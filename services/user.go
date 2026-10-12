package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	crud "github.com/dronm/crudifier/v2"
	fields "github.com/dronm/crudifier/v2/fields"
	crudTypes "github.com/dronm/crudifier/v2/types"
	"github.com/dronm/ds/v4"
	gmodels "github.com/dronm/gobizapp/models"
	gsrv "github.com/dronm/gobizapp/services"
	"github.com/dronm/session"

	"github.com/dronm/gobizapp/errs"

	"github.com/dronm/eurobeton/config"
	"github.com/dronm/eurobeton/models"
)

type UserService struct {
	DB      ds.Provider
	Session session.Session
}

func NewUserService(db ds.Provider, sess session.Session) *UserService {
	return &UserService{DB: db, Session: sess}
}

func (s *UserService) FetchList(ctx context.Context, params crud.CollectionParams) ([]*models.User, *gmodels.TotCount, error) {
	return gsrv.FetchCollectionModel(ctx, s.DB, &models.User{}, &gmodels.TotCount{}, params)
}

func (s *UserService) FetchDetail(ctx context.Context, id int) (*models.User, error) {
	model := models.User{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.UserKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *UserService) Delete(ctx context.Context, keyModels []models.UserKey) (int64, error) {
	models := make([]crudTypes.DBModel, len(keyModels))
	for i, m := range keyModels {
		models[i] = m
	}
	cnt, err := gsrv.DeleteModel(ctx, s.DB, models, nil)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (s *UserService) Update(ctx context.Context, keyModel models.UserKey, model models.User) (int64, error) {
	cnt, err := gsrv.UpdateModel(ctx, s.DB, keyModel, &model)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (s *UserService) Insert(ctx context.Context, model models.User) (map[string]any, error) {
	pwd := GenUserPassword()
	model.Pwd.SetValue(gsrv.GetMd5(pwd))

	retFields, err := gsrv.InsertModel(ctx, s.DB, &model, nil)
	if err != nil {
		return nil, err
	}
	return retFields, nil
}

type UserLoginInf struct {
	Headers    http.Header
	RemoteAddr string
}

type UserLoginResponse struct {
	User *models.UserLogin `json:"user"`
	Auth *models.Auth      `json:"auth"`
}

func (s *UserService) Logout(ctx context.Context) error {
	userLogin := models.UserLogin{}
	if err := s.Session.Get("user", &userLogin); err != nil {
		return fmt.Errorf("Session.Get() failed: %v", err)
	}

	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if _, err := conn.Prepare(ctx,
		"user_logout",
		`UPDATE logins 
		SET date_time_out = now() 
		WHERE id = $1`,
	); err != nil {
		return fmt.Errorf("conn.Prepare() failed: %v", err)
	}

	if _, err := conn.Exec(ctx, "BEGIN"); err != nil {
		return fmt.Errorf("conn.Exec() BEGIN failed: %v", err)
	}
	if _, err := conn.Exec(ctx, "user_logout", userLogin.LoginID); err != nil {
		_, _ = conn.Exec(ctx, "ROLLBACK")
		return fmt.Errorf("conn.Exec() failed: %v", err)
	}
	if err := s.Session.Delete(s.Session.SessionID()); err != nil {
		_, _ = conn.Exec(ctx, "ROLLBACK")
		return fmt.Errorf("Session.Delete() failed: %v", err)
	}

	if _, err := conn.Exec(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("conn.Exec() COMMIT failed: %v", err)
	}

	return nil
}

func (s *UserService) Login(ctx context.Context, userInf UserLoginInf, model models.User_login) (*UserLoginResponse, error) {
	funcName := "Login"

	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s GetPrimary() failed: %w", funcName, err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	userRow := models.UserLogin{}
	userFields, userIds, err := gsrv.MakeStructRowFields(&userRow, "")
	if err != nil {
		return nil, fmt.Errorf("%s MakeStructRowFields() failed: %v", funcName, err)
	}

	if _, err := conn.Prepare(ctx,
		USER_LOGIN_Q,
		fmt.Sprintf(`SELECT %s 
			FROM users_login 
			WHERE name = $1 AND pwd = md5($2)`,
			userIds),
	); err != nil {
		return nil, fmt.Errorf("%s conn.Prepare() for USER_LOGIN_Q failed: %v", funcName, err)
	}

	err = conn.QueryRow(ctx, USER_LOGIN_Q, model.Name, model.Pwd).Scan(userFields...)
	if err == ds.ErrNoRows {
		// no user with this name &&  pwd
		return nil, errs.NewPublicError(http.StatusForbidden, errs.UserCredIncorrect)
	} else if err != nil {
		return nil, fmt.Errorf("conn.QueryRow() form SELECT from users_login failed: %v", err)
	}

	if err := loginUser(ctx, s.Session, userInf, conn, &userRow); err != nil {
		return nil, err
	}

	tokenRefresh := ""
	tokenExpires := time.Time{}
	if config.Config.Session.MaxLifeTime > 0 {
		tokenExpires = time.Now().Add(time.Duration(config.Config.Session.MaxLifeTime) * time.Second)
	}
	return &UserLoginResponse{
		User: &userRow,
		Auth: &models.Auth{
			Token:        s.Session.SessionID(),
			TokenRefresh: tokenRefresh,
			Expires:      tokenExpires,
		},
	}, nil
}

func (s *UserService) FetchProfile(ctx context.Context, id int) (*models.UserProfile, error) {
	model := models.UserProfile{}
	if err := gsrv.FetchModel(ctx, s.DB, &models.UserKey{ID: fields.NewFieldInt(int64(id), true, false)}, &model); err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *UserService) userLoginResponse(userRow models.UserLogin) (*UserLoginResponse, error) {
	tokenRefresh := ""
	tokenExpires := time.Time{}
	if config.Config.Session.MaxLifeTime > 0 {
		tokenExpires = time.Now().Add(time.Duration(config.Config.Session.MaxLifeTime) * time.Second)
	}
	return &UserLoginResponse{
		User: &userRow,
		Auth: &models.Auth{
			Token:        s.Session.SessionID(),
			TokenRefresh: tokenRefresh,
			Expires:      tokenExpires,
		},
	}, nil
}

func (s *UserService) UpdatePwd(ctx context.Context, userID int, pwd string) error {
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	if _, err := conn.Exec(ctx, "UPDATE users SET pwd = md5($1) WHERE id = $2", pwd, userID); err != nil {
		return fmt.Errorf("conn.Exec() UpdatePwd failed: %v", err)
	}

	return nil
}

func (s *UserService) RecoverPassword(ctx context.Context, email string) error {
	// check email exists
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	var userID int
	if err := conn.QueryRow(ctx,
		"SELECT id FROM users WHERE email = $1",
		email,
	).Scan(&userID); err != nil && err != ds.ErrNoRows {
		return fmt.Errorf("conn.QueryRow() select from users failed: %v", err)
	} else if err != nil && err == ds.ErrNoRows {
		return errs.NewPublicError(http.StatusForbidden, errs.UserEmailNotFound)
	}

	newPassword := GenUserPassword()
	if err := RecoverPasswordEmail(ctx, conn, userID, config.Config.BaseURL, newPassword); err != nil {
		return fmt.Errorf("RecoverPasswordEmail() failed: %w", err)
	}

	if _, err := conn.Exec(ctx, `UPDATE users SET pwd = md5($1) WHERE id = $2`, userID, newPassword); err != nil {
		return fmt.Errorf("conn.Exec() UPDATE users pwd failed: %w", err)
	}

	return nil
}

func (s *UserService) SetDefaultPassword(ctx context.Context, userID int) error {
	// check email exists
	poolConn, connID, err := s.DB.GetPrimary(ctx)
	if err != nil {
		return fmt.Errorf("GetPrimary() failed: %w", err)
	}
	defer s.DB.Release(poolConn, connID)
	conn := poolConn.Conn()

	newPassword := GenUserPassword()
	if _, err := conn.Exec(ctx, `UPDATE users SET pwd = md5($2) WHERE id = $1`, userID, newPassword); err != nil {
		return fmt.Errorf("conn.Exec() UPDATE users pwd failed: %w", err)
	}

	return nil
}

func GenUserPassword() string {
	return "123456"
}

//
