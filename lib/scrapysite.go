package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"

	// "github.com/gocolly/colly/v2/extensions"
	"github.com/allegro/bigcache/v3"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
)

var nThreads = 16
var Cache *bigcache.BigCache
var EsThreas = make(chan struct{}, nThreads*10)

type ScrapySite struct {
	Scrapy *colly.Collector
	resUrl string
	fnCbk  func(string, string, *colly.HTMLElement) bool
}

func GetCache(k string) (interface{}, error) {
	s, err := Cache.Get(k)
	if nil == err {
		aB := []byte(s)
		var aRs interface{}
		json.Unmarshal(aB, &aRs)
		return aRs, nil
	}
	return nil, err
}

func SetCache(k string, o interface{}) {
	b, err := json.Marshal(o)
	if nil == err {
		Cache.Set(k, b)
	}

}

// 默认MaxDepth 微0，不受深度限制
// default: IgnoreRobotsTxt
func NewScrapySite(resUrl string, fnCbk func(string, string, *colly.HTMLElement) bool) *ScrapySite {
	// Cache = cache.New(10000*time.Hour, 10000*time.Hour)
	Cache1, _ := bigcache.NewBigCache(bigcache.DefaultConfig(10 * time.Minute))
	Cache = Cache1
	var r *ScrapySite
	r = &ScrapySite{resUrl: resUrl, fnCbk: fnCbk}
	// Cache = cache.New(10000*time.Hour, 10000*time.Hour)
	r.Scrapy = colly.NewCollector(colly.CacheDir("./coursera_cache"), colly.MaxDepth(0), colly.Async(), colly.AllowURLRevisit(), colly.IgnoreRobotsTxt())
	r.Scrapy.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: nThreads})
	// extensions.RandomUserAgent(r.Scrapy)
	// extensions.Referer(r.Scrapy)
	r.init()
	return r
}

func I2S(data []interface{}) []string {
	a := []string{}
	for _, j := range data {
		a = append(a, fmt.Sprintf("%v", j))
	}
	return a
}

// 获取domain的所有ip地址
func (r *ScrapySite) GetDomainIps(domain string) (aRst []string) {
	re := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	aRst = []string{}
	if "" != re.FindString(domain) {
		aRst = []string{domain}
	} else {
		oRst, err := GetCache(domain)
		if err == nil && nil != oRst {
			aRst = I2S(oRst.([]interface{}))
			return
		}
		log.Println("GetDomainIps", domain)
		ips, err := net.LookupIP(domain)
		if nil == err {
			for _, ip := range ips {
				if ipv4 := ip.To4(); ipv4 != nil {
					aRst = append(aRst, ipv4.String())
				}
			}
		}
		SetCache(domain, aRst)
	}
	return
}

// 获取ip，ip信息
func (r *ScrapySite) GetDomainInfo(url string) map[string]interface{} {
	s1 := "://"
	if -1 < strings.Index(url, s1) {
		s := strings.Split(url, s1)[1]
		re := regexp.MustCompile(`[;,\?\/:#]`)
		s1 = re.Split(s, -1)[0]
		oRst := make(map[string]interface{})
		aRst := r.GetDomainIps(s1)
		var xxx []map[string]interface{}
		if 0 < len(aRst) {
			for _, x := range aRst {
				xD, err := r.GetIpInfo(x)
				if nil != err {
					log.Println(err)
					continue
				}
				if nil != xD {
					xxx = append(xxx, xD)
				} else {
					xxx = append(xxx, map[string]interface{}{"query": x})
				}
			}
		}
		oRst["ips"] = xxx
		return oRst
	}
	return nil
}

func (ss *ScrapySite) GetIpInfo(ip string) (map[string]interface{}, error) {
	oRst, err := GetCache(ip)
	if nil == err {
		return oRst.(map[string]interface{}), nil
	}
	req, err := http.NewRequest("GET", "http://ip-api.com/json/"+ip, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	// keep-alive
	req.Header.Add("Connection", "close")
	req.Close = true

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close() // resp 可能为 nil，不能读取 Body
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return nil, err
	}
	var rst map[string]interface{}
	json.Unmarshal(body, &rst)
	SetCache(ip, rst)
	return rst, nil
}

func (ss *ScrapySite) SendReq(data *bytes.Buffer, id string) {
	EsThreas <- struct{}{}
	defer func() {
		<-EsThreas
	}()
	req, err := http.NewRequest("POST", ss.resUrl+url.QueryEscape(id), data)
	if err != nil {
		log.Println(err)
		return
	}
	// 取消全局复用连接
	// tr := http.Transport{DisableKeepAlives: true}
	// client := http.Client{Transport: &tr}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
	req.Header.Add("Content-Type", "application/json;charset=UTF-8")
	// keep-alive
	req.Header.Add("Connection", "close")
	req.Close = true
	defer req.Body.Close()

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close() // resp 可能为 nil，不能读取 Body
	}
	if err != nil {
		log.Println("SendReq", ss.resUrl, id, err)
		return
	}
	s2, err := ioutil.ReadAll(resp.Body)
	log.Println("Elasticsearch save ok: ", id, string(s2))
}
func (ss *ScrapySite) SendJsonReq(data map[string]interface{}, id string) {
	jsonValue, err := json.Marshal(data)
	if nil == err {
		ss.SendReq(bytes.NewBuffer(jsonValue), id)
	}
}

