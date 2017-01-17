package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

func deleteFile(uuid string) error {
	fmt.Println("Deleting: ", uuid)
	err := gc.FileStructDB.DeleteFile(uuid)

	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := gc.StorageBucket.Object(uuid).Delete(ctx); err != nil {
		return fmt.Errorf("unable to delete file: %s", uuid)
	} else {
		return nil
	}
}

func deleteFolder(folderPath string) error {
	fmt.Println("Delete folder: ", folderPath)
	var deleteFileIDs []string
	folderPath = filepath.Clean(folderPath)
	nestedObjects, err := listFileSystem(folderPath)

	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, fsObject := range nestedObjects {
		if fsObject.Type == "folder" {
			deleteFolder(fsObject.FullPath)
			fmt.Println("Deleting: ", gc.FileStructDB.DeleteFolder(fsObject.ObjectData.(*gc.FolderTree).ID))
		} else {
			fileID := fsObject.ObjectData.(*gc.File).ID
			deleteFileIDs = append(deleteFileIDs, fileID)
		}
	}

	var wg sync.WaitGroup

	if err != nil {
		return err
	}

	deleteTasks := make(chan string, 64)
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			for uuid := range deleteTasks {
				// ignore errors
				deleteFile(uuid)
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
