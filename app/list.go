package main

import (
	"fmt"
	"path/filepath"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/gin-gonic/gin"
)

type FileStructure struct {
	Type     string
	Path     string
	FullPath string
	FileData *gc.File
}

func pathAlreadyFileStructure(path string, list []FileStructure) bool {
	for _, b := range list {
		if b.Path == path {
			return true
		}
	}
	return false
}

func listFileSystem(path string, c *gin.Context) (*[]FileStructure, error) {
	files, err := gc.FileStructDB.ListFiles(path)
	fmt.Println(files, err)
	fs := []FileStructure{}

	for _, f := range files {
		newFSEntry := FileStructure{
			Type:     "filename",
			Path:     f.Filename,
			FullPath: filepath.Clean(filepath.Join(f.Folder, f.Filename)),
			FileData: f}
		fs = append(fs, newFSEntry)
	}

	folders, _, err := gc.FileStructDB.ListFolders(path)
	for _, f := range folders {
		newFSEntry := FileStructure{
			Type:     "folder",
			Path:     f.Folder,
			FullPath: filepath.Clean(filepath.Join(f.Folder)),
			FileData: nil}
		fs = append(fs, newFSEntry)
	}

	fmt.Println("folders: ", folders)
	fmt.Println(err)

	return &fs, nil
	/*
		fmt.Println("A")
		files, err := gc.FileStructDB.ListFiles("")
		fmt.Println("B")
		fmt.Println(len(files))

		if err != nil {
			c.JSON(http.StatusInternalServerError, err.Error())
			return nil, err
		}

		fs := []FileStructure{}
		for _, v := range files {
			remoteObjectFolder := filepath.Clean(v.Folder)
			listPath := filepath.Clean(path)

			if !strings.HasSuffix(listPath, "/") {
				listPath += "/"
			}

			if !strings.HasSuffix(remoteObjectFolder, "/") {
				remoteObjectFolder += "/"
			}

			if listPath == remoteObjectFolder {
				newFSEntry := FileStructure{Type: "filename",
					Path:     v.Filename,
					FullPath: filepath.Clean(filepath.Join(v.Folder, v.Filename)),
					FileData: v}
				ok := pathAlreadyFileStructure(v.Filename, fs)
				if !ok {
					fs = append(fs, newFSEntry)
				}
			} else if strings.HasPrefix(remoteObjectFolder, listPath) {
				inclosedFolder := strings.TrimPrefix(remoteObjectFolder, listPath)
				nestedFolder := strings.Split(inclosedFolder, "/")[0]
				newFSEntry := FileStructure{Type: "folder",
					Path:     nestedFolder,
					FullPath: filepath.Clean(filepath.Join(path, nestedFolder)),
					FileData: v}
				ok := pathAlreadyFileStructure(nestedFolder, fs)
				if len(nestedFolder) > 0 && !ok {
					fs = append(fs, newFSEntry)
				}
			}
		}
	*/
	//return nil, nil
}
