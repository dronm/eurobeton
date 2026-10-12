package models

import (
	"time"
)

type UserLogin struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	RoleID           string    `json:"role_id"`
	CreateDt         time.Time `json:"create_dt"`
	Pwd              string    `f:"pwd" json:"-"`
	Banned           bool      `f:"banned" json:"-"`
	PubKey           string    `f:"-" json:"pub_key"`
	LoginID          int       `f:"-" json:"-"`
	BanHash          *string   `f:"ban_hash" json:"-"`
}
