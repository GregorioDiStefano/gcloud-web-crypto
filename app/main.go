package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	crypto "github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	zxcvbn "github.com/nbutton23/zxcvbn-go"

	"github.com/gin-gonic/gin"
)

const (
	AUTH_ENABLED = true
	DEV_MODE     = true
)

func main() {
	cryptoKey := &crypto.CryptoKey{}
	cloudio := &cloudIO{}

	r := gin.Default()
	private := r.Group("/auth")

	setupMiddleware(cryptoKey, cloudio)

	if AUTH_ENABLED {
		private.Use(jwtMiddleware.MiddlewareFunc())
	}

	r.POST("/account/login", jwtMiddleware.LoginHandler)

	r.GET("/account/status", func(c *gin.Context) {
		// send 404 if no accounts exists else 204
		if passwordData, _ := gc.PasswordDB.GetCryptoPasswordHash(); passwordData != nil {
			c.Status(http.StatusNoContent)
			return
		}
		c.Status(http.StatusNotFound)
	})

	r.POST("/account/signup", func(c *gin.Context) {
		type signup struct {
			Password string `form:"password" json:"password"`
		}

		if passwordData, _ := gc.PasswordDB.GetCryptoPasswordHash(); passwordData != nil {
			c.JSON(409, gin.H{"status": "account already exists"})
			return
		}

		var signupRequest signup
		if err := c.BindJSON(&signupRequest); err != nil {
			c.JSON(401, gin.H{"status": "unauthorized"})
			return
		}

		password := signupRequest.Password
		passwordInfo := zxcvbn.PasswordStrength(string(password), []string{})

		if passwordInfo.Score < 3 {
			c.JSON(401, gin.H{"status": "the password you picked isn't secure enough."})
			return
		}

		iterations := 100000
		salt := make([]byte, 32)
		rand.Read(salt)

		newPasswordHash, err := generatePasswordHash([]byte(password))

		if err != nil {
			panic(err)
		}

		pgpKey, _ := crypto.RandomBytes(32)
		hmacSecret, _ := crypto.RandomBytes(32)

		fmt.Println("creating hmac secret: ", hmacSecret)

		encryptedPGPKey, err := encrypt([]byte(password), pgpKey, salt, iterations)
		encryptedHMACSecret, err := encrypt([]byte(password), hmacSecret, salt, iterations)

		passwordHash := &gc.PasswordHash{
			CreatedDate:         time.Now(),
			Hash:                newPasswordHash,
			EncryptedPGPKey:     encryptedPGPKey,
			EncryptedHMACSecret: encryptedHMACSecret,
			Iterations:          iterations,
			Salt:                salt,
		}

		gc.PasswordDB.SetCryptoPasswordHash(passwordHash)
	})

	private.POST("/file/", func(c *gin.Context) {
		err := cloudio.processFileUpload(c)

		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"fail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, map[string]string{"upload": "success"})
	})

	private.DELETE("/file/:uuid", func(c *gin.Context) {

		id, err := strconv.ParseInt(c.Param("uuid"), 10, 64)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		err = cloudio.deleteFile(id)

		if err != nil {
			fmt.Println(err)
		}
	})

	private.DELETE("/folder", func(c *gin.Context) {
		folderDeletePath := c.Query("path")
		err := cloudio.deleteFolder(folderDeletePath)

		if err != nil {
			c.JSON(http.StatusForbidden, err.Error())
			return
		}
		_, folderID, err := gc.FileStructDB.ListFolders(folderDeletePath)
		gc.FileStructDB.DeleteFolder(folderID)

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

		if fs, err := cloudio.listFileSystem(path); err != nil {
			c.JSON(http.StatusInternalServerError, err)
		} else {
			c.IndentedJSON(http.StatusOK, fs)
		}
	})

	private.GET("/folder", func(c *gin.Context) {
		path := c.Query("path")
		path = filepath.Clean(path)

		if err := cloudio.downloadFolder(*c, path); err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	})

	private.GET("/file/:key", func(c *gin.Context) {
		key := c.Param("key")
		id, err := strconv.ParseInt(key, 10, 64)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}

		if err := cloudio.downloadFile(c, id); err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}
	})

	r.Run(":3000")
}
