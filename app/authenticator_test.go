package main

import (
	"github.com/levigross/grequests"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
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
	signupDetails := map[string]string{
		"username": "admin",
		"password": "b033377c682a41ccbe874578339f1adf",
	}

	clearDatastore()
	resp, err := grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode, "didn't login successfully"+resp.String())

	resp, err = grequests.Post(ts.URL+"/account/signup", &grequests.RequestOptions{JSON: signupDetails})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode, "didn't login successfully")
}

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
	l := new(login)
	resp.JSON(&l)

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
				Value:    l.Token,
				HttpOnly: true,
				Secure:   false,
			}}})

	assert.Nil(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode, "login not successful")

	// finally, login to non-admin account
	resp, err = grequests.Post(ts.URL+"/account/login", &grequests.RequestOptions{JSON: signupDetails})
	resp.JSON(&l)
	assert.Nil(t, err)
}
