package main

import (
	"context"
	"fmt"

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

/*
func deleteFolder(folderpath string) error {
	folderpath = filepath.Clean(folderpath)
	nestedFiles, err := gc.FileStructDB.ListNestedByFolderPath(folderpath)
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

	for _, v := range nestedFiles {
		deleteTasks <- v.ID
	}
	close(deleteTasks)
	wg.Wait()
	return nil
}
*/
