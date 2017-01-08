package gscrypto

import "time"

type PasswordHash struct {
	CreatedDate time.Time
	Hash        []byte
	Salt        []byte
	Iterations  int
}

// BookDatabase provides thread-safe access to a database of books.
type PasswordDatabase interface {
	SetCryptoPasswordHash(*PasswordHash) error
	GetCryptoPasswordHash() (*PasswordHash, error)
}
