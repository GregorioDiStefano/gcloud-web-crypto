package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

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

func createTestFile(fn, size string) string {
	cmd := exec.Command("dd", "if=/dev/urandom", "of="+fn, "bs="+size, "count=1")
	fmt.Println(cmd)
	err := cmd.Run()

	if err != nil {
		panic(err.Error())
	}

	return fn
}

func TestUploadDownloadFile(t *testing.T) {

	signupDetails := map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	l := new(login)
	resp.JSON(&l)

	testfile := createTestFile("testfile1", "2M")
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

	assert.Equal(t, http.StatusOK, resp.StatusCode, "fail to get files: "+resp.String())

	downloadStruct := make([]DownloadStruct, 0)
	resp.JSON(&downloadStruct)

	resp, err = grequests.Get(ts.URL+"/auth/file/"+strconv.Itoa(downloadStruct[0].Id),
		&grequests.RequestOptions{
			Cookies: []*http.Cookie{{Name: "jwt", Value: l.Token, HttpOnly: true, Secure: false}}})

	ioutil.WriteFile("/var/tmp/decrypted-downloaded-file", resp.Bytes(), 0644)
	cmd := exec.Command("cmp", "/var/tmp/decrypted-downloaded-file", testfile)
	err = cmd.Run()

	if err != nil {
		t.Fatal("downloaded file not same as original")
	}
}

func TestFileIsEncrypted(t *testing.T) {

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

	testfile := createTestFile("testfile2", "2M")
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

	downloadStruct := make([]DownloadStruct, 0)
	resp.JSON(&downloadStruct)

	fileID := downloadStruct[0].Id

	file, err := gc.FileStructDB.GetFile(signupDetails["username"], int64(fileID))
	ctx := context.Background()

	reader, err := gc.StorageBucket.Object(file.GoogleCloudObject).NewReader(ctx)
	user, _, err := gc.UserDB.GetUserEntry(signupDetails["username"])

	c := crypto.NewCryptoData([]byte(signupDetails["password"]), nil, user.Salt, user.Iterations)
	pgpKey, err := c.DecryptText(user.EncryptedPGPKey)

	assert.NoError(t, err, "errored trying to decrypt pgp key")
	assert.NotNil(t, pgpKey, "pgp key is empty!")

	userCrypto := crypto.NewCryptoData(pgpKey, nil, nil, 0)

	df, err := os.OpenFile(testfile+"-decrypted", os.O_CREATE|os.O_WRONLY, 0644)

	assert.NoError(t, err, "failed to create decryption file")
	userCrypto.DecryptFile(reader, df)
	df.Close()

}
