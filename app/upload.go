package main

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	//"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

const (
	errorFileIsDuplicate = "this filename already exists in folder"
)

type uploadedFile struct {
	fileID      string
	contentType string
	md5Hash     string
	fileSize    int64
	fileName    string
}

func (cio *cloudIO) isFileDuplicate(plaintextFolder, plaintextFilename string) bool {
	fullPath := []byte(plaintextFolder + plaintextFilename)
	hmac := cio.cryptoKey.GenerateHMAC(fullPath)
	return gc.FileStructDB.FilenameHMACExists(hmac)
}

func (cio *cloudIO) doUpload(fileReader io.Reader) (string, int64, string, error) {
	ctx := context.Background()
	md5Hash := md5.New()

	filename := uuid.NewV4().String()
	googleStorageWrite := gc.StorageBucket.Object(filename).NewWriter(ctx)
	googleStorageWrite.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	googleStorageWrite.ContentType = "octet/stream"
	googleStorageWrite.CacheControl = "public, max-age=86400"

	r := io.TeeReader(fileReader, md5Hash)
	w := io.MultiWriter(googleStorageWrite)

	written, err := cio.cryptoKey.EncryptFile(r, w)
	if err != nil {
		return "", 0, "", err
	}

	if err := googleStorageWrite.Close(); err != nil {
		return "", 0, "", err
	}

	return filename, written, fmt.Sprintf("%x", md5Hash.Sum(nil)), nil
}

func (f *uploadedFile) createFileEntry(cio *cloudIO, desc, virtualFolder, title string, tags []string) error {
	folder := normalizeFolder(filepath.Join(virtualFolder, filepath.Dir(f.fileName)))
	_, filename := filepath.Split(f.fileName)

	if encryptedFilename, err := cio.cryptoKey.EncryptText([]byte(filename)); err != nil {
		fmt.Println(err)
		return err
	} else {
		newFile := &gc.File{
			UploadDate:        time.Now(),
			MD5:               f.md5Hash,
			Folder:            folder,
			Filename:          encryptedFilename,
			FilenameHMAC:      cio.cryptoKey.GenerateHMAC([]byte(folder + filename)),
			GoogleCloudObject: f.fileID,
			FileSize:          f.fileSize,
			FileType:          f.contentType,
			Description:       desc,
			Tags:              tags}

		fmt.Println(folder, filename, cio.isFileDuplicate(folder, filename))

		if !cio.isFileDuplicate(folder, filename) {
			newFileID, err := gc.FileStructDB.AddFile(newFile)
			if err != nil {
				return errors.New("error adding file to database: " + err.Error())
			} else {
				if _, err := createDirectoryTree(folder); err == nil {
					fmt.Println(newFileID)
				}
			}
		} else {
			return errors.New(errorFileIsDuplicate)
		}

	}
	return nil
}

func (cio *cloudIO) processFileUpload(c *gin.Context) error {
	mr, _ := c.Request.MultipartReader()

	var description, title, virtfolder string
	var tags []string
	var uploadedFiles []uploadedFile

	for {
		p, err := mr.NextPart()

		// only after the entire response is posted do we have access to all form parameters
		if err == io.EOF {
			for _, f := range uploadedFiles {
				if err := f.createFileEntry(cio, description, virtfolder, title, tags); err != nil {
					return err
				}
			}
			break
		}

		if err != nil {
			break
		}

		fileName := p.FileName()
		contentType := p.Header.Get("Content-Type")

		switch p.FormName() {
		case "description":
			tmp, _ := ioutil.ReadAll(p)
			description = string(tmp)
		case "virtfolder":
			tmp, _ := ioutil.ReadAll(p)
			virtfolder = string(tmp)
		case "tags":
			tmp, _ := ioutil.ReadAll(p)
			unprocessedTags := string(tmp)

			for _, tag := range strings.Split(unprocessedTags, ",") {
				trimmedTag := strings.TrimSpace(tag)
				if len(trimmedTag) > 0 {
					tags = append(tags, strings.TrimSpace(tag))
				}
			}
		}

		if p.FormName() == "file" && len(fileName) > 0 && len(contentType) > 0 {
			fileid, filesize, md5, err := cio.doUpload(p)
			if err != nil {
				return err
			} else {
				uploadedFiles = append(uploadedFiles, uploadedFile{fileID: fileid, contentType: contentType, fileSize: filesize, md5Hash: md5, fileName: fileName})
			}
		}

		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("Upload files: ", uploadedFiles)
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
