package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

func TestListing(t *testing.T) {
	clearDatastore()

	users := createAdminAndNormalUsers()
	admin := users[0]

	type uploads struct {
		filename string
		path     string
		tags     []string
	}

	uploadFiles := []uploads{
		uploads{filename: "a_no_tag", path: "/a/", tags: []string{}},
		uploads{filename: "a_a0", path: "/a/", tags: []string{"a"}},
		uploads{filename: "abc0_abc", path: "/a/b/c/", tags: []string{"a", "b", "c"}},
		uploads{filename: "abc1_abc", path: "/a/b/c/", tags: []string{"a", "b", "c"}},
		uploads{filename: "abc2_abc", path: "/a/b/c/", tags: []string{"a", "b", "c"}},
		uploads{filename: "abc3_no_tag", path: "/a/b/c/", tags: []string{}},
		uploads{filename: "abc3_b", path: "/a/b/c/", tags: []string{"b"}},
		uploads{filename: "a_a1", path: "/a/", tags: []string{"a"}},
		uploads{filename: "a_b0", path: "/a/", tags: []string{"b"}},
	}

	for _, uploadFile := range uploadFiles {
		f := grequests.FileUpload{FieldName: "",
			FileName:     uploadFile.filename,
			FileContents: ioutil.NopCloser(strings.NewReader("foo"))}

		resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{admin},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"filename": uploadFile.filename, "virtfolder": uploadFile.path, "tags": strings.Join(uploadFile.tags, ",")}})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend")
	}

	fmt.Println(grequests.Get(ts.URL+"/auth/list/fs", &grequests.RequestOptions{Cookies: []*http.Cookie{admin}, Params: map[string]string{"tags": "a", "path": "/a/"}}))

}
