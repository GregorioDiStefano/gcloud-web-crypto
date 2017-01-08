package main

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/gin-gonic/gin"
)

const (
	errorUnableToLoadNestedFolders = "unable to read from nested folder"
)

func downloadFolder(httpContext gin.Context, path string) error {

	zipfile := strings.Split(path, "/")
	zipfileStr := zipfile[len(zipfile)-1] + ".zip"

	//files, err := gc.FileStructDB.ListNestedByFolderPath(path)
	files, err := gc.FileStructDB.ListFiles(path)
	if err != nil {
		return errors.New(errorUnableToLoadNestedFolders)
	}

	httpContext.Writer.Header().Set("Content-Type", "application/zip")
	httpContext.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipfileStr))
	zw := zip.NewWriter(httpContext.Writer)
	defer zw.Close()

	for _, v := range files {
		ctx := context.Background()
		r, err := gc.StorageBucket.Object(v.ID).NewReader(ctx)

		if err != nil {
			return err
		}

		header := &zip.FileHeader{
			Name: "fix",
			//Name:         filepath.Join(v.Folder, v.Filename),
			Method:       zip.Deflate,
			ModifiedTime: uint16(time.Now().UnixNano()),
			ModifiedDate: uint16(time.Now().UnixNano()),
		}

		fw, err := zw.CreateHeader(header)

		if err != nil {
			return err
		}

		Decrypt(r, fw)

	}

	return nil
}
