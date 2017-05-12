package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	nounceSize = 12
)

type CryptoData struct {
	SymmetricKey []byte
	HMACSecret   []byte

	Salt       []byte
	Iterations int
}

func NewCryptoData(password []byte, hmacSecret []byte, salt []byte, iterations int) *CryptoData {
	symmetricKey := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)
	return &CryptoData{SymmetricKey: symmetricKey, HMACSecret: hmacSecret, Salt: salt, Iterations: iterations}
}

func RandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)

	return b, err
}

func (cryptoData *CryptoData) EncryptText(str []byte) ([]byte, error) {
	plaintext := []byte(str)
	block, err := aes.NewCipher(cryptoData.SymmetricKey)
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

func (cryptoData *CryptoData) DecryptText(str []byte) ([]byte, error) {
	nonce := str[:12]
	ciphertext := str[12:]
	block, err := aes.NewCipher(cryptoData.SymmetricKey)
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
