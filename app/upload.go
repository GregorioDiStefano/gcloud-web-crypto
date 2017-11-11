package main

import (
	"context"
	"crypto/sha256"
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
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

const (
	errorFileIsDuplicate = "this filename already exists in folder"
)

type uploadedFile struct {
	fileID      string
	contentType string
	sha2        string
	fileSize    int64
	fileName    string
}

func (user *userData) isFileDuplicate(plaintextFolder, plaintextFilename string) bool {
	fullPath := []byte(plaintextFolder + plaintextFilename)
	hmac := user.cryptoData.GenerateHMAC(fullPath)
	return gc.FileStructDB.FilenameHMACExists(user.userEntry.Username, hmac)
}

func (user *userData) doUpload(fileReader io.Reader, storageClass string) (string, int64, string, error) {
	ctx := context.Background()
	sha256hash := sha256.New()

	filename := uuid.NewV4().String()
	googleStorageWrite := gc.StorageBucket.Object(filename).NewWriter(ctx)
	googleStorageWrite.ACL = []storage.ACLRule{{Entity: storage.AllAuthenticatedUsers, Role: storage.RoleReader}}
	googleStorageWrite.ContentType = "octet/stream"
	googleStorageWrite.CacheControl = "public, max-age=86400"
	googleStorageWrite.StorageClass = storageClass

	r := io.TeeReader(fileReader, sha256hash)
	w := io.MultiWriter(googleStorageWrite)

	//	compressedRead := io.TeeReader(r, zstd.NewWriterLevel(w, zstd.BestCompression))
	written, err := user.cryptoData.EncryptFile(r, w)

	if err != nil {
		return "", 0, "", err
	}

	if err := googleStorageWrite.Close(); err != nil {
		return "", 0, "", err
	}

	return filename, written, fmt.Sprintf("%x", sha256hash.Sum(nil)), nil
}

func (user *userData) createFileEntry(file *uploadedFile, desc, virtualFolder, title string, tags []string) error {
	folder := normalizeFolder(filepath.Join(virtualFolder, filepath.Dir(file.fileName)))
	_, filename := filepath.Split(file.fileName)

	if encryptedFilename, err := user.cryptoData.EncryptText([]byte(filename)); err != nil {
		return err
	} else {
		newFile := &gc.File{
			Username:          user.userEntry.Username,
			UploadDate:        time.Now(),
			SHA2:              file.sha2,
			Folder:            folder,
			Filename:          encryptedFilename,
			FilenameHMAC:      user.cryptoData.GenerateHMAC([]byte(folder + filename)),
			GoogleCloudObject: file.fileID,
			FileSize:          file.fileSize,
			FileType:          file.contentType,
			Description:       desc,
			Tags:              tags}

		if !user.isFileDuplicate(folder, filename) {
			newFileID, err := gc.FileStructDB.AddFile(newFile)
			if err != nil {
				return errors.New("error adding file to database: " + err.Error())
			} else {
				if _, err := user.createDirectoryTree(folder); err == nil {
					fmt.Println(newFileID)
				}
			}
		} else {
			return errors.New(errorFileIsDuplicate)
		}

	}
	return nil
}

func (user *userData) processFileUpload(c *gin.Context) error {
	mr, _ := c.Request.MultipartReader()

	var description, title, virtfolder, storageClass string
	var tags []string
	var uploadedFiles []uploadedFile

	for {
		p, err := mr.NextPart()

		// only after the entire response is posted do we have access to all form parameters
		if err == io.EOF {
			for _, f := range uploadedFiles {
				if err := user.createFileEntry(&f, description, virtfolder, title, tags); err != nil {
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

		case "storage_class":
			//TODO: validate input maybe?
			tmp, _ := ioutil.ReadAll(p)
			storageClass = strings.ToUpper(string(tmp))

		case "tags":
			tmp, _ := ioutil.ReadAll(p)
			unprocessedTags := string(tmp)

			for _, tag := range strings.Split(unprocessedTags, ",") {
				tag := strings.ToLower(strings.TrimSpace(tag))
				if len(tag) > 0 {
					tags = append(tags, tag)
				}
			}
		}

		if p.FormName() == "file" && len(fileName) > 0 && len(contentType) > 0 {
			fileid, filesize, sha2, err := user.doUpload(p, storageClass)
			if err != nil {
				return err
			} else {
				uploadedFiles = append(uploadedFiles,
					uploadedFile{fileID: fileid,
						contentType: contentType,
						fileSize:    filesize,
						sha2:        sha2,
						fileName:    fileName})
			}
		}

		if err != nil {
			log.Println(err)
		}
	}

	fmt.Println("Upload files: ", uploadedFiles)
	return nil
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

// createDirectoryTree traverses the datastore and creates folders if they don't exist.
func (user *userData) createDirectoryTree(path string) (int64, error) {
	var lastSeenKey int64
	var lastFolder []string

	for _, pathSegment := range gc.PathToFolderTree(path) {
		lastFolder = append(lastFolder, pathSegment.Folder)
		searchFolder := normalizeFolder(strings.Join(lastFolder, "/"))

		if foundExistingFolder, foundExistingKey, _ := gc.FileStructDB.ListFolders(user.userEntry.Username, searchFolder); foundExistingFolder != nil {
			lastSeenKey = foundExistingKey
		} else {
			pathSegment.Username = user.userEntry.Username
			pathSegment.ParentKey = lastSeenKey
			pathSegment.UploadDate = time.Now()
			newFolderKey, err := gc.FileStructDB.AddFolder(pathSegment)

			if err != nil {
				return 0, err
			}

			lastSeenKey = newFolderKey
		}

	}

	return lastSeenKey, nil
}
