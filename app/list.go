package main

import (
	"path/filepath"
	"sync"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

type FileSystemStructure struct {
	ID         int64     `json:"id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	FullPath   string    `json:"fullpath"`
	UploadDate time.Time `json:"upload_date"`

	/* Only displayed for files */
	FileType    string   `json:"filetype,omitempty"`
	FileSize    int64    `json:"filesize,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	MD5         string   `json:"md5,omitempty"`
}

const (
	typeFilename = "filename"
	typeFolder   = "folder"
)

func (user *userData) listAllNestedFiles(path string) []gc.File {
	var nestedFiles []gc.File
	path = normalizeFolder(filepath.Clean(path))
	username := user.userEntry.Username
	folders, _, _ := gc.FileStructDB.ListFolders(username, path)
	files, _ := gc.FileStructDB.ListFiles(username, path)

	for _, file := range files {
		nestedFiles = append(nestedFiles, file)
	}

	var wg sync.WaitGroup

	for _, folder := range folders {
		wg.Add(1)
		go func(f gc.FolderTree) {
			defer wg.Done()
			nestedFiles = append(nestedFiles, user.listAllNestedFiles(filepath.Join(path, f.Folder))...)
		}(folder)
	}

	wg.Wait()
	return nestedFiles
}

func (user *userData) listFileSystem(path string) ([]FileSystemStructure, error) {
	path = normalizeFolder(filepath.Clean(path))
	files, err := gc.FileStructDB.ListFiles(user.userEntry.Username, path)

	if err != nil {
		return nil, err
	}

	fs := []FileSystemStructure{}

	for _, file := range files {

		plainTextFilename, err := user.cryptoData.DecryptText(file.Filename)

		if err != nil {
			return nil, err
		}

		newFSEntry := FileSystemStructure{
			ID:          file.ID,
			Type:        typeFilename,
			Name:        string(plainTextFilename),
			FullPath:    filepath.Clean(filepath.Join(file.Folder, string(plainTextFilename))),
			FileType:    file.FileType,
			FileSize:    file.FileSize,
			Description: file.Description,
			Tags:        file.Tags,
			UploadDate:  file.UploadDate,
			MD5:         file.MD5,
		}
		fs = append(fs, newFSEntry)
	}

	folders, _, err := gc.FileStructDB.ListFolders(user.userEntry.Username, path)

	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		newFSEntry := FileSystemStructure{
			ID:         folder.ID,
			Type:       typeFolder,
			Name:       normalizeFolder(folder.Folder),
			UploadDate: folder.UploadDate,
			FullPath:   normalizeFolder(filepath.Join(path, folder.Folder))}
		fs = append(fs, newFSEntry)
	}

	return fs, nil
}
