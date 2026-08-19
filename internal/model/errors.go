package model

import "errors"

var (
	ErrDatabase             = errors.New("database error")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserEmptyCredentials = errors.New("empty credentials")
	ErrTokenNotProvided     = errors.New("token not provided")
	ErrTokenInvalid         = errors.New("token invalid")
)
