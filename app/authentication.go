package main

import (
	"crypto/sha256"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

func generatePasswordHash(password []byte) ([]byte, error) {
	if p, err := bcrypt.GenerateFromPassword(password, 4); err != nil {
		return nil, err
	} else {
		return p, err
	}
}

func verifyAdminPassword(plainTextPassword []byte) error {
	ph, err := gc.PasswordDB.GetCryptoPasswordHash()

	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword(ph.Hash, plainTextPassword); err != nil {
		return err
	}
	return nil
}

func passwordToCryptoKey(password []byte, salt []byte, iterations int) ([]byte, error) {
	key := pbkdf2.Key(password, salt, iterations, 32, sha256.New)
	return key, nil
}
