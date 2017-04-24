package main

import (
	"crypto/sha256"
	"fmt"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	"github.com/gin-gonic/gin"
	"gopkg.in/appleboy/gin-jwt.v2"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

var jwtMiddleware jwt.GinJWTMiddleware

func setupMiddleware(cryptoKey *crypto.CryptoKey, cloudio *cloudIO) {
	adminLoggedIn := false
	jwtMiddleware = jwt.GinJWTMiddleware{
		Realm:      "auth",
		Key:        []byte(gc.SecretKey),
		Timeout:    time.Hour * 24 * 7,
		MaxRefresh: time.Hour * 24,
		Authenticator: func(userId string, password string, c *gin.Context) (string, bool) {
			if userId == "admin" && verifyAdminPassword([]byte(password)) == nil {
				ph, err := gc.PasswordDB.GetCryptoPasswordHash()
				if err != nil {
					return userId, false
				}

				pgpKey, err := decrypt([]byte(password), ph.EncryptedPGPKey, ph.Salt, ph.Iterations)

				if err != nil {
					return userId, false
				}

				hmacSecret, err := decrypt([]byte(password), ph.EncryptedHMACSecret, ph.Salt, ph.Iterations)
				fmt.Println("plaintext hmac: ", hmacSecret)

				if err != nil {
					return userId, false
				}

				adminLoggedIn = true

				newCryptoKey := crypto.CryptoKey{Key: pgpKey, HMACSecret: hmacSecret}
				newCloudio := cloudIO{cryptoKey: newCryptoKey, storageBucket: gc.StorageBucket}

				*cryptoKey = newCryptoKey
				*cloudio = newCloudio

				return userId, true

			}
			return userId, false
		},
		Authorizator: func(userId string, c *gin.Context) bool {
			if !adminLoggedIn {
				return false
			}

			if userId == "admin" {
				return true
			}

			return false
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			c.JSON(code, gin.H{
				"code":    code,
				"message": message,
			})
		},
		TokenLookup: "cookie:jwt",
		TimeFunc:    time.Now,
	}
}

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

func encrypt(password []byte, pgpkey []byte, salt []byte, iterations int) ([]byte, error) {
	key := pbkdf2.Key(password, salt, iterations, 32, sha256.New)
	setupKey := crypto.CryptoKey{Key: key}
	fileCryptoKey, err := setupKey.EncryptText(pgpkey)
	return fileCryptoKey, err
}

func decrypt(password []byte, pgpkey []byte, salt []byte, iterations int) ([]byte, error) {
	key := pbkdf2.Key(password, salt, iterations, 32, sha256.New)
	setupKey := crypto.CryptoKey{Key: key}
	fileCryptoKey, err := setupKey.DecryptText(pgpkey)
	return fileCryptoKey, err
}
