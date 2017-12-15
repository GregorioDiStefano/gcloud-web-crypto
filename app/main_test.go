package main

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Fullpath   string `json:"fullpath"`
	UploadDate string `json:"upload_date"`
	FileType   string `json:"filetype"`
	FileSize   int    `json:"filesize"`
	Md5        string `json:"md5"`
}

type FSLayout struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	Name       string `json:"name"`
	Fullpath   string `json:"fullpath"`
	UploadDate string `json:"upload_date"`
}

type testUser struct {
	username string
	cookie   *http.Cookie
}

type user map[string]string

var adminLoginDetails user
var normalUserLoginDetails user

func init() {
	adminLoginDetails = map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	normalUserLoginDetails = map[string]string{
		"username": "alice",
		"password": "sdddqs!3482",
	}

	if !strings.Contains(os.Getenv("GOOGLE_CLOUD_STORAGE_BUCKET"), "testing") || !strings.HasPrefix(os.Getenv("DATASTORE_EMULATOR_HOST"), "localhost") {
		panic("GOOGLE_CLOUD_STORAGE_BUCKET must contain 'test' substring and DATASTORE_EMULATOR_HOST must be set to local host when testing")
	}

	gin.SetMode(gin.DebugMode)
}

func clearBucket() {
	ctx := context.Background()
	it := config.storageBucket.Objects(ctx, nil)

	for {
		if obj, err := it.Next(); err != nil {
			return
		} else {
			config.storageBucket.Object(obj.Name).Delete(ctx)
		}
	}
}

func createUser(user map[string]string) *grequests.Response {
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: user})

	if err != nil {
		panic(err)
	}

	return resp
}

func createAdmin() {
	createUser(adminLoginDetails)
}

func createNormalUser() {
	createUser(normalUserLoginDetails)
}

func enableUser(userToEnable user, adminCookie http.Cookie) {
	grequests.Put(ts.URL+"/auth/account/enable/"+userToEnable["username"], &grequests.RequestOptions{Cookies: []*http.Cookie{&adminCookie}})
}

func disableUser(userToEnable user, adminCookie http.Cookie) {
	grequests.Delete(ts.URL+"/auth/account/enable/"+userToEnable["username"], &grequests.RequestOptions{Cookies: []*http.Cookie{&adminCookie}})
}

func loginUser(userdata map[string]string) *http.Cookie {
	// get admin token
	resp, _ := grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: userdata})

	l := new(login)
	resp.JSON(&l)

	cookie := http.Cookie{
		Name:     "jwt",
		Value:    l.Token,
		HttpOnly: true,
		Secure:   false,
	}

	return &cookie
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

func getAllFSObjectsUsingAPI(path string, cookie *http.Cookie) []FSLayout {
	var fsLayout []FSLayout

	resp, _ := grequests.Get(ts.URL+"/auth/list/fs?path="+path, &grequests.RequestOptions{Cookies: []*http.Cookie{cookie}})
	resp.JSON(&fsLayout)

	for _, fsObjects := range fsLayout {
		if fsObjects.Type == "folder" {
			fsLayout = append(fsLayout, getAllFSObjectsUsingAPI(fsObjects.Fullpath, cookie)...)
		}
	}
	return fsLayout
}

func fsLayoutToSimpleFile(fsLayout []FSLayout) []simpleFile {
	var sf []simpleFile

	for _, fs := range fsLayout {
		if fs.Type == "filename" {
			sf = append(sf, simpleFile{path: filepath.Dir(fs.Fullpath), filename: fs.Name})
		}
	}

	return sf
}

