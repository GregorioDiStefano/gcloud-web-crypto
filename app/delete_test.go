package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

type simpleFile struct {
	filename string
	path     string
}

func TestDeleteFolder(t *testing.T) {
	type deleteFolder struct {
		path string
		//		fileExists         bool
		expectedHTTPStatus int
	}

	clearBucket()
	clearDatastore()

	createAdmin()
	adminCookie := loginUser(adminLoginDetails)

	testCases := []struct {
		uploads        []simpleFile
		delete         []deleteFolder
		resultingFiles []simpleFile
	}{
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFolder{
				{path: "/x"},
				{path: "/z/z"},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
		},
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
				simpleFile{filename: "b", path: "/b/c/"},
			},
			[]deleteFolder{
				{path: "/b"},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
			},
		},
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFolder{
				{path: "/a"},
				{path: "/b"},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
			},
		},
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFolder{
				{path: "/", expectedHTTPStatus: http.StatusForbidden},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
		},
	}

	for _, test := range testCases {
		clearDatastore()
		for _, filesToUpload := range test.uploads {

			f := grequests.FileUpload{
				FieldName:    "",
				FileName:     filesToUpload.filename,
				FileContents: ioutil.NopCloser(strings.NewReader("foo")),
			}

			resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
				Files: []grequests.FileUpload{f}, Data: map[string]string{"filename": filesToUpload.filename, "virtfolder": filesToUpload.path}})

			assert.Nil(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend")
		}

		for _, filesToDelete := range test.delete {
			resp, _ := grequests.Delete(ts.URL+"/auth/folder/?path="+filesToDelete.path, &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
			if filesToDelete.expectedHTTPStatus != 0 {
				assert.Equal(t, filesToDelete.expectedHTTPStatus, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusNoContent, resp.StatusCode)
			}
		}

		assert.EqualValues(t, test.resultingFiles, fsLayoutToSimpleFile(getAllFSObjectsUsingAPI("/", adminCookie)))
	}
}

func TestDeleteFiles(t *testing.T) {

	type deleteFile struct {
		simpleFile         simpleFile
		fileExists         bool
		expectedHTTPStatus int
	}

	clearDatastore()
	createAdmin()

	adminCookie := loginUser(adminLoginDetails)

	testCases := []struct {
		uploads        []simpleFile
		delete         []deleteFile
		resultingFiles []simpleFile
	}{
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFile{
				{simpleFile{filename: "a_a", path: "/a"}, true, http.StatusNoContent},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "b", path: "/b"},
			},
		},
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFile{
				{simpleFile{filename: "x", path: "/"}, false, http.StatusNoContent},
				{simpleFile{filename: "b", path: "/b"}, true, http.StatusInternalServerError},
			},
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
			},
		},
		{
			[]simpleFile{
				simpleFile{filename: "a", path: "/"},
				simpleFile{filename: "a_a", path: "/a"},
				simpleFile{filename: "b", path: "/b"},
			},
			[]deleteFile{
				{simpleFile{filename: "a", path: "/"}, true, http.StatusNoContent},
				{simpleFile{filename: "b", path: "/b"}, true, http.StatusNoContent},
				{simpleFile{filename: "a_a", path: "/a"}, true, http.StatusNoContent},
			},
			[]simpleFile{},
		},
	}

	for _, test := range testCases {
		clearDatastore()
		for _, filesToUpload := range test.uploads {

			f := grequests.FileUpload{
				FieldName:    "",
				FileName:     filesToUpload.filename,
				FileContents: ioutil.NopCloser(strings.NewReader("foo")),
			}

			resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
				Files: []grequests.FileUpload{f}, Data: map[string]string{"filename": filesToUpload.filename, "virtfolder": filesToUpload.path}})

			assert.Nil(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend")
		}

		var fsLayout []FSLayout

		for _, filesToDelete := range test.delete {
			resp, _ := grequests.Get(ts.URL+"/auth/list/fs?path="+filesToDelete.simpleFile.path, &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
			resp.JSON(&fsLayout)

			if !filesToDelete.fileExists {
				resp, _ := grequests.Delete(ts.URL+"/auth/file/0", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
				assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
				fmt.Print(resp.String())
			} else {
				for _, fileList := range fsLayout {
					if fileList.Fullpath == filepath.Join(filesToDelete.simpleFile.path, filesToDelete.simpleFile.filename) {
						resp, _ := grequests.Delete(ts.URL+"/auth/file/"+strconv.Itoa(fileList.ID), &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
						assert.Equal(t, http.StatusNoContent, resp.StatusCode)
					}
				}
			}
		}

		if len(test.resultingFiles) > 0 {
			assert.EqualValues(t, test.resultingFiles, fsLayoutToSimpleFile(getAllFSObjectsUsingAPI("/", adminCookie)))
		} else {
			//make sure nothing remains
			for _, f := range test.uploads {
				existingFiles, _ := gc.FileStructDB.ListFiles("admin", f.path)
				assert.Empty(t, existingFiles)
			}
		}
	}
}
