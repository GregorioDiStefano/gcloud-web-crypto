package main

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
)

func (cio *cloudIO) doUpload(title string, description string, virtualFolder string, tags []string, fh *multipart.FileHeader) error {
	f, err := fh.Open()

	if err != nil {
		return err
	}

	folder := normalizeFolder(filepath.Join("/", virtualFolder, filepath.Dir(fh.Filename)))

	fmt.Println("folder: ", folder)
	_, filename := filepath.Split(fh.Filename)
	filetype := fh.Header.Get("Content-Type")

	md5Hash := md5.New()
	filesize, err := io.Copy(md5Hash, f)

	if err != nil {
		return err
	}

	f.Seek(0, 0)

	md5HashHex := md5Hash.Sum(nil)
	//TODO: check for file duplicates

	fmt.Println("Uploading file:", filename)

	if encryptedFile, err := cio.cryptoKey.EncryptText([]byte(filename)); err != nil {
		fmt.Println(err)
		return err
	} else {
		newFile := &gc.File{
			UploadDate:  time.Now(),
			MD5:         fmt.Sprintf("%x", md5HashHex),
			Folder:      folder,
			Filename:    encryptedFile,
			FileSize:    filesize,
			FileType:    filetype,
			Description: description,
			Tags:        tags}

		newFileID, err := gc.FileStructDB.AddFile(newFile)

		if err != nil {
			return errors.New("error adding file to database: " + err.Error())
		} else {
			if err := cio.uploadFileFromForm(fh, f, newFileID); err == nil {
				spew.Dump(newFile)
			}
		}
	}

	return nil
}

func (cio *cloudIO) processFileUpload(c *gin.Context) error {
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

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			for fh := range uploadTasks {
				// ignore errors
				cio.doUpload(title, description, virtualFolder, tags, &fh)
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

func (cio *cloudIO) uploadFileFromForm(fh *multipart.FileHeader, f multipart.File, key int64) (err error) {
	ctx := context.Background()
	w := gc.StorageBucket.Object(string(key)).NewWriter(ctx)

	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	w.ContentType = fh.Header.Get("Content-Type")
	w.CacheControl = "public, max-age=86400"

	if err = cio.cryptoKey.EncryptFile(f, w); err != nil {
		return
	}

	if err := w.Close(); err != nil {
		return err
	}

	return nil
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

	if path == "." {
		return "/"
	}

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
