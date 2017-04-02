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
	"gopkg.in/appleboy/gin-jwt.v2"

	"github.com/gin-gonic/gin"
)

const (
	AUTH_ENABLED = true
	DEV_MODE     = true
)

func main() {
	r := gin.Default()

	adminLoggedIn := false
	private := r.Group("/auth")
	cryptoKey := new(crypto.CryptoKey)
	cloudio := new(cloudIO)

	authMiddleware := &jwt.GinJWTMiddleware{
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

				if password, err := passwordToCryptoKey([]byte(password), ph.Salt, ph.Iterations); err != nil {
					return userId, false
				} else {
					adminLoggedIn = true
					setupKey := crypto.CryptoKey{Key: password}
					fileCryptoKey, err := setupKey.DecryptText(ph.EncryptedPGPKey)
					fmt.Println(fileCryptoKey, err)
					cryptoKey = &crypto.CryptoKey{Key: fileCryptoKey}
					cloudio = &cloudIO{cryptoKey: *cryptoKey, storageBucket: gc.StorageBucket}
					return userId, true
				}
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

	if AUTH_ENABLED {
		private.Use(authMiddleware.MiddlewareFunc())
	}

	r.POST("/account/login", authMiddleware.LoginHandler)

	r.POST("/account/signup", func(c *gin.Context) {
		type signup struct {
			Password string `form:"password" json:"password"`
		}

		var signupRequest signup
		if err := c.BindJSON(&signupRequest); err != nil {
			c.JSON(401, gin.H{"status": "unauthorized"})
			return
		}

		password := signupRequest.Password
		passwordInfo := zxcvbn.PasswordStrength(string(password), []string{})

		if passwordInfo.Score < 3 {
			panic("The password you picked isn't secure enough.")
		}

		iterations := 50000
		salt := make([]byte, 32)
		rand.Read(salt)

		newPasswordHash, err := generatePasswordHash([]byte(password))

		if err != nil {
			panic(err)
		}

		pgpKey, _ := crypto.RandomBytes(32)
		encryptionPassword, err := passwordToCryptoKey([]byte(password), salt, iterations)
		fmt.Println(err)
		cr := crypto.CryptoKey{Key: []byte(encryptionPassword)}

		encryptedPGPKey, err := cr.EncryptText(pgpKey)
		fmt.Println(err)
		passwordHash := &gc.PasswordHash{
			CreatedDate:     time.Now(),
			Hash:            newPasswordHash,
			EncryptedPGPKey: encryptedPGPKey,
			Iterations:      iterations,
			Salt:            salt,
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
