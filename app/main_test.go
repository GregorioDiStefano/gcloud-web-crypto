package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

var adminSignDetails = map[string]string{}
var normalSignDetails = map[string]string{}

func init() {
	adminSignDetails = map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	normalSignDetails = map[string]string{
		"username": "alice",
		"password": "sdddqs!3482",
	}

	if !strings.Contains(os.Getenv("GOOGLE_CLOUD_STORAGE_BUCKET"), "testing") || !strings.HasPrefix(os.Getenv("DATASTORE_EMULATOR_HOST"), "localhost") {
		panic("GOOGLE_CLOUD_STORAGE_BUCKET must contain 'test' substring and DATASTORE_EMULATOR_HOST must be set to local host when testing")
	}
}

func createAdminAndNormalUsers() []*http.Cookie {
	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: adminSignDetails})

	if err != nil {
		panic(err)
	}

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: adminSignDetails})
	l := new(login)
	resp.JSON(&l)
	adminCookie := []*http.Cookie{
		{
			Name:     "jwt",
			Value:    l.Token,
			HttpOnly: true,
			Secure:   false,
		}}

	// create an account
	resp, err = grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: normalSignDetails})

	// enable non-admin account using admin token
	resp, err = grequests.Put(ts.URL+"/auth/account/enable/"+normalSignDetails["username"], &grequests.RequestOptions{Cookies: adminCookie})

	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: normalSignDetails})
	normalLogin := new(login)
	resp.JSON(&normalLogin)
	normalUserCookie := []*http.Cookie{
		{
			Name:     "jwt",
			Value:    normalLogin.Token,
			HttpOnly: true,
			Secure:   false,
		}}

	return []*http.Cookie{adminCookie[0], normalUserCookie[0]}
}

func createTestFile(fn string, size int64) string {
	cmd := exec.Command("openssl", "rand", fmt.Sprintf("%d", size), "-out", fn)
	err := cmd.Run()

	if err != nil {
		panic(err.Error())
	}

	return fn
}

func TestUploadDownloadFile(t *testing.T) {

	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: adminSignDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: adminSignDetails})
	l := new(login)
	resp.JSON(&l)

	testfile := createTestFile("testfile1", 2*1024*1024)
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
	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: adminSignDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: adminSignDetails})
	l := new(login)
	resp.JSON(&l)

	testfile := createTestFile("testfile2", 2*1024*1024)
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

	file, err := gc.FileStructDB.GetFile(adminSignDetails["username"], int64(fileID))
	ctx := context.Background()

	reader, err := gc.StorageBucket.Object(file.GoogleCloudObject).NewReader(ctx)
	user, _, err := gc.UserDB.GetUserEntry(adminSignDetails["username"])

	c := crypto.NewCryptoData([]byte(adminSignDetails["password"]), nil, user.Salt, user.Iterations)
	pgpKey, err := c.DecryptText(user.EncryptedPGPKey)

	assert.NoError(t, err, "errored trying to decrypt pgp key")
	assert.NotNil(t, pgpKey, "pgp key is empty!")

	userCrypto := crypto.NewCryptoData(pgpKey, nil, nil, 0)

	df, err := os.OpenFile(testfile+"-decrypted", os.O_CREATE|os.O_WRONLY, 0644)

	assert.NoError(t, err, "failed to create decryption file")
	userCrypto.DecryptFile(reader, df)
	df.Close()
}

func TestErrorOnDuplicateFiles(t *testing.T) {
	clearDatastore()

	accounts := createAdminAndNormalUsers()

	adminCookie := accounts[0]
	//userCookie := accounts[1]

	testfile := createTestFile("testfile3", 2*1024*1024)
	f, err := grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp1, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
		Files: f, Data: map[string]string{"filename": "/abc/12345/test"}})

	assert.Equal(t, http.StatusCreated, resp1.StatusCode, "failed to create folder/file")

	f, err = grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp2, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
		Files: f, Data: map[string]string{"filename": "/abc/12345/test"}})

	assert.Equal(t, http.StatusConflict, resp2.StatusCode, "no http error when writing over file")
	assert.Contains(t, resp2.String(), "already exists", "nothing in response indicates that this is due to file already existing")
}

func TestFilesUniqueToAccount(t *testing.T) {
	clearDatastore()

	accounts := createAdminAndNormalUsers()

	adminCookie := accounts[0]
	userCookie := accounts[1]
	fmt.Println(adminCookie, userCookie)

	testfile := createTestFile("testfile4", 2*1024*1024)
	f, err := grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp1, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
		Files: f, Data: map[string]string{"filename": "/abc/12345/test"}})

	assert.Equal(t, http.StatusCreated, resp1.StatusCode, "failed to create folder/file")

	f, err = grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp2, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{userCookie},
		Files: f, Data: map[string]string{"filename": "/abc/12345/test"}})

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)
}

func TestFileListing(t *testing.T) {
	clearDatastore()

	accounts := createAdminAndNormalUsers()

	adminCookie := accounts[0]
	userCookie := accounts[1]

	testfile := createTestFile("testfile5", 1*1024*1024)
	f, err := grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp1, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
		Files: f, Data: map[string]string{"virtfolder": "/abc/12345/test/admin/"}})

	assert.Equal(t, http.StatusCreated, resp1.StatusCode, "failed to create folder/file")

	f, err = grequests.FileUploadFromDisk(testfile)

	if err != nil {
		panic(err)
	}

	resp2, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{userCookie},
		Files: f, Data: map[string]string{"virtfolder": "/abc/12345/test/greg/"}})

	assert.Equal(t, http.StatusCreated, resp2.StatusCode)

	listRespAdmin, err := grequests.Get(ts.URL+"/auth/list/fs?path=/abc/12345/test/admin/", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
	adminFile := make([]DownloadStruct, 0)
	listRespAdmin.JSON(&adminFile)
	assert.Equal(t, testfile, adminFile[0].Name)

	listRespGreg, err := grequests.Get(ts.URL+"/auth/list/fs?path=/abc/12345/test/greg/", &grequests.RequestOptions{Cookies: []*http.Cookie{userCookie}})
	userFile := make([]DownloadStruct, 0)
	listRespGreg.JSON(&userFile)
	assert.Equal(t, testfile, userFile[0].Name)
}
