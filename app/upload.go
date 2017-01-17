package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
)

func doUpload(title string, description string, virtualFolder string, tags []string, fh *multipart.FileHeader) error {
	f, err := fh.Open()

	if err != nil {
		return err
	}

	folder := normalizeFolder(filepath.Clean(filepath.Join("/", virtualFolder, filepath.Dir(fh.Filename))))
	_, filename := filepath.Split(fh.Filename)

	/*
		if remoteFolders, err := gc.FileStructDB.ListFilesByFolder(folder); err != nil {
			return err
		} else if len(remoteFolders) > 0 {
			for _, v := range remoteFolders {
				if v.Filename == filename && v.Folder == folder {
					return errors.New("file/folder already exists")
				}
			}
		}
	*/

	filetype := fh.Header.Get("Content-Type")

	fmt.Println("Uploading file:", filename)
	if key, md5, filesize, err := uploadFileFromForm(fh, f); err == nil {
		newFile := &gc.File{
			ID:            key,
			PublishedDate: time.Now(),
			MD5:           md5,
			Folder:        folder,
			Filename:      filename,
			FileSize:      filesize,
			FileType:      filetype,
			Title:         title,
			Description:   description,
			Tags:          tags}

		//workaround for bug.
		gc.FileStructDB.NOOP()
		gc.FileStructDB.NOOP()

		_, err := gc.FileStructDB.AddFile(newFile)

		if err != nil {
			return errors.New("error adding file to database: " + err.Error())
		} else {
			spew.Dump(newFile)
		}
	}
	return nil
}

func processFileUpload(c *gin.Context) error {
	var tags []string

	title := c.PostForm("title")
	description := c.PostForm("description")
	virtualFolder := c.PostForm("virtfolder")

	for _, v := range strings.Split(c.PostForm("tags"), ",") {
		tags = append(tags, strings.TrimSpace(v))
	}

	allFiles := c.Request.MultipartForm.File["file"]

	if len(allFiles) == 0 {
		return errors.New("No files found in payload")
	}

	seenPathsMap := make(map[string]int64)
	for _, f := range allFiles {
		folder := normalizeFolder(filepath.Join(virtualFolder, filepath.Dir(f.Filename)))
		for seenFolder := range seenPathsMap {
			if seenFolder == folder {
				continue
			}
		}
		createDirectoryTree(folder, seenPathsMap)
	}

	uploadTasks := make(chan multipart.FileHeader, 64)
	var wg sync.WaitGroup

	for i := 0; i < 15; i++ {
		wg.Add(1)
		go func() {
			for fh := range uploadTasks {
				// ignore errors
				doUpload(title, description, virtualFolder, tags, &fh)
			}
			wg.Done()
		}()
	}

	for _, fh := range allFiles {
		uploadTasks <- *fh
	}

	close(uploadTasks)
	wg.Wait()

	return nil
}

func uploadFileFromForm(fh *multipart.FileHeader, f multipart.File) (key, md5 string, size int64, err error) {

	if gc.StorageBucket == nil {
		return "", "", 0, errors.New("storage bucket is missing - check config.go")
	}

	key = uuid.NewV4().String()

	ctx := context.Background()
	w := gc.StorageBucket.Object(key).NewWriter(ctx)

	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	w.ContentType = fh.Header.Get("Content-Type")
	w.CacheControl = "public, max-age=86400"

	if err = Encrypt(f, w); err != nil {
		return "", "", 0, err
	}

	if err := w.Close(); err != nil {
		return "", "", 0, err
	}

	if r, err := gc.StorageBucket.Object(key).Attrs(ctx); err != nil {
		return "", "", 0, fmt.Errorf("unable to grab file attributes after uploading")
	} else {
		return key, hex.EncodeToString(r.MD5), r.Size, nil
	}
}

func getParentFolderFromFolder(path string) string {
	splitDirs := strings.Split(path, "/")
	parentFolder := strings.Join(splitDirs[0:len(splitDirs)-2], "/")
	if len(parentFolder) == 0 {
		return "/"
	} else {
		return parentFolder + "/"
	}
}

func normalizeFolder(path string) string {
	path = filepath.Clean(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}
	return path
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
