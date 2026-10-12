package models

import (
	fields "github.com/dronm/crudifier/v2/fields"
	gmodels "github.com/dronm/gobizapp/models"
)

const (
	userRelation        = "users"
	userProfileRelation = "users_profile"
)

type User_login struct {
	Name fields.FieldText `json:"name" required:"true" maxLen:"50"`
	Pwd  fields.FieldText `json:"pwd" required:"true" maxLen:"50"`
}

type User struct {
	ID     fields.FieldInt  `json:"id" primaryKey:"true" srvCalc:"true"`
	Name   fields.FieldText `json:"name" required:"true" maxLen:"100"`
	RoleID fields.FieldText `json:"role_id" enum:"role_id"`
	Pwd    fields.FieldText `json:"pwd"`
}

func (m User) Relation() string {
	return userRelation
}

func (m User) CollectionAgg() any {
	return &gmodels.TotCount{}
}

type UserKey struct {
	ID fields.FieldInt `json:"id" required:"true"`
}

func (m UserKey) Relation() string {
	return userRelation
}

type UserUpdate struct {
	Keys  UserKey `json:"keys"`
	Model User    `json:"model"`
}

type UserProfileUpdate struct {
	Name fields.FieldText `json:"name" required:"true" maxLen:"100"`
}

type UserProfile struct {
	Name string `json:"name" required:"true" maxLen:"100"`
}

func (m UserProfile) Relation() string {
	return userProfileRelation
}

type User_update_password struct {
	Pwd fields.FieldText `json:"pwd" required:"true" maxLen:"50"`
}

// type User_recover_password struct {
// 	Email fields.FieldText `json:"email" required:"true" maxLen:"50"`
// }

type User_set_default_password struct {
	Id fields.FieldInt `json:"id" required:"true"`
}
