package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	gc "github.com/GregorioDiStefano/gcloud-web-crypto"
	"github.com/GregorioDiStefano/gcloud-web-crypto/app/crypto"
	"github.com/gin-gonic/gin"
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

	gin.SetMode(gin.DebugMode)
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

func createTestFile(fn string, size int64) (string, []byte) {
	cmd := exec.Command("openssl", "rand", fmt.Sprintf("%d", size), "-out", fn)
	err := cmd.Run()

	if err != nil {
		panic(err.Error())
	}

	h := sha1.New()
	f, err := os.Open(fn)

	if err != nil {
		log.Fatal(err)
	}

	if _, err = io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	f.Close()
	return fn, h.Sum(nil)
}

func TestDownloadZipFile(t *testing.T) {
	clearDatastore()

	accounts := createAdminAndNormalUsers()
	adminCookie := accounts[0]

	type uploads struct {
		filename string
		filesize int64
		path     string
		hash     []byte
	}

	uploadFiles := []uploads{
		uploads{filename: "a", filesize: 1000, path: "/a/", hash: nil},
		uploads{filename: "c", filesize: 1000, path: "/a/b/c/", hash: nil},
		uploads{filename: "x", filesize: 2000, path: "/a/", hash: nil},
	}

	for idx, uploadFile := range uploadFiles {
		testfile, sha1 := createTestFile(uploadFile.filename, uploadFile.filesize)
		uploadFiles[idx].hash = sha1

		f, err := grequests.FileUploadFromDisk(testfile)
		if err != nil {
			t.Fatalf("failed to upload test file with: " + err.Error())
		}

		resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
			Files: f, Data: map[string]string{"filename": uploadFile.filename, "virtfolder": uploadFile.path}})

		assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend")
	}

	resp, err := grequests.Get(ts.URL+"/auth/folder?path=/a/",
		&grequests.RequestOptions{
			Cookies: []*http.Cookie{adminCookie}})

	if err != nil {
		t.Failed()
	}

	ioutil.WriteFile("/var/tmp/downloaded-zip-file", resp.Bytes(), 0644)
	out, err := exec.Command("zipinfo", "-1", "/var/tmp/downloaded-zip-file").Output()

	if err != nil {
		t.Fail()
	}

	assert.Equal(t, "/a/a\n/a/x\n/a/b/c/c\n", string(out))

	c := exec.Command("unzip", "-o", "downloaded-zip-file")
	c.Dir = "/var/tmp/"
	c.Run()

	for _, uploadFile := range uploadFiles {
		filepath := "/var/tmp" + uploadFile.path + uploadFile.filename
		fmt.Println(filepath)

		f, err := os.Open(filepath)

		if err != nil {
			t.Fail()
		}

		h := sha1.New()
		read, err := io.Copy(h, f)

		fmt.Println(read)
		if err != nil {
			t.Fail()
		}

		f.Close()

		fmt.Println("hash: ", uploadFile.hash)
		assert.Equal(t, uploadFile.hash, h.Sum(nil), "file hash unexpected after unzipping")
	}
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

	testfile, _ := createTestFile("testfile1", 2*1024*1024)
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

	testfile, _ := createTestFile("testfile2", 2*1024*1024)
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

	testfile, _ := createTestFile("testfile3", 2*1024*1024)
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

	testfile, _ := createTestFile("testfile4", 2*1024*1024)
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

	testfile, _ := createTestFile("testfile5", 1*1024*1024)
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
