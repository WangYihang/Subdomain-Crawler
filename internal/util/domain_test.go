package util_test

import (
	"reflect"
	"testing"

	"github.com/WangYihang/Subdomain-Crawler/internal/util"
)

func TestMatchDomains(t *testing.T) {
	testCases := []struct {
		body     []byte
		expected []string
	}{
		{
			[]byte(`<a class="nav-card mr20" href="https://hao.360.com/" target="_blank">a.360.com<a class="nav-card mr20" href="https://yule.360.com/" target="_blank">`),
			[]string{
				"hao.360.com",
				"a.360.com",
				"yule.360.com",
			},
		},
	}
	for _, tc := range testCases {
		got := util.MatchDomains(tc.body)
		t.Logf("util.MatchDomains(%s) = %v", tc.body, got)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Expected %s but got %s", tc.expected, got)
		}
	}
}

func TestFilterDomain(t *testing.T) {
	testCases := []struct {
		domains  []string
		root     string
		expected []string
	}{
		{
			[]string{
				"www.tsinghua.edu.cn",
				"join-tsinghua.edu.cn",
				"www.join-tsinghua.edu.cn",
				"www.baidu.com",
			},
			"tsinghua.edu.cn",
			[]string{
				"www.tsinghua.edu.cn",
				"join-tsinghua.edu.cn",
				"www.join-tsinghua.edu.cn",
			},
		},
	}
	for _, tc := range testCases {
		got := util.FilterDomain(tc.domains, tc.root)
		t.Logf("util.FilterDomain(%v, %v) = %v", tc.domains, tc.root, got)
		if !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("Expected %s but got %s", tc.expected, got)
		}
	}
}
