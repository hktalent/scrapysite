package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
	"github.com/patrickmn/go-cache"
)

type ScrapySite struct {
	Scrapy *colly.Collector
	Cache  *cache.Cache
	resUrl string
	fnCbk  func(string, string, *colly.HTMLElement) bool
}

// 默认MaxDepth 微0，不受深度限制
// default: IgnoreRobotsTxt
func NewScrapySite(resUrl string, fnCbk func(string, string, *colly.HTMLElement) bool) *ScrapySite {
	var r *ScrapySite
	r = &ScrapySite{resUrl: resUrl, fnCbk: fnCbk}
	r.Cache = cache.New(10000*time.Hour, 10000*time.Hour)
	r.Scrapy = colly.NewCollector(colly.CacheDir("./coursera_cache"), colly.MaxDepth(0), colly.Async(), colly.AllowURLRevisit(), colly.IgnoreRobotsTxt())
	r.Scrapy.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 256})
	extensions.RandomUserAgent(r.Scrapy)
	extensions.Referer(r.Scrapy)
	r.init()
	return r
}

// 获取domain的所有ip地址
func (r *ScrapySite) GetDomainIps(domain string) (aRst []string) {
	re := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	aRst = []string{}
	if "" != re.FindString(domain) {
		aRst = []string{domain}
		return
	}
	ips, err := net.LookupIP(domain)
	if nil == err {
		for _, ip := range ips {
			if ipv4 := ip.To4(); ipv4 != nil {
				aRst = append(aRst, ipv4.String())
			}
		}
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
		oRst["url"] = url
		aRst := r.GetDomainIps(s1)
		var xxx []map[string]interface{}
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
		oRst["ips"] = xxx
		return oRst
	}
	return nil
}

func (ss *ScrapySite) GetIpInfo(ip string) (map[string]interface{}, error) {
	oRst, bFound := ss.Cache.Get(ip)
	if bFound {
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
	ss.Cache.Set(ip, rst, cache.NoExpiration)
	return rst, nil
}

func (ss *ScrapySite) SendReq(data *bytes.Buffer, id string) {
	req, err := http.NewRequest("POST", ss.resUrl+id, data)
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

	resp, err := http.DefaultClient.Do(req)
	if resp != nil {
		defer resp.Body.Close() // resp 可能为 nil，不能读取 Body
	}
	if err != nil {
		log.Println("SendReq", ss.resUrl, id, err)
		return
	}
	_, err = io.Copy(ioutil.Discard, resp.Body) // 手动丢弃读取完毕的数据
	log.Println("Elasticsearch save ok: ", ss.resUrl, id)
	// defer req.Body.Close()
}
func (ss *ScrapySite) SendJsonReq(data map[string]interface{}, id string) {
	jsonValue, err := json.Marshal(data)
	if nil == err {
		ss.SendReq(bytes.NewBuffer(jsonValue), id)
	}
}
func (ss *ScrapySite) DoResponse2Es(r *colly.Response) {
	oPost := make(map[string]interface{})

	szId := r.Request.AbsoluteURL(r.Request.URL.String())
	oRst, bFound := ss.Cache.Get(szId)
	if bFound {
		oPost = oRst.(map[string]interface{})
	} else {
		if 200 != r.StatusCode {
			oPost["url"] = r.Request.URL.String()
		} else {
			if xx := ss.GetDomainInfo(r.Request.URL.String()); nil != xx {
				oPost = xx
			} else {
				log.Println(r.Request.URL.String())
			}
		}
		if 0 != r.StatusCode {
			oPost["id"] = r.Request.AbsoluteURL(r.Request.URL.String())
			oPost["Headers"] = r.Headers
			oPost["StatusCode"] = r.StatusCode
		}
		ss.Cache.Set(szId, oPost, cache.NoExpiration)
	}
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
		if ss.fnCbk(link, strings.TrimSpace(e.Text), e) {
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
		go ss.DoResponse2Es(r)
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
