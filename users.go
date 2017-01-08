package gscrypto

import "time"

type UserCredentials struct {
	CreatedDate time.Time
	Username    string
}

type UserCredentialsDatabase interface {
	SetUserCreds(*UserCredentials) error
	GetUserCreds(string) (*UserCredentials, error)
}
