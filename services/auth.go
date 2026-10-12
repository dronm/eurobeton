package services

import "fmt"

type AuthError struct {
	Err string
}

func NewAuthError(s string) *AuthError {
	fmt.Println("NewAuthError is called with text:", s)
	return &AuthError{Err: s}
}

func (e *AuthError) Error() string {
	return e.Err
}
