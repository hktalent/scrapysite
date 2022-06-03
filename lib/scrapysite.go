package lib

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly/v2"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	// "github.com/gocolly/colly/v2/extensions"
	"github.com/gocolly/colly/v2/proxy"
	"github.com/gocolly/colly/v2/queue"
)

var CacheDir = DefaultBindConf.CacheDir

// 缓存，基于缓存，避免相同url重复请求，可以修改为redis
var Cache = NewKvDbOp(DefaultBindConf.DnsDbCache)

// es存储的线程控制 nThreads / 2
var EsThreas = make(chan struct{}, DefaultBindConf.ThreadNum)

// 获取ip归宿信息的线程数nThreads / 4
var getIpThread = make(chan struct{}, DefaultBindConf.ThreadNum/4)

// 定义爬虫结构
type ScrapySite struct {
	Scrapy *colly.Collector
}

// 获取缓存
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

// 设置缓存
func SetCache(k string, o interface{}) {
	b, err := json.Marshal(o)
	if nil == err {
		Cache.Put(k, b)
	}
}

// 回调决定是否继续请求url
// 链接、标题、选择器,正则，对象
func (r *ScrapySite) CallBack(link, title string, regUrl string, regTitle string, i interface{}) bool {
	szUrl := link // e.Request.URL.String()
	// 过滤
	if r.FilterHref(szUrl) {
		return false
	}
	if "" != regUrl {
		r1, err := regexp.Compile(regUrl)
		if nil == err {
			if 0 < len(r1.FindAllString(szUrl, -1)) {
				return true
			}
		}
	}
	if "" != regTitle {
		r1, err := regexp.Compile(regTitle)
		if nil == err {
			if 0 < len(r1.FindAllString(title, -1)) {
				return true
			}
		}
	}
	return false
}
func (r *ScrapySite) FilterHref(s string) bool {
	for _, x := range DefaultBindConf.FilterHrefReg {
		reg1 := regexp.MustCompile(x)
		if reg1.MatchString(s) {
			return true
		}
	}
	return false
}

// 创建爬虫对象
// 默认开启异步
// 默认MaxDepth 微0，不受深度限制
// 默认允许、开启AllowURLRevisit
// 默认 忽略了 IgnoreRobotsTxt
// default: IgnoreRobotsTxt
// 默认Timeout 30秒, KeepAlive 15秒
// 最大空闲连接数 100
// 空闲连接超时 90秒
// TLS 握手超时 10秒
// 默认不限制域名
func NewScrapySite() *ScrapySite {
	var r *ScrapySite
	r = &ScrapySite{}
	// Cache = cache.New(10000*time.Hour, 10000*time.Hour)
	r.Scrapy = colly.NewCollector(colly.CacheDir(CacheDir), colly.MaxDepth(0), colly.Async(), colly.AllowURLRevisit(), colly.IgnoreRobotsTxt())
	r.Scrapy.WithTransport(&http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   DefaultBindConf.Timeout * time.Second,   // 超时时间
			KeepAlive: DefaultBindConf.KeepAlive * time.Second, // keepAlive 超时时间
		}).DialContext,
		MaxIdleConns:          DefaultBindConf.MaxIdleConns,                      // 最大空闲连接数
		IdleConnTimeout:       DefaultBindConf.IdleConnTimeout * time.Second,     // 空闲连接超时
		TLSHandshakeTimeout:   DefaultBindConf.TLSHandshakeTimeout * time.Second, // TLS 握手超时
		ExpectContinueTimeout: DefaultBindConf.ExpectContinueTimeout * time.Second,
	})

	r.Scrapy.Limit(&colly.LimitRule{
		DomainGlob:  DefaultBindConf.DomainGlob,
		Parallelism: DefaultBindConf.ThreadNum,
		RandomDelay: DefaultBindConf.RandomDelay * time.Second})
	// extensions.RandomUserAgent(r.Scrapy)
	// extensions.Referer(r.Scrapy)
	// http://go-colly.org/docs/best_practices/multi_collector/
	// c2 := r.Scrapy.Clone()
	r.init()
	return r
}