//TODO: fix test
func TestDownloadZipFile(t *testing.T) {
	clearDatastore()
	createAdmin()
	adminCookie := loginUser(adminLoginDetails)

	type upload struct {
		filename        string
		filesize        int64
		path            string
		hash            []byte
		existsInZipFile bool
	}

	type testCase struct {
		downloadPath      string
		uploads           []upload
		expectedZipFiles  []string
		expectedErrorCode int
	}

	testCases := []testCase{
		{
			downloadPath: "/a/",
			uploads: []upload{
				upload{filename: "a", filesize: 1000, path: "/a/", hash: nil, existsInZipFile: true},
				upload{filename: "c", filesize: 1000, path: "/a/b/c/", hash: nil, existsInZipFile: true},
				upload{filename: "x", filesize: 2000, path: "/a/", hash: nil, existsInZipFile: true},
			},
			expectedZipFiles: []string{"/a/a", "/a/x", "/a/b/c/c"},
		},

		{
			downloadPath: "/a/b/",
			uploads: []upload{
				upload{filename: "a", filesize: 10, path: "/a/", hash: nil},
				upload{filename: "c", filesize: 10000000, path: "/a/b/c/", hash: nil, existsInZipFile: true},
				upload{filename: "x", filesize: 100, path: "/a/", hash: nil},
			},
			expectedZipFiles: []string{"/a/b/c/c"},
		},

		{
			downloadPath: "/b/",
			uploads: []upload{
				upload{filename: "b", filesize: 1000, path: "/b/", hash: nil, existsInZipFile: true},
				upload{filename: "c", filesize: 1000, path: "/a/b/c/", hash: nil},
				upload{filename: "x", filesize: 2000, path: "/a/", hash: nil},
			},
			expectedZipFiles: []string{"/b/b"},
		},
		{
			downloadPath: "/z/",
			uploads: []upload{
				upload{filename: "b", filesize: 1000, path: "/b/", hash: nil},
				upload{filename: "c", filesize: 1000, path: "/a/b/c/", hash: nil},
				upload{filename: "x", filesize: 2000, path: "/a/", hash: nil},
			},
			expectedErrorCode: http.StatusInternalServerError,
		},
	}

	for _, testCase := range testCases {
		clearDatastore()
		for idx, uploadFile := range testCase.uploads {
			testfile, sha1 := createTestFile(uploadFile.filename, uploadFile.filesize)
			defer os.Remove(testfile)

			testCase.uploads[idx].hash = sha1

			f, err := grequests.FileUploadFromDisk(testfile)
			if err != nil {
				t.Fatalf("failed to upload test file with: " + err.Error())
			}

			resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
				Files: f, Data: map[string]string{"filename": uploadFile.filename, "virtfolder": uploadFile.path}})

			assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend")
		}

		resp, err := grequests.Get(ts.URL+"/auth/folder?path="+testCase.downloadPath,
			&grequests.RequestOptions{
				Cookies: []*http.Cookie{adminCookie}})

		if err != nil {
			t.Failed()
		}

		if testCase.expectedErrorCode != 0 {
			assert.Equal(t, testCase.expectedErrorCode, resp.StatusCode)
		} else {
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			ioutil.WriteFile("/var/tmp/downloaded-zip-file", resp.Bytes(), 0644)
			out, err := exec.Command("zipinfo", "-1", "/var/tmp/downloaded-zip-file").Output()

			if err != nil {
				t.Fail()
			}

			assert.EqualValues(t,
				sort.StringSlice(testCase.expectedZipFiles),
				sort.StringSlice(strings.Split(strings.TrimSpace(string(out)), "\n")))

			c := exec.Command("unzip", "-o", "downloaded-zip-file")
			c.Dir = "/var/tmp/"
			c.Run()

			for _, uploadFile := range testCase.uploads {
				if uploadFile.existsInZipFile {
					filepath := "/var/tmp" + uploadFile.path + uploadFile.filename
					fmt.Println(filepath)

					f, err := os.Open(filepath)

					if err != nil {
						t.Fail()
					}

					h := sha1.New()
					_, err = io.Copy(h, f)

					if err != nil {
						t.Fail()
					}

					f.Close()
					assert.Equal(t, uploadFile.hash, h.Sum(nil), "file hash unexpected after unzipping")
					os.Remove(filepath)
				}
			}
		}
	}
}

