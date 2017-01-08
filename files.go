package gscrypto

import (
	"strings"
	"time"
)

type File struct {
	ID            string
	Filename      string
	Folder        string
	FileType      string
	FileSize      int64
	PublishedDate time.Time
	Description   string
	Tags          []string
	Title         string
	MD5           string
}

type FolderTree struct {
	PublishedDate time.Time
	ParentKey     int64
	ParentFolder  string
	Folder        string
}

type FileDatabase interface {
	ListFiles(string) ([]*File, error)
	ListFolders(string) ([]*FolderTree, int64, error)

	ListFilesByFolder(string) ([]*File, error)
	ListFilesByFolderID(string) ([]*File, error)
	ListFilesByFolderPath(string) ([]*File, error)

	DoesFolderExist(parent, folder string) (bool, int64)

	AddFolder(f *FolderTree) (int64, error)
	AddFile(f *File) (id int64, err error)
	GetFile(id string) (*File, error)
	DeleteFile(uuid string) error

	NOOP()
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
