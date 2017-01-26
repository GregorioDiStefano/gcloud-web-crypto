package main

import (
	"strings"
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

// createDirectoryTree traverses the datastore and creates folders if they don't exist.
func createDirectoryTree(path string, cache map[string]int64) int64 {
	var lastSeenKey int64
	var lastFolder []string

	for _, pathSegment := range gc.PathToFolderTree(path) {
		lastFolder = append(lastFolder, pathSegment.Folder)
		searchFolder := normalizeFolder(strings.Join(lastFolder, "/"))
		var continueToNextPath bool

		// look for this folder in the cache, if it's a hit, we know this folder exists
		for seenPath, seenFolderID := range cache {
			if seenPath == searchFolder {
				lastSeenKey = seenFolderID
				continueToNextPath = true
				break
			}
		}

		// perform the skip
		if continueToNextPath {
			continueToNextPath = false
			continue
		}

		if foundExistingFolder, foundExistingKey, _ := gc.FileStructDB.ListFolders(searchFolder); foundExistingFolder != nil {
			lastSeenKey = foundExistingKey
		} else {
			pathSegment.ParentKey = lastSeenKey
			pathSegment.UploadDate = time.Now()
			newFolderKey, _ := gc.FileStructDB.AddFolder(pathSegment)
			lastSeenKey = newFolderKey
		}

		cache[searchFolder] = lastSeenKey
	}

	return lastSeenKey
}