func TestDownloadNonExistingFile(t *testing.T) {

	type testCase struct {
		filenameToDownload string
		expectedHTTPStatus int
	}

	tests := []testCase{
		testCase{"abc", http.StatusInternalServerError},
		testCase{"1.1", http.StatusInternalServerError},
		testCase{"100000", http.StatusNotFound},
		testCase{"1", http.StatusNotFound},
		testCase{"-1", http.StatusNotFound},
		testCase{"100000", http.StatusNotFound},
	}

	for _, test := range tests {
		clearDatastore()
		createAdmin()

		cookie := loginUser(adminLoginDetails)

		resp, err := grequests.Get(ts.URL+"/auth/file/"+test.filenameToDownload,
			&grequests.RequestOptions{Cookies: []*http.Cookie{cookie}})

		assert.Nil(t, err)
		assert.Equal(t, test.expectedHTTPStatus, resp.StatusCode)
	}
}

func TestUploadDownloadFile(t *testing.T) {
	clearDatastore()

	type testCases struct {
		filename string
		filepath string
		filesize int64
	}

	tests := []testCases{
		testCases{"foo", "/foo", 1},
		testCases{"foo", "/foo", 20 * 1024 * 1024},
		testCases{"bar", "/bar", 1024},
	}

	for _, test := range tests {
		clearDatastore()

		createAdmin()
		cookie := loginUser(adminLoginDetails)

		testfile, _ := createTestFile(test.filename, test.filesize)
		defer os.Remove(test.filename)

		f, err := grequests.FileUploadFromDisk(testfile)

		if err != nil {
			t.Fatalf("failed to upload test file with: " + err.Error())
		}

		resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Files: f, Cookies: []*http.Cookie{cookie}})
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend successfully")

		resp, err = grequests.Get(ts.URL+"/auth/list/fs?path=/", &grequests.RequestOptions{Cookies: []*http.Cookie{cookie}})

		if err != nil {
			panic(err)
		}

		assert.Equal(t, http.StatusOK, resp.StatusCode, "fail to get files: "+resp.String())

		downloadStruct := make([]DownloadStruct, 0)
		resp.JSON(&downloadStruct)

		resp, err = grequests.Get(ts.URL+"/auth/file/"+strconv.Itoa(downloadStruct[0].ID),
			&grequests.RequestOptions{Cookies: []*http.Cookie{cookie}})

		ioutil.WriteFile("/var/tmp/decrypted-downloaded-file", resp.Bytes(), 0644)
		cmd := exec.Command("cmp", "/var/tmp/decrypted-downloaded-file", testfile)
		err = cmd.Run()

		if err != nil {
			t.Fatal("downloaded file not same as original")
		}

		os.Remove("/var/tmp/decrypted-downloaded-file")
	}
}

func TestFileIsEncrypted(t *testing.T) {
	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: adminLoginDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: adminLoginDetails})
	l := new(login)
	resp.JSON(&l)

	testfile, _ := createTestFile("testfile2", 2*1024*1024)
	defer os.Remove("testfile2")
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

	fileID := downloadStruct[0].ID

	file, err := gc.FileStructDB.GetFile(adminLoginDetails["username"], int64(fileID))
	ctx := context.Background()

	reader, err := gc.StorageBucket.Object(file.GoogleCloudObject).NewReader(ctx)
	user, _, err := gc.UserDB.GetUserEntry(adminLoginDetails["username"])

	c := crypto.NewCryptoData([]byte(adminLoginDetails["password"]), nil, user.Salt, user.Iterations)
	pgpKey, err := c.DecryptText(user.EncryptedPGPKey)

	assert.NoError(t, err, "errored trying to decrypt pgp key")
	assert.NotNil(t, pgpKey, "pgp key is empty!")

	userCrypto := crypto.NewCryptoData(pgpKey, nil, nil, 0)

	df, err := os.OpenFile(testfile+"-decrypted", os.O_CREATE|os.O_WRONLY, 0644)

	assert.NoError(t, err, "failed to create decryption file")
	userCrypto.DecryptFile(reader, df, false)
	df.Close()
}

