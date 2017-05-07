package main

import (
	"archive/zip"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	gc "github.com/GregorioDiStefano/gcloud-web-crypto"

	"github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	"github.com/gin-gonic/gin"
)

const (
	errorUnableToLoadNestedFolders = "unable to read from nested folder"
)

type cloudIO struct {
	cryptoKey     crypto.CryptoKey
	storageBucket *storage.BucketHandle
}

func (cIO *cloudIO) downloadFile(httpContext *gin.Context, id int64) error {
	ef, err := gc.FileStructDB.GetFile(id)

	if err != nil {
		return err
	}

	ctx := context.Background()
	r, err := cIO.storageBucket.Object(ef.GoogleCloudObject).NewReader(ctx)

	if err != nil {
		return err
	}

	if plainTextFilename, err := cIO.cryptoKey.DecryptText(ef.Filename); err != nil {
		return err
	} else {
		httpContext.Writer.Header().Set("content-disposition", "attachment; filename=\""+string(plainTextFilename)+"\"")
	}

	if err := cIO.cryptoKey.DecryptFile(r, httpContext.Writer); err != nil {
		return err
	}

	httpContext.Writer.Flush()
	return nil
}

func (cIO *cloudIO) downloadFolder(httpContext gin.Context, path string) error {
	zipfile := strings.Split(path, "/")
	zipfileStr := zipfile[len(zipfile)-1] + ".zip"

	files := listAllNestedFiles(path)

	httpContext.Writer.Header().Set("Content-Type", "application/zip")
	httpContext.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipfileStr))
	zw := zip.NewWriter(httpContext.Writer)
	defer zw.Close()

	for _, file := range files {
		ctx := context.Background()
		r, err := cIO.storageBucket.Object(file.GoogleCloudObject).NewReader(ctx)

		if r == nil {
			fmt.Println(file.ID, " is nil")
			continue
		} else {
			fmt.Println("filesize: ", r.Size())
		}

		if err != nil {
			return err
		}

		plainTextFilename, err := cIO.cryptoKey.DecryptText(file.Filename)

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

		if err := cIO.cryptoKey.DecryptFile(r, fw); err != nil {
			fmt.Println(err)
			return err
		}
	}

	return nil
}
