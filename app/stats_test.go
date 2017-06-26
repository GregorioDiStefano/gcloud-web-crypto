package main

import (
	"net/http"
	"testing"

	"github.com/levigross/grequests"
)

func getStats(userToEnable user, cookie http.Cookie) {
	grequests.Get(ts.URL+"/auth/account/stat/", &grequests.RequestOptions{Cookies: []*http.Cookie{&cookie}})
}

func TestStats(t *testing.T) {
	clearDatastore()
	createAdmin()
	cookie := loginUser(adminLoginDetails)
	getStats(adminLoginDetails, *cookie)
}