// ips 转字符串数组
func I2S(data []interface{}) []string {
	a := []string{}
	for _, j := range data {
		a = append(a, fmt.Sprintf("%v", j))
	}
	return a
}

// 获取domain的所有ip地址
// 注意：一个域名可能解析出多个ip
func (r *ScrapySite) GetDomainIps(domain string) (aRst []string) {
	for _, x := range DefaultBindConf.IpRegs {
		re := regexp.MustCompile(x)
		aRst = []string{}
		if "" != re.FindString(domain) {
			aRst = []string{domain}
		} else {
			oRst, err := GetCache(domain)
			if err == nil && nil != oRst {
				aRst = I2S(oRst.([]interface{}))
				return
			}
			ips, err := net.LookupIP(domain)
			if nil == err {
				for _, ip := range ips {
					if ipv4 := ip.To4(); ipv4 != nil {
						aRst = append(aRst, ipv4.String())
						//go SetCache(ipv4.String(), domain)
					}
				}
				if 0 < len(aRst) {
					SetCache(domain, aRst)
				}
			}
		}
	}
	return
}

// 获取url的domain对应的ip，及ip归宿信息
func (r *ScrapySite) GetDomainInfo(url string) map[string]interface{} {
	s1 := "://"
	if -1 < strings.Index(url, s1) {
		s := strings.Split(url, s1)[1]
		re := regexp.MustCompile(`[;,\?\/:# ]`)
		s1 = re.Split(s, -1)[0]
		var oRst = make(map[string]interface{})
		aRst := r.GetDomainIps(s1)
		var xxx []map[string]interface{} = []map[string]interface{}{}
		if 0 < len(aRst) {
			for _, x := range aRst {
				xD, err := r.GetIpInfo(x)
				if nil != err {
					FnLog(err)
					continue
				}
				if nil != xD {
					xxx = append(xxx, xD)
				} else { // 没有找到，或者网络异常的情况
					xxx = append(xxx, map[string]interface{}{"query": x})
				}
			}
		}
		oRst["ips"] = xxx
		return oRst
	}
	return nil
}

