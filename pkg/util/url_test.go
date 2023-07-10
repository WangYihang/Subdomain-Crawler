package util_test

import (
	"testing"

	"github.com/WangYihang/Subdomain-Crawler/pkg/util"
)

func TestUrlDecode(t *testing.T) {
	testCases := []struct {
		body         string
		expectedBody string
	}{
		{"%25%2f%2f", "%//"},
		{"my%2Fcool+blog&about%2Cstuff", "my/cool+blog&about,stuff"},
		{"my%2Fcool+blog&about%2Cstuff%20", "my/cool+blog&about,stuff "},
		{"%2fcool+blog&about%2Cstuff%20", "/cool+blog&about,stuff "},
	}
	for _, tc := range testCases {
		gotBody := util.URLDecode(tc.body)
		t.Logf("util.UrlDecode(%s) = %v", tc.body, gotBody)
		if gotBody != tc.expectedBody {
			t.Errorf("Expected %s but got %s", tc.expectedBody, gotBody)
		}
	}
}
