package gscrypto

import "time"

type PasswordHash struct {
	CreatedDate     time.Time
	Hash            []byte
	EncryptedPGPKey []byte
	Salt            []byte
	Iterations      int
}

type PasswordDatabase interface {
	SetCryptoPasswordHash(*PasswordHash) error
	GetCryptoPasswordHash() (*PasswordHash, error)
}
