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
	Path       string    `json:"folder"`
	FullPath   string    `json:"fullpath"`
	UploadDate time.Time `json:"upload_date"`

	/* Only displayed for files */
	Filename    string   `json:"filename,omitempty"`
	Folder      string   `json:"folder2,omitempty"`
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

func listAllNestedFiles(path string) []gc.File {
	var nestedFiles []gc.File
	path = normalizeFolder(filepath.Clean(path))

	folders, _, _ := gc.FileStructDB.ListFolders(path)
	files, _ := gc.FileStructDB.ListFiles(path)

	for _, file := range files {
		nestedFiles = append(nestedFiles, file)
	}

	var wg sync.WaitGroup

	for _, folder := range folders {
		wg.Add(1)
		go func(f gc.FolderTree) {
			defer wg.Done()
			nestedFiles = append(nestedFiles, listAllNestedFiles(filepath.Join(path, f.Folder))...)
		}(folder)
	}

	wg.Wait()
	return nestedFiles
}

/*
func listFilesWithTags(tags []string) ([]FileSystemStructure, error) {
	fmt.Println("tags:", tags)
	if files, err := gc.FileStructDB.ListFilesWithTags(tags); err != nil {
		return nil, err
	} else {
		fs := []FileSystemStructure{}
		fmt.Println(files)
		for _, file := range files {
			newFSEntry := FileSystemStructure{
				ID:         0,
				Type:       typeFilename,
				Path:       file.Filename,
				FullPath:   filepath.Clean(filepath.Join(file.Folder, file.Filename)),
				ObjectData: file}
			fs = append(fs, newFSEntry)
		}

		return fs, nil
	}
}
*/

func (cio *cloudIO) listFileSystem(path string) ([]FileSystemStructure, error) {
	path = normalizeFolder(filepath.Clean(path))
	files, err := gc.FileStructDB.ListFiles(path)

	if err != nil {
		return nil, err
	}

	fs := []FileSystemStructure{}

	for _, file := range files {

		plainTextFilename, err := cio.cryptoKey.DecryptText(file.Filename)

		if err != nil {
			return nil, err
		}

		newFSEntry := FileSystemStructure{
			ID:          file.ID,
			Type:        typeFilename,
			Path:        string(plainTextFilename),
			FullPath:    filepath.Clean(filepath.Join(file.Folder, string(plainTextFilename))),
			Filename:    string(plainTextFilename),
			Folder:      file.Folder,
			FileType:    file.FileType,
			FileSize:    file.FileSize,
			Description: file.Description,
			Tags:        file.Tags,
			UploadDate:  file.UploadDate,
			MD5:         file.MD5,
		}
		fs = append(fs, newFSEntry)
	}

	folders, _, err := gc.FileStructDB.ListFolders(path)

	if err != nil {
		return nil, err
	}

	for _, folder := range folders {
		newFSEntry := FileSystemStructure{
			ID:         folder.ID,
			Type:       typeFolder,
			Path:       normalizeFolder(folder.Folder),
			UploadDate: folder.UploadDate,
			FullPath:   normalizeFolder(filepath.Join(path, folder.Folder))}
		fs = append(fs, newFSEntry)
	}

	return fs, nil
}
