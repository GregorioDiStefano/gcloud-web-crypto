package main

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
)

const (
	errorFileIsDuplicate = "this filename already exists in folder"
)

func (cio *cloudIO) isFileDuplicate(plaintextFolder, plaintextFilename string) bool {
	fullPath := []byte(plaintextFolder + plaintextFilename)
	hmac := cio.cryptoKey.GenerateHMAC(fullPath)
	return gc.FileStructDB.FilenameHMACExists(hmac)
}

func (cio *cloudIO) doUpload(title string, description string, virtualFolder string, tags []string, fh *multipart.FileHeader) error {
	f, err := fh.Open()

	if err != nil {
		return err
	}

	folder := normalizeFolder(filepath.Join("/", virtualFolder, filepath.Dir(fh.Filename)))

	_, filename := filepath.Split(fh.Filename)
	filetype := fh.Header.Get("Content-Type")

	md5Hash := md5.New()
	filesize, err := io.Copy(md5Hash, f)

	if err != nil {
		return err
	}

	f.Seek(0, 0)
	md5HashHex := md5Hash.Sum(nil)

	if cio.isFileDuplicate(folder, filename) {
		fmt.Println("file is duplicate")
		return fmt.Errorf(errorFileIsDuplicate)
	}

	fmt.Println("Uploading file:", filename)

	if encryptedFilename, err := cio.cryptoKey.EncryptText([]byte(filename)); err != nil {
		fmt.Println(err)
		return err
	} else {
		newFile := &gc.File{
			UploadDate:   time.Now(),
			MD5:          fmt.Sprintf("%x", md5HashHex),
			Folder:       folder,
			Filename:     encryptedFilename,
			FilenameHMAC: cio.cryptoKey.GenerateHMAC([]byte(folder + filename)),
			FileSize:     filesize,
			FileType:     filetype,
			Description:  description,
			Tags:         tags}

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
	fmt.Println("cio: ", cio.cryptoKey.Key)
	errorsSeen := false

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

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			for fh := range uploadTasks {
				if err := cio.doUpload(title, description, virtualFolder, tags, &fh); err != nil {
					errorsSeen = true
					fmt.Println("error occured while uploading: ", err.Error())
				}
			}
			wg.Done()
		}()
	}

	for _, fh := range allFiles {
		uploadTasks <- *fh
	}

	close(uploadTasks)
	wg.Wait()

	if errorsSeen {
		return fmt.Errorf("error occured while uploading")
	} else {
		return nil
	}
}

func (cio *cloudIO) uploadFileFromForm(fh *multipart.FileHeader, f multipart.File, key int64) (err error) {
	ctx := context.Background()
	keyAsString := strconv.FormatInt(key, 10)
	w := gc.StorageBucket.Object(keyAsString).NewWriter(ctx)

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
