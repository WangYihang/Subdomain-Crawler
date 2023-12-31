package util_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/WangYihang/Subdomain-Crawler/pkg/util"
)

func TestExtractDomains(t *testing.T) {
	type args struct {
		body []byte
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test1",
			args: args{
				body: []byte("dhsjkalhfjklh.nxs.,cnd,.f/tsinghua.edu.cn|dds.d/"),
			},
			want: []string{"tsinghua.edu.cn", "dds.d"},
		},
		{
			name: "test2",
			args: args{
				body: []byte("dhsjkalhfjklh.nxs.,cnd,.f/tsinghua.edu.cn|dds.d/%2fgithub.com"),
			},
			want: []string{"tsinghua.edu.cn", "dds.d", "github.com"},
		},
		{
			name: "test3",
			args: args{
				body: []byte(`
				<li style="float:left;"><a href="https://classx.pku.edu.cn/cloudCourse/#/index" style="font-size:14px;">燕云直播</a></li>
				<li style="float:left;"><a href="https://mail.pku.edu.cn/" style="font-size:14px;">邮箱</a></li>
				<li style="float:left;"><a href="https://portal.pku.edu.cn" style="font-size:14px;">门户</a></li>
				<li><a href="https://classx.pku.edu.cn/cloudCourse/#/index" style="margin-right:10px;">燕云直播</a></li>
				`),
			},
			want: []string{"classx.pku.edu.cn", "mail.pku.edu.cn", "portal.pku.edu.cn", "classx.pku.edu.cn"},
		},
		{
			name: "test4",
			args: args{
				body: []byte(`
				<li class="yui3-u sm-footer-navitem sm-footer-navitem-login sm-login-button" data-nav="login"><a href="https://secure.smugmug.com/login?goTo=https%3A%2F%2Fm.smugmug.com%2F&goToToken=eyJzdHJpbmciOiJodHRwczovL20uc211Z211Zy5jb20vIiwidGltZSI6MTY5MDE3MzE3Miwic2lnbmF0dXJlIjoiTnpjM01HWTBNV0l5T0RrellUazFOMk5sWldWbU5ERmtOVFprWVRCa1pEUTJObVF4TXpKbVlRPT0iLCJ2ZXJzaW9uIjoxLCJhbGdvcml0aG0iOiJzaGExIn0%3D">Owner Log In</a></li>
				`),
			},
			want: []string{"secure.smugmug.com", "m.smugmug.com"},
		},
		{
			name: "test5",
			args: args{
				body: []byte(`=https%3A%2F%2Fm.smugmug.com%2F&`),
			},
			want: []string{"m.smugmug.com"},
		},
		{
			name: "test6",
			args: args{
				body: []byte(`=https%3A%2F%2F%m.smugmug.com%2F&`),
			},
			want: []string{"m.smugmug.com"},
		},
		{
			name: "test7",
			args: args{
				body: []byte(`=https%3A%2F%2F%%m.smugmug.com%2F&`),
			},
			want: []string{"m.smugmug.com"},
		},
		{
			name: "test8",
			args: args{
				body: []byte(`=https%3A%2F%2F%%4xm.smugmug.com%2F&`),
			},
			want: []string{"4xm.smugmug.com"},
		},
		{
			name: "test9",
			args: args{
				body: []byte(`=https%3A%2F%2F%%44m.smugmug.com%2F&`),
			},
			want: []string{"m.smugmug.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := []string{}
			reader := io.NopCloser(bytes.NewReader(tt.args.body))
			for domain := range util.ExtractDomains(reader) {
				domains = append(domains, domain)
			}
			if got := domains; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractDomains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomainBuilder(t *testing.T) {
	domainBuilder := util.DomainBuilder{}
	domainBuilder.Append('a')
	domainBuilder.Append('b')
	domainBuilder.Append('c')
	domainBuilder.Append('d')
	if domainBuilder.String() != "abcd" {
		t.Errorf("DomainBuilder.String() = %v, want %v", domainBuilder.String(), "abcd")
	}
}

func BenchmarkDomainBuilderString(b *testing.B) {
	domainBuilder := util.DomainBuilder{}
	for i := 0; i < b.N; i++ {
		domainBuilder.Append('a')
		domainBuilder.Append('b')
		domainBuilder.Append('c')
		domainBuilder.Append('d')
		_ = domainBuilder.String()
	}
}

func BenchmarkDomainBuilderStringSlow(b *testing.B) {
	domainBuilder := util.DomainBuilder{}
	for i := 0; i < b.N; i++ {
		domainBuilder.Append('a')
		domainBuilder.Append('b')
		domainBuilder.Append('c')
		domainBuilder.Append('d')
		_ = domainBuilder.StringSlow()
	}
}

func BenchmarkDomainBuilderStringQuick(b *testing.B) {
	domainBuilder := util.DomainBuilder{}
	for i := 0; i < b.N; i++ {
		domainBuilder.Append('a')
		domainBuilder.Append('b')
		domainBuilder.Append('c')
		domainBuilder.Append('d')
		_ = domainBuilder.StringUnsafe()
	}
}

func BenchmarkExtractDomains(b *testing.B) {
	for i := 0; i < b.N; i++ {
		reader := io.NopCloser(bytes.NewReader([]byte("dhsjkalhfjklh.nxs.,cnd,.f/tsinghua.edu.cn|dds.d/%2fgithub.com")))
		util.ExtractDomains(reader)
	}
}
