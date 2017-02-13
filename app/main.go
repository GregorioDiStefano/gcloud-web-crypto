package main

import (
	"bytes"
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
	"github.com/gin-gonic/gin"
)

const (
	AUTH_ENABLED = true
	DEV_MODE     = true
)

func main() {
	r := gin.Default()

	private := r.Group("/auth")
	cryptoKey := crypto.CryptoKey{Key: gc.Password}
	cloudio := cloudIO{cryptoKey: cryptoKey, storageBucket: gc.StorageBucket}

	if AUTH_ENABLED {
		private.Use(jwt.Auth(gc.SecretKey))
	}

	r.POST("/account/login", func(c *gin.Context) {
		if passwordFromForm, ok := c.GetPostForm("password"); !ok || !bytes.Equal([]byte(passwordFromForm), gc.PlainTextPassword) {
			c.Status(http.StatusForbidden)
			return
		}

		token := jwt_lib.New(jwt_lib.GetSigningMethod("HS256"))

		token.Claims = jwt_lib.MapClaims{
			"id":  "admin",
			"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
		}

		tokenString, err := token.SignedString([]byte(gc.SecretKey))
		if err != nil {
			c.JSON(500, gin.H{"message": "Could not generate token"})
		}
		c.JSON(200, gin.H{"token": tokenString})
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

	private.GET("/file/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", gin.H{})
	})

	private.GET("/folder", func(c *gin.Context) {
		path := c.Query("path")
		path = filepath.Clean(path)
		err := cloudio.downloadFolder(*c, path)

		if err != nil {
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
	r.Run(":3000")
}