func TestErrorOnDuplicateFiles(t *testing.T) {

	type testCases struct {
		filename string
		filepath string
	}

	clearDatastore()
	createAdmin()
	adminCookie := loginUser(adminLoginDetails)

	tests := []testCases{
		testCases{"a", "/"},
		testCases{"b", "/b"},
		testCases{"a", "/b"},
	}

	for _, test := range tests {
		f := grequests.FileUpload{
			FieldName:    "",
			FileName:     test.filename,
			FileContents: ioutil.NopCloser(strings.NewReader("foo")),
		}

		resp1, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"virtfolder": test.filepath}})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, resp1.StatusCode, "failed to create folder/file")

		resp2, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"virtfolder": test.filepath}})

		assert.Equal(t, http.StatusConflict, resp2.StatusCode, "no http error when writing over file")
		assert.Contains(t, resp2.String(), "already exists", "nothing in response indicates that this is due to file already existing")

		// make sure using the same filename in a different folder results in no error!
		resp3, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"virtfolder": test.filepath + "/newfolder/"}})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, resp3.StatusCode)
	}
}

func TestFilesUniqueToAccount(t *testing.T) {

	type testCases struct {
		filename string
		path     string
	}

	tests := []testCases{
		testCases{"foo", "/foo"},
		testCases{"bar", "/bar"},
	}

	clearDatastore()

	createAdmin()
	adminCookie := loginUser(adminLoginDetails)

	createNormalUser()
	enableUser(normalUserLoginDetails, *adminCookie)
	normalUserCookie := loginUser(normalUserLoginDetails)

	for _, test := range tests {

		f := grequests.FileUpload{
			FieldName:    "",
			FileName:     test.filename,
			FileContents: ioutil.NopCloser(strings.NewReader("foo")),
		}

		resp1, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"filename": test.filename, "virtfolder": test.path}})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, resp1.StatusCode, "failed to create folder/file")

		resp2, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{normalUserCookie},
			Files: []grequests.FileUpload{f}, Data: map[string]string{"filename": test.filename, "virtfolder": test.path}})

		assert.Nil(t, err)
		assert.Equal(t, http.StatusCreated, resp2.StatusCode)
	}
}

func TestInitialAccount(t *testing.T) {
	type testCase struct {
		adminExists      bool
		expectedHTTPcode int
	}

	tests := []testCase{
		testCase{adminExists: false, expectedHTTPcode: http.StatusNotFound},
		testCase{adminExists: true, expectedHTTPcode: http.StatusNoContent},
	}

	for _, test := range tests {
		clearDatastore()

		if test.adminExists {
			createAdmin()
		}
		resp1, err := grequests.Get(ts.URL+"/account/initial", nil)
		assert.Equal(t, test.expectedHTTPcode, resp1.StatusCode)
		assert.Nil(t, err)
	}
}

func TestVerifyUserContext(t *testing.T) {
	type testCase struct {
		userHasLoggedIn  bool
		userLogin        map[string]string
		expectedHTTPcode int
	}

	tests := []testCase{
		testCase{userHasLoggedIn: false, userLogin: nil, expectedHTTPcode: http.StatusUnauthorized},
		testCase{userHasLoggedIn: true, userLogin: adminLoginDetails, expectedHTTPcode: http.StatusNoContent},
	}

	var cookie *http.Cookie
	for _, test := range tests {
		if test.userHasLoggedIn {
			createAdmin()
			cookie = loginUser(test.userLogin)
		}

		var resp *grequests.Response
		if cookie != nil {
			resp, _ = grequests.Post(ts.URL+"/auth/account/verify", &grequests.RequestOptions{Cookies: []*http.Cookie{cookie}})
		} else {
			resp, _ = grequests.Post(ts.URL+"/auth/account/verify", nil)
		}
		assert.Equal(t, test.expectedHTTPcode, resp.StatusCode)
	}
}

