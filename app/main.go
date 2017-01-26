package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	crypto "github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	jwt_lib "github.com/dgrijalva/jwt-go"

	"github.com/gin-gonic/contrib/jwt"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

const (
	AUTH_ENABLED = true
	DEV_MODE     = true
)

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	private := r.Group("/auth")

	cryptoKey := crypto.CryptoKey{Key: gc.Password}
	fmt.Println("cryptoKey: ", cryptoKey)
	cloudio := cloudIO{cryptoKey: cryptoKey, storageBucket: gc.StorageBucket}

	if AUTH_ENABLED {
		private.Use(jwt.Auth(gc.SecretKey))
	}

	store := sessions.NewCookieStore([]byte(gc.SecretKey))
	r.Use(sessions.Sessions("session", store))

	private.POST("/create", func(c *gin.Context) {
	})

	r.GET("/", func(c *gin.Context) {
		if _, err := gc.UserCreds.GetUserCreds("admin"); err != nil && err.Error() == gc.ErrorNoDatabaseEntryFound {
			token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))
			token.Claims = jwt_lib.MapClaims{
				"status": "create_admin",
				"exp":    time.Now().Add(time.Hour * 1).Unix(),
			}
			c.HTML(http.StatusOK, "admin_account_creation.html", gin.H{})
		}
	})

	r.GET("/login", func(c *gin.Context) {

		token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))

		token.Claims = jwt_lib.MapClaims{
			"id":  "admin",
			"exp": time.Now().Add(time.Hour * 24 * 3).Unix(),
		}

		tokenString, err := token.SignedString([]byte("a"))
		if err != nil {
			c.JSON(500, gin.H{"message": "Could not generate token"})
		}
		c.JSON(200, gin.H{"token": tokenString})
	})

	r.OPTIONS("/file/", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET")
	})

	r.POST("/file/", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET")
		err := cloudio.processFileUpload(c)

		if err != nil {
			c.JSON(http.StatusInternalServerError, map[string]string{"fail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, map[string]string{"upload": "success"})
	})

	r.OPTIONS("/file/:uuid", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE")
	})

	r.DELETE("/file/:uuid", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE")

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

	r.OPTIONS("/folder", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE")
	})

	r.DELETE("/folder", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "DELETE")

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

	r.GET("/tags/", func(c *gin.Context) {
		tags, err := gc.FileStructDB.ListTags()

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return
		}

		c.JSON(http.StatusOK, tags)
	})

	r.GET("/file/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", gin.H{})
	})

	r.GET("/folder", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
		path := c.Query("path")
		path = filepath.Clean(path)
		err := cloudio.downloadFolder(*c, path)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	})

	r.GET("/file/:key", func(c *gin.Context) {
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

	r.GET("/list/", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
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
	r.Run(":3000")
}
