package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	crypto "github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"

	cache "github.com/robfig/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

type userData struct {
	userEntry     gc.UserEntry
	cryptoData    crypto.CryptoData
	storageBucket *storage.BucketHandle
}

func init() {
	log.SetLevel(log.DebugLevel)
}

func getUserFromContext(c *gin.Context) userData {
	user, exists := c.Get("user")

	log.WithFields(log.Fields{"user": user.(userData).userEntry.Username}).Debug("got user from context")

	if exists == false {
		log.WithFields(log.Fields{"user": user.(userData).userEntry.Username}).Debug("user missing from context")
		c.JSON(http.StatusUnauthorized, "user context is missing")
	}

	return user.(userData)
}

func mainGinEngine() *gin.Engine {
	memoryStore := cache.New(tokenTTL, time.Minute*5)

	router := gin.Default()
	private := router.Group("/auth")

	setupMiddleware(memoryStore)
	private.Use(jwtMiddleware.MiddlewareFunc())

	router.POST("/account/login", jwtMiddleware.LoginHandler)

	private.GET("/account/users", func(c *gin.Context) {
		user := getUserFromContext(c)
		if user.userEntry.Admin {
			if users, err := gc.UserDB.GetUsers(); err != nil {
				log.Warn("failed to retrieve users")
				c.JSON(http.StatusInternalServerError, gin.H{"status": "failed to get users"})
			} else {
				c.JSON(http.StatusOK, users)
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"status": "only admin user can get get user data"})
		}
		return
	})

	private.GET("/account/stat", func(c *gin.Context) {
		user := getUserFromContext(c)
		stats, err := user.getUserStats()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "unable to get file stats: " + err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"stats": *stats})
		}
	})

	private.PUT("/account/enable/:user", func(c *gin.Context) {
		user := getUserFromContext(c)
		if user.userEntry.Admin {
			userToEnable := c.Param("user")
			if user, id, err := gc.UserDB.GetUserEntry(userToEnable); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "failed to get users"})
				return
			} else {
				user.Enabled = true
				gc.UserDB.UpdateUser(id, user)
				c.Status(http.StatusNoContent)
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"status": "only admin user can get get user data"})
		}
		return
	})

	private.DELETE("/account/enable/:user", func(c *gin.Context) {
		user := getUserFromContext(c)
		if user.userEntry.Admin {
			userToDisable := c.Param("user")
			if user, id, err := gc.UserDB.GetUserEntry(userToDisable); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"status": "failed to get users"})
			} else {
				user.Enabled = false
				gc.UserDB.UpdateUser(id, user)
				c.Status(http.StatusNoContent)
			}
		} else {
			c.JSON(http.StatusForbidden, gin.H{"status": "only admin user can get get user data"})
		}
		return
	})

	// send 404 if no admin accounts exists else 204
	router.GET("/account/initial", func(c *gin.Context) {
		if passwordData, _, _ := gc.UserDB.GetUserEntry("admin"); passwordData != nil {
			c.Status(http.StatusNoContent)
			return
		}
		c.Status(http.StatusNotFound)
	})

	router.POST("/account/signup", func(c *gin.Context) {
		type signup struct {
			Password string `form:"password" json:"password"`
			Username string `form:"username" json:"username"`
			Email    string `form:"email" json:"email"`
		}
		var signupRequest signup

		if err := c.BindJSON(&signupRequest); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			return
		}

		// if `admin` user doesn't exit, it must be created before creating non-admin users.
		if _, _, err := gc.UserDB.GetUserEntry("admin"); err != nil && err.Error() == gc.ErrorNoDatabaseEntryFound {
			if signupRequest.Username != "admin" {
				log.WithFields(log.Fields{"user": signupRequest.Username, "password": signupRequest.Password}).Debug("attempted signup before 'admin' exists")
				c.JSON(http.StatusForbidden, gin.H{"status": "create 'admin' user first"})
				return
			}
		}

		if passwordData, _, _ := gc.UserDB.GetUserEntry(signupRequest.Username); passwordData != nil {
			c.JSON(http.StatusConflict, gin.H{"status": "account already exists"})
			return
		}

		password := signupRequest.Password

		if len(password) < 8 || !strings.ContainsAny(password, "!@#$%^&*()123456789") {
			c.JSON(http.StatusUnauthorized, gin.H{"status": "the password you picked isn't secure enough."})
			return
		}

		iterations := 10000
		salt := make([]byte, 32)
		rand.Read(salt)

		passwordHash, err := generatePasswordHash([]byte(password))
		log.WithFields(log.Fields{"password_hash": base64.StdEncoding.EncodeToString(passwordHash)}).Debug("password hash created")
		if err != nil {
			panic(err)
		}

		pgpKey, err := crypto.RandomBytes(32)

		if err != nil {
			panic(err)
		}

		hmacSecret, err := crypto.RandomBytes(64)

		if err != nil {
			panic(err)
		}

		log.WithFields(log.Fields{"pgpkey": base64.StdEncoding.EncodeToString(pgpKey), "hmacsecret": base64.StdEncoding.EncodeToString(hmacSecret)}).Debug("keys created")

		cryptoKey := crypto.NewCryptoData([]byte(password), hmacSecret, salt, iterations)
		encryptedPGPKey, err := cryptoKey.EncryptText(pgpKey)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to encrypt pgp key: " + err.Error()})
			return
		}

		encryptedHMACSecret, err := cryptoKey.EncryptText(hmacSecret)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to encrypt hmac secret"})
			return
		}

		log.WithFields(log.Fields{"encrypted pgpkey": base64.StdEncoding.EncodeToString(pgpKey),
			"encrypted hmacsecret": base64.StdEncoding.EncodeToString(encryptedHMACSecret)}).Debug("keys encrypted")

		userEntry := &gc.UserEntry{
			Username: signupRequest.Username,
			Admin:    signupRequest.Username == "admin",
			Email:    signupRequest.Email,

			// by default, all account are disabled unless the user is an admin
			Enabled:             signupRequest.Username == "admin",
			CreatedDate:         time.Now(),
			Hash:                passwordHash,
			EncryptedPGPKey:     encryptedPGPKey,
			EncryptedHMACSecret: encryptedHMACSecret,
			Iterations:          iterations,
			Salt:                salt,
		}

		err = gc.UserDB.SetUserEntry(userEntry)
		if err == nil {
			log.WithFields(log.Fields{"user": userEntry.Username}).Debug("user created successfully")
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusCreated)
	})

	private.POST("/file/", func(c *gin.Context) {
		user := getUserFromContext(c)
		err := user.processFileUpload(c)

		// send resource conflict on duplicate file
		if err != nil {
			if err.Error() == errorFileIsDuplicate {
				c.JSON(http.StatusConflict, err.Error())
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"fail": err.Error()})
			}
			return
		}

		c.Status(http.StatusCreated)
	})

	private.DELETE("/file/:uuid", func(c *gin.Context) {
		user := getUserFromContext(c)
		id, err := strconv.ParseInt(c.Param("uuid"), 10, 64)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		err = user.deleteFile(id)

		if err != nil {
			switch err.Error() {
			case gc.ErrorNotRequestingUsers:
				c.JSON(http.StatusUnauthorized, err.Error())
			default:
				c.JSON(http.StatusInternalServerError, err.Error())
			}
		}
	})

	private.DELETE("/folder", func(c *gin.Context) {
		user := getUserFromContext(c)
		folderDeletePath := c.Query("path")
		err := user.deleteFolder(folderDeletePath)

		if err != nil {
			c.JSON(http.StatusForbidden, err.Error())
			return
		}
		_, folderID, err := gc.FileStructDB.ListFolders(user.userEntry.Username, folderDeletePath)
		gc.FileStructDB.DeleteFolder(user.userEntry.Username, folderID)

		if err != nil {
			c.JSON(http.StatusForbidden, err.Error())
		}
	})

	private.GET("/list/tags/", func(c *gin.Context) {
		tags, err := gc.FileStructDB.ListTags()

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, tags)
	})

	private.GET("/list/fs", func(c *gin.Context) {
		user := getUserFromContext(c)
		path := c.Query("path")
		tags := c.Query("tags")

		if len(path) == 0 && len(tags) == 0 {
			c.JSON(http.StatusInternalServerError, fmt.Errorf("path/tag is missing"))
			return
		} else if len(path) > 0 && len(tags) > 0 {
			c.JSON(http.StatusInternalServerError, fmt.Errorf("path and tags included, only use one paramter"))
			return
		}

		if len(tags) > 0 {
			trimmedTags := []string{}
			for _, tag := range strings.Split(tags, ",") {
				trimmedTags = append(trimmedTags, strings.TrimSpace(tag))
			}

			/*		if fs, err := listFilesWithTags(trimmedTags); err != nil {
						c.JSON(http.StatusInternalServerError, err)
					} else {
						c.JSON(http.StatusOK, fs)
					}
			*/
		}

		if fs, err := user.listFileSystem(path); err != nil {
			c.JSON(http.StatusInternalServerError, err)
		} else {
			c.IndentedJSON(http.StatusOK, fs)
		}
	})

	private.GET("/folder", func(c *gin.Context) {
		user := getUserFromContext(c)
		path := c.Query("path")
		path = filepath.Clean(path)

		if err := user.downloadFolder(*c, path); err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	})

	private.GET("/file/:key", func(c *gin.Context) {
		user := getUserFromContext(c)
		key := c.Param("key")
		id, err := strconv.ParseInt(key, 10, 64)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		if err := user.downloadFile(c, id); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	})

	return router
}

func main() {
	mainGinEngine().Run(":3000")
}
