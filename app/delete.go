package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

const (
	errorDeleteRootNotPermitted = "unable to delete root"
	errorDeletePathEmpty        = "the requested delete path contains no files"
)

func (user *userData) deleteFile(id int64) error {
	f, err := gc.FileStructDB.GetFile(user.userEntry.Username, id)

	if err != nil {
		return err
	}

	err = gc.FileStructDB.DeleteFile(user.userEntry.Username, id)

	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := gc.StorageBucket.Object(f.GoogleCloudObject).Delete(ctx); err != nil {
		return fmt.Errorf("unable to delete file: %d", id)
	} else {
		fmt.Println("deleted: ", id)
		return nil
	}
}

func (user *userData) deleteFolder(folderPath string) error {
	var deleteFileIDs []int64
	folderPath = filepath.Clean(folderPath)

	if folderPath == "/" {
		return errors.New(errorDeleteRootNotPermitted)
	}
	nestedObjects, err := user.listFileSystem(folderPath, nil)

	if err != nil {
		return err
	}

	fmt.Println("Delete folder: ", folderPath)

	for _, fsObject := range nestedObjects {
		if fsObject.Type == "folder" {
			user.deleteFolder(fsObject.FullPath)
			gc.FileStructDB.DeleteFolder(user.userEntry.Username, fsObject.ID)
		} else {
			fileID := fsObject.ID
			deleteFileIDs = append(deleteFileIDs, fileID)
		}
	}

	var wg sync.WaitGroup

	if err != nil {
		return err
	}

	var deleteError error
	deleteTasks := make(chan int64, 64)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			for id := range deleteTasks {
				// ignore errors
				if deleteError = user.deleteFile(id); deleteError != nil {
					return
				}
			}
		}()
		wg.Done()
	}

	for _, id := range deleteFileIDs {
		deleteTasks <- id
	}

	close(deleteTasks)
	wg.Wait()

	fmt.Println("deleteError: ", deleteError)
	return deleteError
}
