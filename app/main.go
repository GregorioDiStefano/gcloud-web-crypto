package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
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

	if AUTH_ENABLED {
		private.Use(jwt.Auth(gc.SecretKey))
	}

	go func() {
		for {
			fmt.Println("running noop")
			time.Sleep(30 * time.Second)
			gc.FileStructDB.NOOP()
		}
	}()

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
		err := processFileUpload(c)

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
		uuid := c.Param("uuid")
		err := deleteFile(uuid)

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
		//folderDeletePath := c.Query("path")
		//err := deleteFolder(folderDeletePath)

		//if err != nil {
		//	fmt.Println(err)
		//}
	})

	r.GET("/file/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "upload.html", gin.H{})
	})

	r.GET("/folder", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
		path := c.Query("path")
		path = filepath.Clean(path)
		err := downloadFolder(*c, path)

		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
	})

	r.GET("/file/:key", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET")
		k := c.Param("key")
		ef, err := gc.FileStructDB.GetFile(k)
		if err != nil {
			c.Abort()
		}

		ctx := context.Background()
		r, err := gc.StorageBucket.Object(ef.ID).NewReader(ctx)

		if err != nil {
			panic(err)
		}

		c.Writer.Header().Set("content-disposition", "attachment; filename=\""+ef.Filename+"\"")

		Decrypt(r, c.Writer)

	})

	r.GET("/list/", func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		path := c.Query("path")

		if len(path) == 0 {
			c.JSON(http.StatusBadRequest, "path is missing")
		} else if fs, err := listFileSystem(path, c); err != nil {
			c.JSON(http.StatusInternalServerError, err)
		} else {
			c.IndentedJSON(http.StatusOK, fs)
		}
	})
	r.Run(":3000")
}
