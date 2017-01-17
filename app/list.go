package main

import (
	"path/filepath"
	"sync"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

type FileSystemStructure struct {
	ID         int64
	Type       string
	Path       string
	FullPath   string
	ObjectData interface{}
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

	for _, f := range files {
		nestedFiles = append(nestedFiles, f)
	}

	var wg sync.WaitGroup

	for _, f := range folders {
		wg.Add(1)
		go func(f gc.FolderTree) {
			defer wg.Done()
			nestedFiles = append(nestedFiles, listAllNestedFiles(filepath.Join(path, f.Folder))...)
		}(f)
	}

	wg.Wait()
	return nestedFiles
}

func listFileSystem(path string) ([]FileSystemStructure, error) {
	path = normalizeFolder(filepath.Clean(path))
	files, err := gc.FileStructDB.ListFiles(path)

	if err != nil {
		return nil, err
	}

	fs := []FileSystemStructure{}

	for _, file := range files {
		newFSEntry := FileSystemStructure{
			ID:         0,
			Type:       typeFilename,
			Path:       file.Filename,
			FullPath:   filepath.Clean(filepath.Join(file.Folder, file.Filename)),
			ObjectData: file}
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
			FullPath:   normalizeFolder(filepath.Join(path, folder.Folder)),
			ObjectData: folder}
		fs = append(fs, newFSEntry)
	}

	return fs, nil
}