// 获取ip归宿信息
func (ss *ScrapySite) GetIpInfo(ip string) (map[string]interface{}, error) {
	oRst, err := GetCache(ip)
	if nil == err {
		x1, ok := oRst.(map[string]interface{})
		if ok {
			return x1, nil
		}
	}
	getIpThread <- struct{}{}
	defer func() {
		<-getIpThread
	}()
	req, err := http.NewRequest("GET", fmt.Sprintf(DefaultBindConf.GetIpUrlFormat, ip), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "curl/11")
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

//var esCache = NewKvDbOp("EsCache")

// 指定id发送数据到ES url，，data为json转换后到数据
func (ss *ScrapySite) SendReq(data *bytes.Buffer, id string) {
	EsThreas <- struct{}{}
	defer func() {
		<-EsThreas
	}()
	//// 处理过了
	//szR, err := esCache.Get(id)
	//if nil == err && nil != szR {
	//	return
	//}
	//esCache.Put(id, []byte("1"))
	req, err := http.NewRequest("POST", DefaultBindConf.EsUrl+url.QueryEscape(id), data)
	if err != nil {
		FnLog(err)
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
		FnLog("SendReq", DefaultBindConf.EsUrl, id, err)
		return
	}
	s2, err := ioutil.ReadAll(resp.Body)
	if nil == err {
		FnLog("Elasticsearch save ok: ", id, string(s2))
	}
}

// 发送man到ES
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
// 计算md5、sha1、sha256 到map
func (ss *ScrapySite) DoNotText4Parms(body []byte, ContentType string) map[string]interface{} {
	// 非文本，计算sha1、md5
	if nil != body {
		if strings.Index(ContentType, "text/") == -1 && 0 < len(body) {
			x1 := body
			md5R, sha1R, sha256R := ss.Hash(x1)
			return map[string]interface{}{"md5": md5R, "sha1": sha1R, "sha256": sha256R}
		}
	}
	return nil
}

// 处理非文本文件
// 计算md5、sha1、sha256 到map
func (ss *ScrapySite) DoNotText(r *colly.Response) map[string]interface{} {
	return ss.DoNotText4Parms(r.Body, r.Headers.Get("Content-Type"))
}

func (ss *ScrapySite) DoResponseMap(absUrl, url, ContentType string, StatusCode int, body []byte) {
	oPost := make(map[string]interface{})

	szId := absUrl
	if "" == szId || 14 > len(szId) {
		return
	}
	oRst, err := GetCache(szId)
	if nil == err && nil != oRst {
		oPost = oRst.(map[string]interface{})
		// spew.Dump(oPost["urls"])
		// interface conversion: interface {} is []interface {}, not []string
		oPost["urls"] = ss.AddUrls(url, oPost["urls"].([]interface{}))
	} else {
		// 对于爬虫而言，非安全检测一类到，只有200有意义，如果是安全检测，非200的也非常有意义
		if 200 == StatusCode {
			if xx := ss.GetDomainInfo(url); nil != xx {
				oPost = xx
			}
		} else {
			FnLog(StatusCode, url)
		}
		oPost["urls"] = []string{url}
		if 0 != StatusCode {
			//xh := r.Headers
			//oPost["Headers"] = xh
			//oPost["StatusCode"] = r.StatusCode
			szTt, err := GetCache(url + "_title")
			if nil == err && "" != szTt.(string) {
				oPost["title"] = szTt.(string)
			}
		}
	}
	x1 := ss.DoNotText4Parms(body, ContentType)
	if nil != x1 {
		oPost["hash"] = x1
	} else if nil != body && 0 < len(body) {
		oPost["body"] = string(body)
	}

	SetCache(szId, oPost)
	go ss.SendJsonReq(oPost, szId)
}

// 处理请求响应，并转换为ES需要到结构
// id长度不能小雨14
func (ss *ScrapySite) DoResponse2Es(r *colly.Response) {
	ss.DoResponseMap(r.Request.AbsoluteURL(r.Request.URL.String()), r.Request.URL.String(), r.Headers.Get("Content-Type"), r.StatusCode, r.Body)
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
	for _, x := range DefaultBindConf.ScrapyRule {
		func(x1 ScrapyRule) {
			ss.Scrapy.OnHTML(x1.Selector, func(e *colly.HTMLElement) {
				link := e.Attr("href")
				title := strings.TrimSpace(e.Text)
				if "" != title {
					SetCache(link+"_title", title)
				}
				if ss.CallBack(link, title, x1.CbkUrlReg, x1.CbkTitleReg, e) {
					e.Request.Visit(link)
					// if !strings.HasPrefix(link, "http") {
					// } else {
					// 	// FnLog("xxx:", e.Request.AbsoluteURL("")
					// 	ss.Scrapy.Visit(link)
					// }
				}
			})
		}(x)
	}

	// Set error handler
	ss.Scrapy.OnError(func(r *colly.Response, err error) {
		ss.DoResponse2Es(r)
		fmt.Println("Request URL:", r.Request.URL, "failed with response:", "Error:", err)
	})

}

// 求hash md5,sha1,sha256,SM3: 国密hash算法库
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

// 注册处理响应数据的回调
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
		// FnLog("我的请求", r.URL.String(), r.AbsoluteURL(r.URL.String()), r.Headers.Get("User-Agent"))
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.2 Safari/605.1.15")
		onRequest(r)
	})
}

// 支持多个url，分隔符号：[;,| ]
func (ss *ScrapySite) Start() {
	storage := &Storage{
		Filename: DefaultBindConf.SqliteDbName,
	}
	defer storage.Close()
	ss.Scrapy.SetStorage(storage)
	q, _ := queue.New(
		DefaultBindConf.ThreadNum, // Number of consumer threads
		storage,
		// &queue.InMemoryQueueStorage{MaxSize: nThreads}, // Use default queue storage
	)
	for _, x := range DefaultBindConf.ScrapyRule {
		for _, j := range x.StartUrls {
			q.AddURL(j)
		}
		// ss.Scrapy.Visit(url)
	}
	q.Run(ss.Scrapy)
	ss.Scrapy.Wait()
}
