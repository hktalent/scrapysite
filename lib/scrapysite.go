package lib

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
)

type ScrapySite struct {
	Scrapy *colly.Collector
}

// 默认MaxDepth 微0，不受深度限制
// default: IgnoreRobotsTxt
func NewScrapySite() *ScrapySite {
	var r *ScrapySite
	r = &ScrapySite{}
	r.Scrapy = colly.NewCollector(colly.CacheDir("./coursera_cache"), colly.MaxDepth(0), colly.Async(), colly.AllowURLRevisit(), colly.IgnoreRobotsTxt())
	r.Scrapy.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 256})
	return r
}

func (ss *ScrapySite) SetProxys(proxys []string) {
	rp, err := proxy.RoundRobinProxySwitcher(proxys...)
	if err != nil {
		return
	}
	ss.Scrapy.SetProxyFunc(rp)
}

// 请求到url：e.Request.URL.String()
// https://github.com/gocolly/colly/blob/master/_examples/multipart/multipart.go
// e.Request.PostMultipart("http://localhost:8080/", generateFormData())
func (ss *ScrapySite) Init(fnCbk func(string, string, *colly.HTMLElement) bool) {
	// Cache responses to prevent multiple download of pages
	// even if the collector is restarted
	ss.Scrapy.OnHTML("*[href],*[src]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if fnCbk(link, strings.TrimSpace(e.Text), e) {
			// requestIDURL := e.Request.AbsoluteURL(e.ChildAttr(`link[as="script"]`, "href"))
			// log.Println("xxx:", e.Request.AbsoluteURL("")
			if strings.HasPrefix(link, "/") || strings.HasPrefix(link, ".") {
				e.Request.Visit(link)
			} else {
				// log.Println("xxx:", e.Request.AbsoluteURL("")
				ss.Scrapy.Visit(link)
			}
		}
	})
	// Set error handler
	ss.Scrapy.OnError(func(r *colly.Response, err error) {
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

}

// 求hash md5,sha1,sha256
func (ss *ScrapySite) Hash(data []byte) (md5R string, sha1R string, sha256R string) {
	md5R = fmt.Sprintf("%x", md5.Sum(data))
	sha1R = fmt.Sprintf("%x", sha1.Sum(data))
	sha256R = fmt.Sprintf("%x", sha256.Sum256(data))
	return
}

// base64 编码到string
func (ss *ScrapySite) Base64EncodeToString(data []byte) string {
	return b64.StdEncoding.EncodeToString(data)
}

// base64 解码到string
func (ss *ScrapySite) Base64DecodeString(data string) string {
	return string(ss.Base64DecodeString2byte(data))
}

// base64 解码到[]byte
func (ss *ScrapySite) Base64DecodeString2byte(data string) []byte {
	szR, _ := b64.StdEncoding.DecodeString(data)
	return szR
}

func (ss *ScrapySite) OnResponse(onResponse func(r *colly.Response)) {
	ss.Scrapy.OnResponse(func(r *colly.Response) {
		onResponse(r)
		// d := ss.Scrapy.Clone()
		// d.Request("GET", u, nil, ctx, nil)
	})
}

// https://github.com/gocolly/colly/blob/b151a08fbde2b67d960bd9991c1f346e5a1cdd77/_examples/instagram/instagram.go#L94
// r.URL.String()
func (ss *ScrapySite) OnRequest(onRequest func(*colly.Request)) {
	ss.Scrapy.OnRequest(func(r *colly.Request) {
		onRequest(r)
	})
}

// 支持多个url，分隔符号：[;,| ]
func (ss *ScrapySite) Start(szUrl string) {

	re := regexp.MustCompile("[;,| ]")
	url := re.Split(szUrl, -1)

	q, _ := queue.New(
		256, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: 10000}, // Use default queue storage
	)
	for _, x := range url {
		q.AddURL(x)
	}
	// ss.Scrapy.Visit(url)
	q.Run(ss.Scrapy)
	ss.Scrapy.Wait()
}
