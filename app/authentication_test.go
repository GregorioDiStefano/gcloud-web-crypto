package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
)

var ts *httptest.Server

type login struct {
	Expire string `json:"expire"`
	Token  string `json:"token"`
}

func init() {
	clearDatastore()
	ts = httptest.NewServer(mainGinEngine())
}

func clearDatastore() {
	host := os.Getenv("DATASTORE_EMULATOR_HOST")
	grequests.Post("http://"+host+"/reset", nil)
}

func TestAdminNeededFirst(t *testing.T) {
	signupDetails := map[string]string{
		"username": "greg",
		"password": "sdfiopdnndsajiiwqqs3482",
	}
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "expected http forbidden")
	// check if admin messsage exists
}

func TestCreateAdminAndLogin(t *testing.T) {
	signupDetails := map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "didn't login successfully")
}

func TestIncorrectPassword(t *testing.T) {
	signupDetails := map[string]string{
		"username": "admin",
		"password": "b033377c-682a-41cc-be87-4578339f1adf",
	}

	signupDetailsWrongPassword := map[string]string{
		"username": "admin",
		"password": "B033377C-682A-41CC-BE87-4578339F1ADF",
	}

	clearDatastore()

	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "didn't login successfully")

	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetailsWrongPassword})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "didn't get http unauthorized")

}

func TestNonExistingAccount(t *testing.T) {
	signupDetails := map[string]string{
		"username": "greg",
		"password": "b033377c-682a-41cc-be87-4578339f1adf",
	}

	clearDatastore()
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "didn't get http unauthorized")
}

func TestDuplicateAccount(t *testing.T) {
	type testCase struct {
		u user
	}

	tests := []testCase{
		testCase{adminLoginDetails},
		testCase{normalUserLoginDetails},
	}

	clearDatastore()

	for _, test := range tests {
		resp := createUser(test.u)
		assert.Equal(t, http.StatusCreated, resp.StatusCode, "didn't login successfully"+resp.String())
		resp = createUser(test.u)
		assert.Equal(t, http.StatusConflict, resp.StatusCode, "didn't login successfully")
	}
}

//TODO: cleanup this mess
func TestNonAdminAccount(t *testing.T) {

	signupDetails := map[string]string{
		"username": "greg",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	adminSignupDetails := map[string]string{
		"username": "admin",
		"password": "sdfiopdnndsajiiwqqs3482",
	}

	clearDatastore()

	// create an admin account
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: adminSignupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// get admin token
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: adminSignupDetails})
	adminLogin := new(login)
	resp.JSON(&adminLogin)

	// create an account
	resp, err = grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "expected http accepted")

	// shouldn't be able to login
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "login successfully when account is not enabled")

	// enable non-admin account using admin token
	resp, err = grequests.Put(ts.URL+"/auth/account/enable/"+signupDetails["username"], &grequests.RequestOptions{
		Cookies: []*http.Cookie{
			{
				Name:     "jwt",
				Value:    adminLogin.Token,
				HttpOnly: true,
				Secure:   false,
			}}})

	assert.Nil(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "login not successful")

	// finally, login to non-admin account
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Nil(t, err)

	// disable user
	disableUser(signupDetails, http.Cookie{
		Name:     "jwt",
		Value:    adminLogin.Token,
		HttpOnly: true,
		Secure:   false,
	})

	// finally, login to non-admin account
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestVerifyCaptcha(t *testing.T) {
	type googleResponse struct {
		Success    bool
		ErrorCodes []string `json:"error-codes"`
	}

	captchaRouter := gin.Default()

	captchaRouter.POST("/fake_captcha_acceptor", func(c *gin.Context) {
		c.JSON(http.StatusOK, googleResponse{true, nil})
	})

	captchaRouter.POST("/fake_captcha_denier", func(c *gin.Context) {
		c.JSON(http.StatusOK, googleResponse{false, nil})
	})

	captchaServer := httptest.NewServer(captchaRouter)
	config.googleCaptchaURL = captchaServer.URL + "/fake_captcha_acceptor"

	fmt.Println(verifyGoogleCaptcha("abc"))

	config.googleCaptchaURL = captchaServer.URL + "/fake_captcha_denier"

	fmt.Println(verifyGoogleCaptcha("abc"))
}

func TestPasswords(t *testing.T) {
	clearDatastore()
	createAdmin()

	type testCase struct {
		u             user
		validUsername bool
		validPassword bool
	}

	tests := []testCase{
		{user{"username": "greg", "password": "greg"}, true, false},
		{user{"username": "greg0", "password": ""}, true, false},
		{user{"username": "greg0", "password": "               "}, true, false},
		{user{"username": "greg1", "password": "abcd12345!"}, true, true},
		{user{"username": "greg2", "password": strings.Repeat("abcd12345", 100)}, true, true},
		{user{"username": " ", "password": strings.Repeat("abcd12345", 100)}, false, true},
		{user{"username": "", "password": strings.Repeat("abcd12345", 100)}, false, true},
		{user{"username": " greg", "password": strings.Repeat("abcd12345", 100)}, false, true},
	}

	for _, test := range tests {
		r := createUser(test.u)

		if !test.validUsername {
			assert.JSONEq(t, fmt.Sprintf("{\"status\":\""+errWeakUsername+"\"}"), r.String())
			assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		} else if !test.validPassword {
			assert.JSONEq(t, fmt.Sprintf("{\"status\":\""+errWeakPassword+"\"}"), r.String())
			assert.Equal(t, http.StatusUnauthorized, r.StatusCode)
		} else {
			assert.Equal(t, http.StatusCreated, r.StatusCode)
		}

	}
}
