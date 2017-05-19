package main

import (
	"net/http"
	"os/exec"
	"strconv"
	"testing"

	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

func createTestFile() string {
	cmd := exec.Command("dd", "if=/dev/urandom", "of=1MB", "bs=1M", "count=1")
	err := cmd.Run()

	if err != nil {
		panic(err)
	}

	return "1MB"
}

func TestUploadDownloadFile(t *testing.T) {

	signupDetails := map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	clearDatastore()
	defer clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	l := new(login)
	resp.JSON(&l)

	testfile := createTestFile()
	f, err := grequests.FileUploadFromDisk(testfile)
	if err != nil {
		t.Fatalf("failed to upload test file with: " + err.Error())
	}

	resp, err = grequests.Post(ts.URL+"/auth/file",
		&grequests.RequestOptions{Files: f,
			Cookies: []*http.Cookie{{Name: "jwt", Value: l.Token, HttpOnly: true, Secure: false}}})

	assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend successfully")

	resp, err = grequests.Get(ts.URL+"/auth/list/fs?path=/",
		&grequests.RequestOptions{
			Cookies: []*http.Cookie{{Name: "jwt", Value: l.Token, HttpOnly: true, Secure: false}}})

	if err != nil {
		panic(err)
	}

	type DownloadStruct struct {
		Id         int    `json:"id"`
		Type       string `json:"type"`
		Name       string `json:"name"`
		Fullpath   string `json:"fullpath"`
		UploadDate string `json:"upload_date"`
		FileType   string `json:"filetype"`
		FileSize   int    `json:"filesize"`
		Md5        string `json:"md5"`
	}

	downloadStruct := make([]DownloadStruct, 0)
	resp.JSON(&downloadStruct)

	resp, err = grequests.Get(ts.URL+"/auth/file/"+strconv.Itoa(downloadStruct[0].Id),
		&grequests.RequestOptions{
			Cookies: []*http.Cookie{{Name: "jwt", Value: l.Token, HttpOnly: true, Secure: false}}})

}
