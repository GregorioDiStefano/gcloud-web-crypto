package main

import (
	"net/http"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	"github.com/gin-gonic/gin"
	cache "github.com/robfig/go-cache"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/appleboy/gin-jwt.v2"
)

var jwtMiddleware jwt.GinJWTMiddleware

const (
	tokenTTL    = time.Hour * 24 * 7
	notVerified = "user not verified"
)

func setupMiddleware(memoryStore *cache.Cache) {
	jwtMiddleware = jwt.GinJWTMiddleware{
		Realm:   "auth",
		Key:     []byte(gc.SecretKey),
		Timeout: tokenTTL,

		Authenticator: func(userId string, password string, context *gin.Context) (string, bool) {
			if verifyUserPassword(userId, []byte(password)) == nil {
				user, _, err := gc.UserDB.GetUserEntry(userId)
				if err != nil {
					return userId, false
				}

				if user.Enabled == false {
					context.Set("reason", notVerified)
					return userId, false
				}

				c := crypto.NewCryptoData([]byte(password), nil, user.Salt, user.Iterations)
				pgpKey, err := c.DecryptText(user.EncryptedPGPKey)

				if err != nil {
					return userId, false
				}

				hmacSecret, err := c.DecryptText(user.EncryptedHMACSecret)

				if err != nil {
					return userId, false
				}

				// once a user logs in, story credentials in memory, and expire when token expires.
				userCrypto := crypto.NewCryptoData(pgpKey, hmacSecret, user.Salt, user.Iterations)
				userCloudIO := userData{cryptoData: *userCrypto, storageBucket: gc.StorageBucket, userEntry: *user}
				memoryStore.Add(userId, userCloudIO, tokenTTL)

				return userId, true

			}
			return userId, false
		},
		Authorizator: func(userId string, c *gin.Context) bool {
			if user, exists := memoryStore.Get(userId); exists == true {
				c.Set("user", user.(userData))
			}

			return true
		},
		Unauthorized: func(c *gin.Context, code int, message string) {
			if reason, exists := c.Get("reason"); exists && reason == notVerified {
				c.JSON(http.StatusUnauthorized, gin.H{"message": "account is not verified"})
			} else {
				c.JSON(code, gin.H{
					"code":    code,
					"message": message,
				})
			}
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

func verifyUserPassword(username string, plainTextPassword []byte) error {
	ph, _, err := gc.UserDB.GetUserEntry(username)

	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword(ph.Hash, plainTextPassword); err != nil {
		return err
	}
	return nil
}
