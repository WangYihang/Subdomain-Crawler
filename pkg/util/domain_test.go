package util_test

import (
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
				<li><a href="https://mail.pku.edu.cn/" style="margin-right:10px;">邮箱</a></li>
				<li><a href="https://portal.pku.edu.cn" target="_blank" style="margin-right:10px;">门户</a></li>
				<li><a href="https://its4.pku.edu.cn/">旧版</a></li>
				<a href="http://www.pku.edu.cn/" title="北京大学"><img src="/img/pkulogo_white.png"></a>
				<li><a id="center" href="https://card.pku.edu.cn/" class="nav_phone_style">校园卡</a></li>
				<li><a id="hpc" href="http://hpc.pku.edu.cn" class="nav_hpc"><span class="nav_long">高性能计算</span><span class="nav_short">高性能</span></a></li>
				<div style="width:100%;text-align:center;line-height:20px;float:left;padding-top:10px;padding-bottom:20px" class="changefont"><a href="https://its.pku.edu.cn/service_1_vpn_readme.jsp"><span>VPN登录</span></a>&nbsp;&nbsp;&nbsp;<a href="https://wpn.pku.edu.cn/"><span>WPN登录</span></a><!--&nbsp;&nbsp;&nbsp;<a href="https://vpns.pku.edu.cn/"><span>VPNS登录</span></a>-->&nbsp;&nbsp;&nbsp;<a href="https://ds.carsi.edu.cn/Shibboleth.sso/Login?entityID=https://idp.pku.edu.cn/idp/shibboleth&target=https%3A%2F%2Fds.carsi.edu.cn%2Fresource%2Fresource.php"><span>CARSI登录</span></a></div>
										<a href="https://news.pku.edu.cn/xwzh/ed44e679503f4de58f2d42d765e57dde.htm" target="_blank"-->
										<a href="https://news.pku.edu.cn/xwzh/6d37e6c750354afba4d7e5043f6611b4.htm" target="_blank"-->
										<a href="https://its.pku.edu.cn/announce/tz20230428135700.jsp" target="_blank">
										<a href="https://its.pku.edu.cn/announce/tz20230428105741.jsp" target="_blank">
										<a href="https://its.pku.edu.cn/announce/tz20230417205741.jsp" target="_blank">
										<a href="https://news.pku.edu.cn/xwzh/555ad7c97a6340be86d760ef4359452b.htm" target="_blank">
										<a href="https://its.pku.edu.cn/announce/tz20221114162933.jsp" target="_blank">
										<a href="https://news.pku.edu.cn/xwzh/6f3125b3250d4ae4af26fa6d5797677c.htm" target="_blank">
				<li><span class="s-wd pku-red">07-07</span><a href="https://news.pku.edu.cn/xwzh/ed44e679503f4de58f2d42d765e57dde.htm">北大全球学术资源共享平台CARSI让师生学习工作更加智慧便捷</a></li>
				<li><span class="s-wd pku-red">05-10</span><a href="https://news.pku.edu.cn/xwzh/6d37e6c750354afba4d7e5043f6611b4.htm">北京大学夺取第十届世界大学生超级计算机竞赛总冠军</a></li>
				<li><span class="s-wd pku-red">04-28</span><a href="https://its.pku.edu.cn/announce/tz20230428135700.jsp">北京大学pkucc战队斩获第六届“强网杯”人工智能挑战赛冠军</a></li>
				<li><span class="s-wd pku-red">04-28</span><a href="https://its.pku.edu.cn/announce/tz20230428105741.jsp">第零届北京大学高性能计算综合能力竞赛圆满结束！</a></li>
		<a href="https://news.pku.edu.cn/xwzh/555ad7c97a6340be86d760ef4359452b.htm" target="_blank"><img class="cap-img" src="/img/news/news20230225.jpg"/></a>
				<p class="title"><a href="https://news.pku.edu.cn/xwzh/555ad7c97a6340be86d760ef4359452b.htm" ptarget="_blank">高校数据共享应用签约仪式暨工作推进研讨会在北大举行</a> </p>
				<li id="app-3"><a href="https://its.pku.edu.cn/oper/forbidden.htm" onmouseover="iconChang('3')" onmouseout="iconChang2('3')"><span class="app-ico">
				`),
			},
			want: []string{"tsinghua.edu.cn", "dds.d", "github.com"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domains := []string{}
			for domain := range util.ExtractDomains(tt.args.body) {
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

func BenchmarkExtractDomains(b *testing.B) {
	for i := 0; i < b.N; i++ {
		util.ExtractDomains([]byte("dhsjkalhfjklh.nxs.,cnd,.f/tsinghua.edu.cn|dds.d/%2fgithub.com"))
	}
}
