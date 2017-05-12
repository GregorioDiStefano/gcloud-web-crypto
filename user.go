package gscrypto

import "time"

type UserEntry struct {
	Username            string
	Email               string
	Admin               bool
	Enabled             bool
	CreatedDate         time.Time
	Hash                []byte
	EncryptedPGPKey     []byte
	EncryptedHMACSecret []byte
	Salt                []byte
	Iterations          int
}

type UserDatabase interface {
	SetUserEntry(*UserEntry) error
	GetUserEntry(string) (*UserEntry, int64, error)
	GetUsers() ([]*UserEntry, error)
	UpdateUser(int64, *UserEntry) error
}