func TestGetUsers(t *testing.T) {

	type testCase struct {
		requestAsAdmin bool

		users         []user
		expectedUsers []string
	}

	tests := []testCase{
		{requestAsAdmin: true,
			expectedUsers: []string{"admin", "greg", "alice", "jack"},
			users: []user{
				user{"username": "greg", "password": "greggreg!234567"},
				user{"username": "alice", "password": "alice12345!"},
				user{"username": "jack", "password": "jack12345!"},
			},
		},
		{requestAsAdmin: false, expectedUsers: []string{"admin", "greg"}, users: []user{user{"username": "greg", "password": "greggreg!234567"}}},
		{requestAsAdmin: true, expectedUsers: []string{"admin"}, users: nil},
	}

	for _, test := range tests {
		clearDatastore()
		createAdmin()

		adminCookie := loginUser(adminLoginDetails)

		for _, u := range test.users {
			createUser(u)
			enableUser(u, *adminCookie)
		}

		if test.requestAsAdmin {
			resp, err := grequests.Get(ts.URL+"/auth/account/users", &grequests.RequestOptions{Cookies: []*http.Cookie{adminCookie}})
			var users []gc.UserEntry
			resp.JSON(&users)

			assert.Equal(t, len(users), len(test.users)+1)
			var usersReturned []string

			for _, u := range users {
				usersReturned = append(usersReturned, u.Username)
			}

			assert.EqualValues(t, test.expectedUsers, usersReturned)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Nil(t, err)
		} else {
			// create a normal user and try requesting list of users
			randomUser := user{"username": "billy_bob", "password": "billy_bob_yay123"}
			createUser(randomUser)
			enableUser(randomUser, *adminCookie)

			normalUserCookie := loginUser(randomUser)
			resp, err := grequests.Get(ts.URL+"/auth/account/users", &grequests.RequestOptions{Cookies: []*http.Cookie{normalUserCookie}})

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
			assert.Nil(t, err)
		}

	}
}

