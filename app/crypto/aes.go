package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

const (
	nounceSize = 12
)

type CryptoKey struct {
	Key        []byte
	HMACSecret []byte
}

func RandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)

	return b, err
}

func (ac *CryptoKey) EncryptText(str []byte) ([]byte, error) {
	plaintext := []byte(str)

	block, err := aes.NewCipher(ac.Key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, nounceSize)

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	ivCiphertext := append(nonce, ciphertext...)

	return ivCiphertext, nil
}

func (ac *CryptoKey) DecryptText(str []byte) ([]byte, error) {
	nonce := str[:12]
	ciphertext := str[12:]

	block, err := aes.NewCipher(ac.Key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
