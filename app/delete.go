package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

const (
	errorDeleteRootNotPermitted = "unable to delete root"
	errorDeletePathEmpty        = "the requested delete path contains no files"
)

func (cIO *cloudIO) deleteFile(id int64) error {
	err := gc.FileStructDB.DeleteFile(id)

	if err != nil {
		return err
	}

	ctx := context.Background()
	idAsString := strconv.FormatInt(id, 10)

	if err := gc.StorageBucket.Object(idAsString).Delete(ctx); err != nil {
		return fmt.Errorf("unable to delete file: %d", id)
	} else {
		fmt.Println("deleted: ", id)
		return nil
	}
}

func (cIO *cloudIO) deleteFolder(folderPath string) error {
	var deleteFileIDs []int64
	folderPath = filepath.Clean(folderPath)

	if folderPath == "/" {
		return errors.New(errorDeleteRootNotPermitted)
	}
	nestedObjects, err := cIO.listFileSystem(folderPath)

	if len(nestedObjects) == 0 {
		return errors.New(errorDeletePathEmpty)
	} else if err != nil {
		return err
	}

	fmt.Println("Delete folder: ", folderPath)

	for _, fsObject := range nestedObjects {
		if fsObject.Type == "folder" {
			cIO.deleteFolder(fsObject.FullPath)
			gc.FileStructDB.DeleteFolder(fsObject.ID)
		} else {
			fileID := fsObject.ID
			deleteFileIDs = append(deleteFileIDs, fileID)
		}
	}

	var wg sync.WaitGroup

	if err != nil {
		return err
	}

	deleteTasks := make(chan int64, 64)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			for id := range deleteTasks {
				// ignore errors
				cIO.deleteFile(id)
			}
		}()
		wg.Done()
	}

	for _, id := range deleteFileIDs {
		deleteTasks <- id
	}

	close(deleteTasks)
	wg.Wait()

	return nil
}
