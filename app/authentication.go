package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
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
	tokenTTL                 = time.Hour * 24 * 7
	userNotInContext         = "user not found in context"
	notVerified              = "user not verified"
	needCaptcha              = "need valid captcha"
	memoryStoreLogFailPrefix = "failed_login_"
)

func setupMiddleware(memoryStore *cache.Cache) {
	jwtMiddleware = jwt.GinJWTMiddleware{
		Realm:      "auth",
		Key:        []byte(gc.SecretKey),
		Timeout:    tokenTTL,
		MaxRefresh: tokenTTL,
		Authenticator: func(userId string, password string, context *gin.Context) (string, bool) {
			if data, exists := memoryStore.Get(memoryStoreLogFailPrefix + userId); exists && data.(bool) {
				captcha := context.Request.Header.Get("google-captcha")

				if len(captcha) == 0 {
					context.Set("reason", needCaptcha)
					return userId, false
				}

				if ok, err := verifyGoogleCaptcha(captcha); err != nil || !ok {
					return userId, false
				}
			}

			if len(userId) > 0 && len(password) > 0 && verifyUserPassword(userId, []byte(password)) == nil {
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
				userCloudIO := userData{cryptoData: *userCrypto, userEntry: *user}
				memoryStore.Add(userId, userCloudIO, tokenTTL)

				memoryStore.Delete(memoryStoreLogFailPrefix + userId)
				return userId, true
			}

			// failed to login, store failed login attempt in memory cache
			if !gin.IsDebugging() {
				memoryStore.Set(memoryStoreLogFailPrefix+userId, true, time.Minute*10)
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
			} else if exists && reason == needCaptcha {
				c.JSON(http.StatusBadRequest, gin.H{"message": "captcha required"})
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

func verifyGoogleCaptcha(response string) (bool, error) {

	type googleResponse struct {
		Success    bool
		ErrorCodes []string `json:"error-codes"`
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.PostForm(config.googleCaptchaURL,
		url.Values{"secret": {config.googleCaptchaSecret}, "response": {response}})

	if err != nil {
		return false, err
	}

	gr := new(googleResponse)
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return false, err
	}

	err = json.Unmarshal(body, gr)

	if err != nil {
		return false, err
	}

	if !gr.Success {
		return false, err
	}

	return true, nil
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