// 添加不重复url
func (ss *ScrapySite) AddUrls(url string, data []interface{}) []string {
	a := I2S(data)
	for _, x := range a {
		if x == url {
			return a
		}
	}
	return append(a, url)
}

// 处理非文本文件
func (ss *ScrapySite) DoNotText(r *colly.Response) map[string]interface{} {
	// 非文本，计算sha1、md5
	if nil != r.Body && strings.Index(r.Headers.Get("Content-Type"), "text/") == -1 && 0 < len(r.Body) {
		x1 := r.Body
		md5R, sha1R, sha256R := ss.Hash(x1)
		return map[string]interface{}{"md5": md5R, "sha1": sha1R, "sha256": sha256R}
	}
	return nil
}

func (ss *ScrapySite) DoResponse2Es(r *colly.Response) {
	oPost := make(map[string]interface{})

	szId := r.Request.AbsoluteURL(r.Request.URL.String())
	if "" == szId || 14 > len(szId) {
		return
	}
	oRst, err := GetCache(szId)
	if nil == err && nil != oRst {
		oPost = oRst.(map[string]interface{})
		// spew.Dump(oPost["urls"])
		// interface conversion: interface {} is []interface {}, not []string
		oPost["urls"] = ss.AddUrls(r.Request.URL.String(), oPost["urls"].([]interface{}))
	} else {
		if 200 == r.StatusCode {
			if xx := ss.GetDomainInfo(r.Request.URL.String()); nil != xx {
				oPost = xx
			}
		} else {
			log.Println(r.StatusCode, r.Request.URL.String())
		}
		oPost["urls"] = []string{r.Request.URL.String()}
		if 0 != r.StatusCode {
			xh := r.Headers
			oPost["Headers"] = xh
			oPost["StatusCode"] = r.StatusCode
			szTt, err := GetCache(r.Request.URL.String() + "_title")
			if nil == err && "" != szTt.(string) {
				oPost["title"] = szTt.(string)
			}

		}
	}
	x1 := ss.DoNotText(r)
	if nil != x1 {
		oPost["hash"] = x1
	}

	SetCache(szId, oPost)
	go ss.SendJsonReq(oPost, szId)
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
func (ss *ScrapySite) init() {
	// Cache responses to prevent multiple download of pages
	// even if the collector is restarted
	// Create a callback on the XPath query searching for the URLs
	// ss.Scrapy.OnXML("//urlset/url/loc", func(e *colly.XMLElement) {
	// 	knownUrls = append(knownUrls, e.Text)
	// })
	ss.Scrapy.OnHTML("*[href],*[src]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		title := strings.TrimSpace(e.Text)
		if ss.fnCbk(link, title, e) {
			if "" != title {
				SetCache(link+"_title", title)
			}
			e.Request.Visit(link)
			// if !strings.HasPrefix(link, "http") {
			// } else {
			// 	// log.Println("xxx:", e.Request.AbsoluteURL("")
			// 	ss.Scrapy.Visit(link)
			// }
		}
	})
	// Set error handler
	ss.Scrapy.OnError(func(r *colly.Response, err error) {
		ss.DoResponse2Es(r)
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", "Error:", err)
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
		ss.DoResponse2Es(r)
		// r.Ctx.Get("url")
		onResponse(r)
		// d := ss.Scrapy.Clone()
		// d.Request("GET", u, nil, ctx, nil)
	})
}

// https://github.com/gocolly/colly/blob/b151a08fbde2b67d960bd9991c1f346e5a1cdd77/_examples/instagram/instagram.go#L94
// r.URL.String()
func (ss *ScrapySite) OnRequest(onRequest func(*colly.Request)) {
	ss.Scrapy.OnRequest(func(r *colly.Request) {
		// r.Ctx.Put("url", r.URL.String())
		// log.Println("我的请求", r.URL.String(), r.AbsoluteURL(r.URL.String()), r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
		onRequest(r)
	})
}

// 支持多个url，分隔符号：[;,| ]
func (ss *ScrapySite) Start(szUrl string) {

	re := regexp.MustCompile("[;,| ]")
	url := re.Split(szUrl, -1)

	q, _ := queue.New(
		nThreads, // Number of consumer threads
		&queue.InMemoryQueueStorage{MaxSize: nThreads}, // Use default queue storage
	)
	for _, x := range url {
		q.AddURL(x)
	}
	// ss.Scrapy.Visit(url)
	q.Run(ss.Scrapy)
	ss.Scrapy.Wait()
}
