package main

import (
	"archive/zip"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/gin-gonic/gin"
)

const (
	errorUnableToLoadNestedFolders = "unable to read from nested folder"
)

func downloadFile(httpContext *gin.Context, key string) error {
	httpContext.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	httpContext.Writer.Header().Set("Access-Control-Allow-Methods", "GET")

	ef, err := gc.FileStructDB.GetFile(key)

	if err != nil {
		return err
	}

	ctx := context.Background()
	r, err := gc.StorageBucket.Object(ef.ID).NewReader(ctx)

	if err != nil {
		return err
	}

	httpContext.Writer.Header().Set("content-disposition", "attachment; filename=\""+ef.Filename+"\"")

	if err := Decrypt(r, httpContext.Writer); err != nil {
		return err
	}

	httpContext.Writer.Flush()
	return nil
}

func downloadFolder(httpContext gin.Context, path string) error {
	zipfile := strings.Split(path, "/")
	zipfileStr := zipfile[len(zipfile)-1] + ".zip"

	files := listAllNestedFiles(path)

	httpContext.Writer.Header().Set("Content-Type", "application/zip")
	httpContext.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipfileStr))
	zw := zip.NewWriter(httpContext.Writer)
	defer zw.Close()

	for _, file := range files {
		ctx := context.Background()
		r, err := gc.StorageBucket.Object(string(file.ID)).NewReader(ctx)

		if r == nil {
			fmt.Println(file.ID, " is nil")
			continue
		} else {
			fmt.Println("filesize: ", r.Size())
		}

		if err != nil {
			return err
		}

		header := &zip.FileHeader{
			Name:         filepath.Join(file.Folder, file.Filename),
			Method:       zip.Deflate,
			ModifiedTime: uint16(time.Now().UnixNano()),
			ModifiedDate: uint16(time.Now().UnixNano()),
		}

		fmt.Println("File: ", header.Name, " added to zip archive")
		fw, err := zw.CreateHeader(header)

		if err != nil {
			return err
		}

		if err := Decrypt(r, fw); err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}
