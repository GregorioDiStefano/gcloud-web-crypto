package main

import (
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

// createDirectoryTree traverses the datastore and creates folders if they don't exist.
func (user *userData) createDirectoryTree(path string) (int64, error) {
	var lastSeenKey int64
	var lastFolder []string

	for _, pathSegment := range gc.PathToFolderTree(path) {
		lastFolder = append(lastFolder, pathSegment.Folder)
		searchFolder := normalizeFolder(strings.Join(lastFolder, "/"))

		if foundExistingFolder, foundExistingKey, _ := gc.FileStructDB.ListFolders(user.userEntry.Username, searchFolder); foundExistingFolder != nil {
			lastSeenKey = foundExistingKey
		} else {
			pathSegment.Username = user.userEntry.Username
			pathSegment.ParentKey = lastSeenKey
			pathSegment.UploadDate = time.Now()
			newFolderKey, err := gc.FileStructDB.AddFolder(pathSegment)

			if err != nil {
				return 0, err
			}

			lastSeenKey = newFolderKey
		}

	}

	return lastSeenKey, nil
}
