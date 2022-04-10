package common

import "errors"

var (
	ErrUnauthenticated      = errors.New("err user failed to authenticate")
	ErrInvalidSigningMethod = errors.New("err invalid signing method")
	ErrInvalidAccessToken   = errors.New("err invalid access token")
	ErrInvalidPhoneNumber   = errors.New("err invalid phone number")
	ErrPhoneNotFound        = errors.New("err phone not found")
)
