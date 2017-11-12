package gscrypto

import (
	"strings"
	"time"
)

type File struct {
	ID                int64 `datastore:"-"`
	Username          string
	Filename          []byte
	FilenameHMAC      string
	GoogleCloudObject string
	Folder            string
	FileType          string
	FileSize          int64
	UploadDate        time.Time
	Downloads         int64
	Description       string
	Tags              []string
	Compressed        bool
	SHA2              string
}

type FolderTree struct {
	ID           int64 `datastore:"-"`
	Username     string
	UploadDate   time.Time
	ParentKey    int64
	ParentFolder string
	Folder       string
}

type FileDatabase interface {
	ListFiles(user string, path string) ([]File, error)
	ListTags() ([]string, error)
	ListFilesWithTags([]string) ([]File, error)

	ListFolders(user, path string) ([]FolderTree, int64, error)
	ListAllFolders(user, path string, limit int) ([]string, error)
	// FolderTree contains a username
	AddFolder(f *FolderTree) (int64, error)

	// File contains a username
	AddFile(f *File) (id int64, err error)
	UpdateFile(f *File, id int64) (err error)
	FilenameHMACExists(user string, hmac string) bool
	GetFile(user string, id int64) (*File, error)
	GetAllFiles(user string) ([]*File, error)
	DeleteFile(user string, id int64) error
	DeleteFolder(user string, id int64) error
	Close()
}

func PathToFolderTree(path string) []*FolderTree {
	var parent string
	ft := make([]*FolderTree, 0)

	splitPath := strings.Split(path, "/")
	for depth, folder := range splitPath {
		if depth == 0 {
			continue
		} else if depth == len(splitPath)-1 {
			break
		} else {
			parent = splitPath[depth-1]
		}
		ft = append(ft, &FolderTree{ParentFolder: parent, Folder: folder})
	}
	return ft
}