func TestRenameFolder(t *testing.T) {
	type upload struct {
		filename string
		path     string
	}

	type mv struct {
		sourceFolder string
		distFolder   string
	}

	type fs struct {
		_type string
		path  string
	}

	type testCase struct {
		uploads      []upload
		moveSrcDst   mv
		outcome      []fs
		returnsError int
	}

	testCases := []testCase{
		{
			uploads: []upload{
				upload{filename: "a", path: "/a/b/c/"},
				upload{filename: "b", path: "/a/b/c/"},
				upload{filename: "c", path: "/a/b/c/"},
			},
			moveSrcDst: mv{sourceFolder: "/a/b/", distFolder: "/x/"},
			outcome: []fs{
				fs{path: "/x/c/a", _type: "filename"},
				fs{path: "/x/c/b", _type: "filename"},
				fs{path: "/x/c/c", _type: "filename"},
				fs{path: "/x/", _type: "folder"},
				fs{path: "/x/c/", _type: "folder"},
			},
		},
		{
			uploads: []upload{
				upload{filename: "a", path: "/a/b/c/"},
				upload{filename: "b", path: "/a/b/c/"},
				upload{filename: "c", path: "/a/b/c/"},
			},
			moveSrcDst: mv{sourceFolder: "/a/b/c/", distFolder: "/x/"},
			outcome: []fs{
				fs{path: "/x/a", _type: "filename"},
				fs{path: "/x/b", _type: "filename"},
				fs{path: "/x/c", _type: "filename"},
				fs{path: "/x/", _type: "folder"},
			},
		},
		{
			uploads: []upload{
				upload{filename: "a", path: "/a/b/c/"},
				upload{filename: "bar", path: "/a/b/bar/"},
				upload{filename: "b", path: "/a/b/c/"},
				upload{filename: "c", path: "/a/b/c/"},
				upload{filename: "d", path: "/a/b/c/d/"},
			},
			moveSrcDst: mv{sourceFolder: "/a/b/bar/", distFolder: "/x/"},
			outcome: []fs{
				fs{path: "/a/b/c/a", _type: "filename"},
				fs{path: "/a/b/c/b", _type: "filename"},
				fs{path: "/a/b/c/c", _type: "filename"},
				fs{path: "/a/b/c/d/d", _type: "filename"},
				fs{path: "/x/bar", _type: "filename"},
				fs{path: "/x/", _type: "folder"},
				fs{path: "/a/", _type: "folder"},
				fs{path: "/a/b/", _type: "folder"},
				fs{path: "/a/b/c/", _type: "folder"},
				fs{path: "/a/b/c/d/", _type: "folder"},
			},
		},
		{
			uploads: []upload{
				upload{filename: "a", path: "/a/b/"},
				upload{filename: "b", path: "/a/b/c/"},
			},
			moveSrcDst: mv{sourceFolder: "/a/b/", distFolder: "/"},
			outcome: []fs{
				fs{path: "/a", _type: "filename"},
				fs{path: "/c/b", _type: "filename"},
				fs{path: "/c/", _type: "folder"},
			},
		},
		{
			uploads: []upload{
				upload{filename: "a", path: "/a/b/"},
				upload{filename: "b", path: "/a/b/c/"},
			},
			moveSrcDst: mv{sourceFolder: "/x/", distFolder: "/"},
			outcome: []fs{
				fs{path: "/a", _type: "filename"},
				fs{path: "/c/b", _type: "filename"},
				fs{path: "/c/", _type: "folder"},
			},
			returnsError: http.StatusConflict,
		},
	}

	for _, test := range testCases {
		clearDatastore()
		createAdmin()

		cookie := loginUser(adminLoginDetails)

		for _, f := range test.uploads {
			testfile, _ := createTestFile(f.filename, 1)
			defer os.Remove(testfile)

			uploadFile, err := grequests.FileUploadFromDisk(testfile)

			resp, err := grequests.Post(ts.URL+"/auth/file", &grequests.RequestOptions{Cookies: []*http.Cookie{cookie},
				Files: uploadFile, Data: map[string]string{"filename": f.filename, "virtfolder": f.path}})

			assert.Equal(t, http.StatusCreated, resp.StatusCode, "failed to upload file to backend successfully")

			if err != nil {
				t.Fatalf("failed to upload test file with: " + err.Error())
			}
		}

		resp, err := grequests.Patch(ts.URL+"/auth/folder", &grequests.RequestOptions{Cookies: []*http.Cookie{cookie},
			JSON: map[string]string{"src": test.moveSrcDst.sourceFolder, "dst": test.moveSrcDst.distFolder}})

		if err != nil {
			panic(err)
		}

		if test.returnsError != 0 {
			assert.Equal(t, test.returnsError, resp.StatusCode)
			break
		}

		files := getAllFSObjectsUsingAPI("/", cookie)
		assert.Len(t, files, len(test.outcome))

		count := 0
		for _, actualFile := range files {
			for _, expectedFile := range test.outcome {
				if expectedFile.path == actualFile.Fullpath &&
					expectedFile._type == actualFile.Type {
					count++
				}
			}
		}
		assert.Equal(t, len(test.outcome), count)
	}
}

//TODO: make sure to make some requests with a broken/non-existant context
func TestGetUserFromContext(t *testing.T) {
	type testCase struct {
		u                user
		successfullLogin bool
		expectedError    bool
	}

	tests := []testCase{
		testCase{adminLoginDetails, true, false},
		testCase{user{"username": "alice", "password": "alice"}, false, true},
	}

	for _, test := range tests {
		if !test.successfullLogin {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			getUserFromContext(c)
			if test.expectedError {
				assert.True(t, c.IsAborted())
			}
		} else {
			createAdmin()
			loginUser(test.u)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			getUserFromContext(c)
			fmt.Println(c.Get("user"))
		}
	}
}
