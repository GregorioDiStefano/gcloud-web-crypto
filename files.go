package gscrypto

import (
	"strings"
	"time"
)

type File struct {
	ID          int64 `datastore:"-"`
	Filename    []byte
	Folder      string
	FileType    string
	FileSize    int64
	UploadDate  time.Time
	Description string
	Tags        []string
	MD5         string
}

type FolderTree struct {
	ID           int64 `datastore:"-"`
	UploadDate   time.Time
	ParentKey    int64
	ParentFolder string
	Folder       string
}

type FileDatabase interface {
	ListFiles(string) ([]File, error)
	ListTags() ([]string, error)
	ListFilesWithTags([]string) ([]File, error)
	ListFolders(string) ([]FolderTree, int64, error)
	AddFolder(f *FolderTree) (int64, error)
	AddFile(f *File) (id int64, err error)
	GetFile(id int64) (*File, error)
	DeleteFile(id int64) error
	DeleteFolder(id int64) error
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
