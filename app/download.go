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

func (user *userData) downloadFile(httpContext *gin.Context, id int64) error {
	ef, err := gc.FileStructDB.GetFile(user.userEntry.Username, id)

	if err != nil {
		return err
	}

	ef.Downloads++
	go gc.FileStructDB.UpdateFile(ef, id)

	ctx := context.Background()
	r, err := config.storageBucket.Object(ef.GoogleCloudObject).NewReader(ctx)

	if err != nil {
		return err
	}

	if plainTextFilename, err := user.cryptoData.DecryptText(ef.Filename); err != nil {
		return err
	} else {
		httpContext.Writer.Header().Set("content-disposition", "attachment; filename=\""+string(plainTextFilename)+"\"")
	}

	f, err := gc.FileStructDB.GetFile(user.userEntry.Username, id)

	if err != nil {
		return err
	}

	if err := user.cryptoData.DecryptFile(r, httpContext.Writer, f.Compressed); err != nil {
		return err
	}

	httpContext.Writer.Flush()
	return nil
}

func (user *userData) downloadFolder(httpContext gin.Context, path string) error {
	zipfile := strings.Split(path, "/")
	zipfileStr := zipfile[len(zipfile)-1] + ".zip"

	files := user.listAllNestedFiles(path)

	if len(files) > 0 {
		httpContext.Writer.Header().Set("Content-Type", "application/zip")
		httpContext.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipfileStr))
		zw := zip.NewWriter(httpContext.Writer)
		defer zw.Close()

		for _, file := range files {
			ctx := context.Background()
			r, err := config.storageBucket.Object(file.GoogleCloudObject).NewReader(ctx)

			if r == nil {
				fmt.Println(file.ID, " is nil")
				continue
			} else {
				fmt.Println("filesize: ", r.Size())
			}

			if err != nil {
				return err
			}

			plainTextFilename, err := user.cryptoData.DecryptText(file.Filename)

			if err != nil {
				return err
			}

			header := &zip.FileHeader{
				Name:         filepath.Join(file.Folder, string(plainTextFilename)),
				Method:       zip.Deflate,
				ModifiedTime: uint16(time.Now().UnixNano()),
				ModifiedDate: uint16(time.Now().UnixNano()),
			}

			fmt.Println("File: ", header.Name, " added to zip archive")
			fw, err := zw.CreateHeader(header)

			if err != nil {
				return err
			}

			if err := user.cryptoData.DecryptFile(r, fw, file.Compressed); err != nil {
				fmt.Println(err)
				return err
			}
		}
		return nil
	} else {
		return fmt.Errorf("no files found for specified path")
	}
}
