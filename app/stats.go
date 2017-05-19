package main

import (
	"time"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
)

type fileSystemStats struct {
	UploadedLast7Days  int64 `json:"uploads_last_7_days"`
	UploadedLast14Days int64 `json:"uploads_last_14_days"`
	UploadedLast30Days int64 `json:"uploads_last_30_days"`
	UploadedLast60Days int64 `json:"uploads_last_60_days"`
	UploadedLast90Days int64 `json:"uploads_last_90_days"`
	TotalFileUsage     int64 `json:"total_usage"`
	TotalFiles         int64 `json:"total_files"`
	TotalDownloads     int64 `json:"total_downloads"`

	FilesBetween0MB500MB int64 `json:"files_0mb_500mb"`
	FilesBetween500MB1GB int64 `json:"files_500mb_1gb"`
	FilesBetween1GB2GB   int64 `json:"files_1gb_2gb"`
	FilesBetween2GB3GB   int64 `json:"files_2gb_3gb"`
	FilesBetween3GB4GB   int64 `json:"files_3gb_4gb"`
	FilesBetween4GB5GB   int64 `json:"files_4gb_5gb"`
	FilesOver5GB         int64 `json:"files_over_5gb"`
}

func (user *userData) getUserStats() (*fileSystemStats, error) {
	fileSysStats := new(fileSystemStats)
	files, err := gc.FileStructDB.GetAllFiles(user.userEntry.Username)

	for _, f := range files {

		fileSysStats.TotalFileUsage += f.FileSize

		switch {
		case f.UploadDate.After(time.Now().AddDate(0, 0, -7)):
			fileSysStats.UploadedLast7Days++
		case f.UploadDate.After(time.Now().AddDate(0, 0, -14)):
			fileSysStats.UploadedLast14Days++
		case f.UploadDate.After(time.Now().AddDate(0, 0, -30)):
			fileSysStats.UploadedLast30Days++
		case f.UploadDate.After(time.Now().AddDate(0, 0, -60)):
			fileSysStats.UploadedLast60Days++
		case f.UploadDate.After(time.Now().AddDate(0, 0, -90)):
			fileSysStats.UploadedLast90Days++
		}

		switch {
		case f.FileSize < 500*1024*1024:
			fileSysStats.FilesBetween0MB500MB++
		case f.FileSize <= 1000*1024*1024:
			fileSysStats.FilesBetween500MB1GB++
		case f.FileSize <= 2000*1024*1024:
			fileSysStats.FilesBetween1GB2GB++
		case f.FileSize <= 3000*1024*1024:
			fileSysStats.FilesBetween2GB3GB++
		case f.FileSize <= 4000*1024*1024:
			fileSysStats.FilesBetween3GB4GB++
		case f.FileSize <= 5000*1024*1024:
			fileSysStats.FilesBetween4GB5GB++
		default:
			fileSysStats.FilesOver5GB++

		}

		fileSysStats.TotalDownloads += f.Downloads
		fileSysStats.TotalFiles++
	}

	return fileSysStats, err
}
